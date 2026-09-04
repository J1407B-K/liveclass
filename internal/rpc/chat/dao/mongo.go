package dao

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"liveclass/internal/rpc/chat/global"
	"liveclass/internal/rpc/chat/model"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	DefaultHistoryLimit int32 = 50
	MaxHistoryLimit     int32 = 100
)

type historyCursor struct {
	CreatedAt time.Time `json:"created_at"`
	MessageID string    `json:"message_id"`
}

func MessagesCollection(client *mongo.Client) *mongo.Collection {
	return client.Database(global.Config.Database).Collection(global.Config.MessagesCollection)
}

func EnsureMessageIndexes(ctx context.Context, client *mongo.Client) error {
	collection := MessagesCollection(client)
	_, err := collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "message_id", Value: 1}},
			Options: options.Index().SetName("uniq_message_id").SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "sender_id", Value: 1}, {Key: "client_message_id", Value: 1}},
			Options: options.Index().SetName("uniq_sender_client_message").SetUnique(true).
				SetPartialFilterExpression(bson.D{{Key: "client_message_id", Value: bson.D{{Key: "$type", Value: "string"}}}}),
		},
		{
			Keys: bson.D{
				{Key: "lesson_id", Value: 1},
				{Key: "created_at", Value: -1},
				{Key: "message_id", Value: -1},
			},
			Options: options.Index().SetName("lesson_created_message"),
		},
		{
			Keys: bson.D{
				{Key: "outbox.status", Value: 1},
				{Key: "outbox.next_attempt_at", Value: 1},
				{Key: "created_at", Value: 1},
			},
			Options: options.Index().SetName("outbox_pending_created"),
		},
		{
			Keys: bson.D{
				{Key: "outbox.status", Value: 1},
				{Key: "outbox.lease_until", Value: 1},
				{Key: "created_at", Value: 1},
			},
			Options: options.Index().SetName("outbox_lease_created"),
		},
	})
	return err
}

func ClaimNextOutbox(ctx context.Context, collection *mongo.Collection, owner string, now time.Time, lease time.Duration) (model.Message, error) {
	leaseUntil := now.Add(lease)
	filter := bson.D{{Key: "$or", Value: bson.A{
		bson.D{
			{Key: "outbox.status", Value: model.OutboxPending},
			{Key: "outbox.next_attempt_at", Value: bson.D{{Key: "$lte", Value: now}}},
		},
		bson.D{
			{Key: "outbox.status", Value: model.OutboxPublishing},
			{Key: "outbox.lease_until", Value: bson.D{{Key: "$lte", Value: now}}},
		},
	}}}
	update := bson.D{{Key: "$set", Value: bson.D{
		{Key: "outbox.status", Value: model.OutboxPublishing},
		{Key: "outbox.lease_owner", Value: owner},
		{Key: "outbox.lease_until", Value: leaseUntil},
	}}}
	var message model.Message
	err := collection.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().
		SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "message_id", Value: 1}}).
		SetReturnDocument(options.After)).Decode(&message)
	return message, err
}

func MarkOutboxPublished(ctx context.Context, collection *mongo.Collection, messageID, owner string, publishedAt time.Time) error {
	result, err := collection.UpdateOne(ctx,
		bson.D{{Key: "message_id", Value: messageID}, {Key: "outbox.status", Value: model.OutboxPublishing}, {Key: "outbox.lease_owner", Value: owner}},
		bson.D{
			{Key: "$set", Value: bson.D{{Key: "outbox.status", Value: model.OutboxPublished}, {Key: "outbox.published_at", Value: publishedAt}}},
			{Key: "$unset", Value: bson.D{{Key: "outbox.lease_owner", Value: ""}, {Key: "outbox.lease_until", Value: ""}, {Key: "outbox.next_attempt_at", Value: ""}, {Key: "outbox.last_error", Value: ""}}},
		})
	if err != nil {
		return err
	}
	if result.MatchedCount != 1 {
		return errors.New("outbox publish lease lost")
	}
	return nil
}

func MarkOutboxRetry(ctx context.Context, collection *mongo.Collection, messageID, owner, lastError string, nextAttempt time.Time) error {
	result, err := collection.UpdateOne(ctx,
		bson.D{{Key: "message_id", Value: messageID}, {Key: "outbox.status", Value: model.OutboxPublishing}, {Key: "outbox.lease_owner", Value: owner}},
		bson.D{
			{Key: "$set", Value: bson.D{{Key: "outbox.status", Value: model.OutboxPending}, {Key: "outbox.next_attempt_at", Value: nextAttempt}, {Key: "outbox.last_error", Value: lastError}}},
			{Key: "$inc", Value: bson.D{{Key: "outbox.attempts", Value: 1}}},
			{Key: "$unset", Value: bson.D{{Key: "outbox.lease_owner", Value: ""}, {Key: "outbox.lease_until", Value: ""}}},
		})
	if err != nil {
		return err
	}
	if result.MatchedCount != 1 {
		return errors.New("outbox retry lease lost")
	}
	return nil
}

func CountPendingOutbox(ctx context.Context, collection *mongo.Collection) (int64, error) {
	return collection.CountDocuments(ctx, bson.D{{Key: "outbox.status", Value: bson.D{{Key: "$in", Value: bson.A{model.OutboxPending, model.OutboxPublishing}}}}})
}

func InsertMongo(ctx context.Context, collection *mongo.Collection, message model.Message) error {
	_, err := collection.InsertOne(ctx, message)
	return err
}

