package initialize

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"liveclass/internal/rpc/chat/global"
	"log"
)

func InitMongo() *mongo.Client {
	ctx := context.Background()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(global.Config.MongoConfig.Addr))
	if err != nil {
		panic(err)
	}

	log.Println("init mongo success")

	return client
}
