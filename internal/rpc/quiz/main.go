package main

import (
	"context"
	quiz "liveclass/idl/kitex_gen/quiz/quizservice"
	"liveclass/internal/rpc/quiz/dao"
	"liveclass/internal/rpc/quiz/flag"
	"liveclass/internal/rpc/quiz/initialize"
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

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("quizservice"),
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

	webrtcCli, err := NewWebRTCLiveClient()
	if err != nil {
		log.Fatalf("Failed to create WebRTC client: %v", err)
		return
	}
	userCli, err := NewUserClient()
	if err != nil {
		log.Fatalf("Failed to create User client: %v", err)
		return
	}

	addr, _ := net.ResolveTCPAddr("tcp", global.Config.ServiceAddr)
	svr := quiz.NewServer(&QuizServiceImpl{DBManager: &dao.DBManager{DB: db, RDB: rdb}, webrtcCli: webrtcCli, userCli: userCli},
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "quizservice"}),
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "quizservice",
		}),
		kServer.WithTracer(prometheus.NewServerTracer(global.Config.PrometheusPort, "/metrics")))

	err = svr.Run()
	if err != nil {
		log.Println(err.Error())
	}
}
