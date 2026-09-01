package initialize

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"liveclass/internal/rpc/chat/global"
	"log"
	"time"
)

func InitMongo() *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(global.Config.MongoConfig.Addr).
		SetConnectTimeout(2*time.Second).
		SetServerSelectionTimeout(2*time.Second).
		SetSocketTimeout(2*time.Second).
		SetRetryReads(true).
		SetRetryWrites(true))
	if err != nil {
		panic(err)
	}

	log.Println("init mongo success")

	return client
}
