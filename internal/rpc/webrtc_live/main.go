package main

import (
	"context"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/rpc/webrtc_live/dao"
	"liveclass/internal/rpc/webrtc_live/flag"
	"liveclass/internal/rpc/webrtc_live/initialize"
	"log"
	"net"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	kServer "github.com/cloudwego/kitex/server"
	prometheus "github.com/kitex-contrib/monitor-prometheus"
	"github.com/kitex-contrib/obs-opentelemetry/provider"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
)

func main() {
	initialize.SetupViper()
	db := initialize.InitGormDB()
	option := flag.Parse()
	ok := flag.DBOption(db, option)
	if !ok {
		log.Println("未自动建表")
	}

	rdb := initialize.InitRedisDB()
	err := initialize.InitBloom(db, rdb)
	if err != nil {
		log.Fatal(err)
	}
	node, err := initialize.InitSnowflake()
	if err != nil {
		log.Fatal(err)
	}

	changesha, delsha, selectsha := initialize.InitScript(rdb)
	initialize.InitWebRTCEngine()
	cosClient := initialize.SetUpCos()

	ctxBloom, cancelBloom := context.WithCancel(context.Background())
	go initialize.InitRuntimeAddMBloom(ctxBloom, db, rdb)
	defer cancelBloom()

	userCli, err := NewUserClient()
	if err != nil {
		log.Fatal(err)
	}

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("webrtc_liveservice"),
		provider.WithExportEndpoint(global.Config.JaegerEndpoint),
		provider.WithInsecure(),
		provider.WithEnableMetrics(false),
	)
	defer p.Shutdown(context.Background())

	r, err := etcd.NewEtcdRegistry([]string{global.Config.EtcdAddr})
	if err != nil {
		log.Fatalf("Failed to create etcd registry: %v", err)
		return
	}

	addr, _ := net.ResolveTCPAddr("tcp", global.Config.ServiceAddr)
	svr := webrtc_live.NewServer(&WebrtcLiveImpl{DBManager: &dao.DBManager{DB: db, RDB: rdb, Node: node}, userCli: userCli, changesha: changesha, selectsha: selectsha, delsha: delsha, cosClient: cosClient},
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "webrtc_liveservice"}),
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "webrtc_liveservice",
		}),
		kServer.WithTracer(prometheus.NewServerTracer(global.Config.PrometheusPort, "/metrics")))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
