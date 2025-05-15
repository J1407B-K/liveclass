package main

import (
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	etcd "github.com/kitex-contrib/registry-etcd"
	live "liveclass/idl/kitex_gen/live/liveservice"
	"liveclass/internal/rpc/live/flag"
	"liveclass/internal/rpc/live/initialize"
	"log"
	"net"
)

func main() {
	initialize.SetupViper()
	db := initialize.InitGormDB()
	rdb := initialize.InitRedisDB()
	getLiveKeyAddr := initialize.InitKeyAddr()
	countsha, membersha, delsha, selectsha := initialize.InitScript(rdb)

	option := flag.Parse()
	ok := flag.DBOption(db, option)
	if !ok {
		log.Println("未自动建表")
	}

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	userCli, err := NewUserClient()
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9002")
	svr := live.NewServer(&LiveServiceImpl{DB: db, RDB: rdb, userCli: userCli, GetLiveKeyAddr: getLiveKeyAddr, countsha: countsha, membersha: membersha, delsha: delsha, selectsha: selectsha},
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "liveservice",
		}))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
