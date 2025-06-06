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
	quiz "liveclass/idl/kitex_gen/quiz/quizservice"
	"liveclass/internal/rpc/quiz/flag"
	"liveclass/internal/rpc/quiz/initialize"
	"log"
	"net"
)

func main() {
	initialize.SetupViper()
	db := initialize.InitGormDB()

	option := flag.Parse()
	ok := flag.DBOption(db, option)
	if !ok {
		log.Println("未自动建表")
	}

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("quizservice"),
		provider.WithExportEndpoint("localhost:4317"),
		provider.WithInsecure(),
		provider.WithEnableMetrics(false),
	)
	defer p.Shutdown(context.Background())

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	liveCli, err := NewLiveClient()
	if err != nil {
		log.Fatal(err)
	}
	userCli, err := NewUserClient()
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9003")
	svr := quiz.NewServer(&QuizServiceImpl{DB: db, liveCli: liveCli, userCli: userCli},
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "quizservice"}),
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "quizservice",
		}),
		kServer.WithTracer(prometheus.NewServerTracer(":10004", "/metrics")))

	err = svr.Run()
	if err != nil {
		log.Println(err.Error())
	}
}
