package dao

import (
	"context"
	"encoding/json"
	"liveclass/internal/rpc/chat/global"
	"strconv"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ChooseCollection(LessonID int64, client *mongo.Client) *mongo.Collection {
	return client.Database(global.Config.Database).Collection(global.Config.CollectionPrefix + strconv.Itoa(int(LessonID)))
}

func InsertMongo(ctx context.Context, coll *mongo.Collection, msg interface{}) error {
	_, err := coll.InsertOne(ctx, msg)
	return err
}

func SelectMongo(ctx context.Context, coll *mongo.Collection) (string, error) {
	const maxHistoryLimit = 100

	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetLimit(maxHistoryLimit)

	cursor, err := coll.Find(ctx, bson.M{}, opts)
	if err != nil {
		return "", err
	}
	defer cursor.Close(ctx)

	var results []bson.M
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			return "", err
		}
		results = append(results, doc)
	}
	if err := cursor.Err(); err != nil {
		return "", err
	}

	b, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DropCollection(ctx context.Context, coll *mongo.Collection) error {
	return coll.Drop(ctx)
}
