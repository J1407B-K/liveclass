package main

import (
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	etcd "github.com/kitex-contrib/registry-etcd"
	chat "liveclass/idl/kitex_gen/chat/chatservice"
	"liveclass/internal/rpc/chat/initialize"
	"log"
	"net"
)

func main() {
	initialize.SetupViper()
	client := initialize.InitMongo()

	liveCli, err := NewLiveClient()
	if err != nil {
		log.Fatal(err)
	}

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9004")

	svr := chat.NewServer(&ChatServiceImpl{mongoClient: client, liveCli: liveCli},
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "chatservice",
		}))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
