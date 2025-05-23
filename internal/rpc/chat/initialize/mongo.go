package initialize

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"liveclass/internal/rpc/chat/global"
	"log"
)

func InitMongo() (*mongo.Client, *mongo.Collection) {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(global.Config.MongoConfig.Addr))
	if err != nil {
		panic(err)
	}

	coll := client.Database(global.Config.Database).Collection(global.Config.Collection)

	log.Println("init mongo success")

	return client, coll
}
