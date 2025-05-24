package dao

import (
	"context"
	"encoding/json"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"liveclass/internal/rpc/chat/global"
)

func ChooseCollection(suffix string, client *mongo.Client) *mongo.Collection {
	return client.Database(global.Config.Database).Collection(global.Config.CollectionPrefix + suffix)
}

func InsertMongo(ctx context.Context, coll *mongo.Collection, msg interface{}) error {
	_, err := coll.InsertOne(ctx, msg)
	return err
}

func SelectMongo(ctx context.Context, coll *mongo.Collection) (string, error) {
	cursor, err := coll.Find(ctx, bson.M{})
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

	// []bson.M -> []byte
	b, err := json.Marshal(results)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func DropCollection(ctx context.Context, coll *mongo.Collection) error {
	return coll.Drop(ctx)
}
