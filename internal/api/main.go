package main

import (
	"context"
	"liveclass/internal/api/global"
	"liveclass/internal/api/initialize"
	"liveclass/internal/api/model"
	"liveclass/internal/api/router"
	"liveclass/internal/api/service"
	"log"

	etcd "github.com/kitex-contrib/registry-etcd"
)

func main() {
	initialize.SetupViper()
	rdb := initialize.InitRedisDB()
	resolver, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	global.Resolver = &resolver
	global.DBManager = &model.DBManager{
		RDB: rdb,
	}
	snowflake, err := initialize.InitSnowflake()
	if err != nil {
		panic(err)
	}
	global.Node = snowflake

	err = initialize.InitNewClient()
	if err != nil {
		panic(err)
	}

	chatReader := initialize.InitChatKafkaReader("chat-api-instance-1")

	go func() {
		if err := service.RunChatConsumer(context.Background(), chatReader); err != nil {
			log.Printf("chat consumer stopped: %v", err)
		}
	}()
	router.InitRouter()
}
