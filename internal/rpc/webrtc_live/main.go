package main

import (
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	etcd "github.com/kitex-contrib/registry-etcd"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/rpc/webrtc_live/initialize"
	"log"
	"net"
)

func main() {
	initialize.SetupViper()
	db := initialize.InitGormDB()
	initialize.InitWebRTCEngine()

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9005")
	svr := webrtc_live.NewServer(&WebrtcLiveImpl{DB: db},
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "webrtc_liveservice",
		}))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
