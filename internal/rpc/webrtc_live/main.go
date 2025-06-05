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

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("webrtc_liveservice"),
		provider.WithExportEndpoint("localhost:4317"),
		provider.WithInsecure(),
	)
	defer p.Shutdown(context.Background())

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9005")
	svr := webrtc_live.NewServer(&WebrtcLiveImpl{DB: db, userCli: userCli, RDB: rdb, countsha: countsha, membersha: membersha, selectsha: selectsha, delsha: delsha, cosClient: cosClient},
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "webrtc_liveservice"}),
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "webrtc_liveservice",
		}),
		kServer.WithTracer(prometheus.NewServerTracer(":10006", "/metrics")))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
