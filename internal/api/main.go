package main

import (
	"context"
	"liveclass/internal/api/global"
	"liveclass/internal/api/initialize"
	"liveclass/internal/api/model"
	"liveclass/internal/api/router"
	"liveclass/internal/api/service"
	"log"
	"os"

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

	go func() {
		hostname, _ := os.Hostname()
		reader := initialize.InitChatKafkaReader("chat-api-" + hostname)
		if err := service.RunChatConsumer(context.Background(), reader); err != nil {
			log.Printf("chat kafka consumer stopped: %v", err)
		}
	}()

	go func() {
		if err := service.RunQuizRedisSubscriber(context.Background(), rdb); err != nil {
			log.Printf("quiz redis subscriber stopped: %v", err)
		}
	}()

	router.InitRouter()
}
