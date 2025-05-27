package main

import (
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	etcd "github.com/kitex-contrib/registry-etcd"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/rpc/webrtc_live/flag"
	"liveclass/internal/rpc/webrtc_live/initialize"
	"log"
	"net"
)

func main() {
	initialize.SetupViper()
	db := initialize.InitGormDB()
	rdb := initialize.InitRedisDB()
	countsha, membersha, delsha, selectsha := initialize.InitScript(rdb)
	initialize.InitWebRTCEngine()
	cosClient := initialize.SetUpCos()

	option := flag.Parse()
	ok := flag.DBOption(db, option)
	if !ok {
		log.Println("未自动建表")
	}

	userCli, err := NewUserClient()
	if err != nil {
		log.Fatal(err)
	}

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9005")
	svr := webrtc_live.NewServer(&WebrtcLiveImpl{DB: db, userCli: userCli, RDB: rdb, countsha: countsha, membersha: membersha, selectsha: selectsha, delsha: delsha, cosClient: cosClient},
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
