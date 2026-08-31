package main

import (
	"context"
	"liveclass/internal/api/chatroom"
	"liveclass/internal/api/global"
	"liveclass/internal/api/initialize"
	"liveclass/internal/api/model"
	"liveclass/internal/api/router"
	"liveclass/internal/api/service"
	"log"
	"os"
	"sync"

	etcd "github.com/kitex-contrib/registry-etcd"
)

func main() {
	initialize.SetupViper()
	chatRooms, err := chatroom.NewManager(chatroom.Config{
		SendQueueSize:  global.Config.ChatWebSocket.SendQueueSize,
		WriteWait:      global.Config.ChatWebSocket.WriteWait,
		PongWait:       global.Config.ChatWebSocket.PongWait,
		PingPeriod:     global.Config.ChatWebSocket.PingPeriod,
		MaxMessageSize: global.Config.ChatWebSocket.MaxMessageSize,
	})
	if err != nil {
		log.Fatal(err)
	}
	global.ChatRooms = chatRooms
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

	backgroundCtx, cancelBackground := context.WithCancel(context.Background())
	var background sync.WaitGroup
	background.Add(1)
	go func() {
		defer background.Done()
		hostname, _ := os.Hostname()
		if err := service.RunChatConsumer(backgroundCtx, func() service.ChatReader {
			return initialize.InitChatKafkaReader("chat-api-" + hostname)
		}); err != nil {
			log.Printf("chat kafka consumer stopped: %v", err)
		}
	}()

	background.Add(1)
	go func() {
		defer background.Done()
		if err := service.RunQuizRedisSubscriber(backgroundCtx, rdb); err != nil && backgroundCtx.Err() == nil {
			log.Printf("quiz redis subscriber stopped: %v", err)
		}
	}()

	router.InitRouter()
	cancelBackground()
	background.Wait()
}