// InsertMessageIdempotent persists a chat message once for a sender-provided
// client_message_id. It returns the original document on a retry. Empty IDs
// retain backwards-compatible at-least-once insertion behavior.
func InsertMessageIdempotent(ctx context.Context, collection *mongo.Collection, message model.Message) (model.Message, bool, error) {
	if message.ClientMessageID == "" {
		if err := InsertMongo(ctx, collection, message); err != nil {
			return model.Message{}, false, err
		}
		return message, true, nil
	}

	filter := bson.D{{Key: "sender_id", Value: message.SenderID}, {Key: "client_message_id", Value: message.ClientMessageID}}
	update := bson.D{{Key: "$setOnInsert", Value: message}}
	after := options.After
	var persisted model.Message
	err := collection.FindOneAndUpdate(ctx, filter, update, options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(after)).Decode(&persisted)
	if mongo.IsDuplicateKeyError(err) {
		err = collection.FindOne(ctx, filter).Decode(&persisted)
	}
	if err != nil {
		return model.Message{}, false, err
	}
	if persisted.LessonID != message.LessonID || persisted.Content != message.Content {
		return model.Message{}, false, errors.New("client_message_id already used with different message content")
	}
	return persisted, persisted.MessageID == message.MessageID, nil
}

func SelectMongo(
	ctx context.Context,
	collection *mongo.Collection,
	lessonID int64,
	cursor string,
	limit int32,
) ([]model.Message, string, error) {
	if limit <= 0 {
		limit = DefaultHistoryLimit
	}
	if limit > MaxHistoryLimit {
		limit = MaxHistoryLimit
	}

	filter := bson.D{{Key: "lesson_id", Value: lessonID}}
	if cursor != "" {
		decoded, err := decodeCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		filter = append(filter, bson.E{Key: "$or", Value: bson.A{
			bson.D{{Key: "created_at", Value: bson.D{{Key: "$lt", Value: decoded.CreatedAt}}}},
			bson.D{
				{Key: "created_at", Value: decoded.CreatedAt},
				{Key: "message_id", Value: bson.D{{Key: "$lt", Value: decoded.MessageID}}},
			},
		}})
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "message_id", Value: -1}}).
		SetLimit(int64(limit) + 1)
	cursorResult, err := collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, "", err
	}
	defer cursorResult.Close(ctx)

	var messages []model.Message
	if err := cursorResult.All(ctx, &messages); err != nil {
		return nil, "", err
	}
	if len(messages) <= int(limit) {
		return messages, "", nil
	}
	messages = messages[:limit]
	last := messages[len(messages)-1]
	next, err := encodeCursor(historyCursor{CreatedAt: last.CreatedAt, MessageID: last.MessageID})
	if err != nil {
		return nil, "", err
	}
	return messages, next, nil
}

// SelectMongoAfter returns messages newer than a previously observed message
// in ascending delivery order. The anchor lookup prevents a client from using
// a timestamp of its choice to bypass the opaque server cursor contract.
func SelectMongoAfter(
	ctx context.Context,
	collection *mongo.Collection,
	lessonID int64,
	afterMessageID string,
	limit int32,
) ([]model.Message, bool, error) {
	if afterMessageID == "" {
		return nil, false, errors.New("after_message_id is required")
	}
	if limit <= 0 {
		limit = DefaultHistoryLimit
	}
	if limit > MaxHistoryLimit {
		limit = MaxHistoryLimit
	}

	var anchor model.Message
	err := collection.FindOne(ctx,
		bson.D{{Key: "lesson_id", Value: lessonID}, {Key: "message_id", Value: afterMessageID}},
		options.FindOne().SetProjection(bson.D{{Key: "created_at", Value: 1}, {Key: "message_id", Value: 1}}),
	).Decode(&anchor)
	if err != nil {
		return nil, false, err
	}

	filter := bson.D{
		{Key: "lesson_id", Value: lessonID},
		{Key: "$or", Value: bson.A{
			bson.D{{Key: "created_at", Value: bson.D{{Key: "$gt", Value: anchor.CreatedAt}}}},
			bson.D{{Key: "created_at", Value: anchor.CreatedAt}, {Key: "message_id", Value: bson.D{{Key: "$gt", Value: anchor.MessageID}}}},
		}},
	}
	cursor, err := collection.Find(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "created_at", Value: 1}, {Key: "message_id", Value: 1}}).
		SetLimit(int64(limit)+1))
	if err != nil {
		return nil, false, err
	}
	defer cursor.Close(ctx)
	var messages []model.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, false, err
	}
	hasMore := len(messages) > int(limit)
	if hasMore {
		messages = messages[:limit]
	}
	return messages, hasMore, nil
}

func DeleteLessonMessages(ctx context.Context, collection *mongo.Collection, lessonID int64) error {
	_, err := collection.DeleteMany(ctx, bson.D{{Key: "lesson_id", Value: lessonID}})
	return err
}

func encodeCursor(cursor historyCursor) (string, error) {
	if cursor.CreatedAt.IsZero() || cursor.MessageID == "" {
		return "", errors.New("invalid empty history cursor")
	}
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeCursor(encoded string) (historyCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return historyCursor{}, errors.New("invalid history cursor encoding")
	}
	var cursor historyCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return historyCursor{}, errors.New("invalid history cursor payload")
	}
	if cursor.CreatedAt.IsZero() || cursor.MessageID == "" {
		return historyCursor{}, errors.New("invalid history cursor fields")
	}
	return cursor, nil
}
