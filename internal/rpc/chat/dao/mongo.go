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
			Keys: bson.D{
				{Key: "lesson_id", Value: 1},
				{Key: "created_at", Value: -1},
				{Key: "message_id", Value: -1},
			},
			Options: options.Index().SetName("lesson_created_message"),
		},
	})
	return err
}

func InsertMongo(ctx context.Context, collection *mongo.Collection, message model.Message) error {
	_, err := collection.InsertOne(ctx, message)
	return err
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
