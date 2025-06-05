package main

import (
	"context"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	kServer "github.com/cloudwego/kitex/server"
	prometheus "github.com/kitex-contrib/monitor-prometheus"
	"github.com/kitex-contrib/obs-opentelemetry/provider"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
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
	cosClient := initialize.SetUpCos()
	countsha, membersha, delsha, selectsha := initialize.InitScript(rdb)

	option := flag.Parse()
	ok := flag.DBOption(db, option)
	if !ok {
		log.Println("未自动建表")
	}

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("liveservice"),
		provider.WithExportEndpoint("localhost:4317"),
		provider.WithInsecure(),
	)
	defer p.Shutdown(context.Background())

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	userCli, err := NewUserClient()
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9002")
	svr := live.NewServer(&LiveServiceImpl{DB: db, RDB: rdb, userCli: userCli, GetLiveKeyAddr: getLiveKeyAddr, countsha: countsha, membersha: membersha, delsha: delsha, selectsha: selectsha, cosClient: cosClient},
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "liveservice"}),
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "liveservice",
		}),
		kServer.WithTracer(prometheus.NewServerTracer(":10003", "/metrics")))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
