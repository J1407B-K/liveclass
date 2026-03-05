package main

import (
	"liveclass/internal/api/global"
	"liveclass/internal/api/initialize"
	"liveclass/internal/api/model"
	"liveclass/internal/api/router"
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

	err = initialize.InitNewClient()
	if err != nil {
		panic(err)
	}

	go initialize.ConsumeKafkaMessages()
	router.InitRouter()
}
