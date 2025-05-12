package main

import (
	etcd "github.com/kitex-contrib/registry-etcd"
	"liveclass/internal/api"
	"liveclass/internal/global"
	"liveclass/internal/initialize"
	"log"
)

func main() {
	resolver, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	global.Resolver = &resolver

	err = initialize.InitNewClient()
	if err != nil {
		panic(err)
	}

	api.InitRouter()
}
