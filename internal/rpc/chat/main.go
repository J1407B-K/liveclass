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

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("chatservice"),
		provider.WithExportEndpoint("localhost:4317"),
		provider.WithInsecure(),
	)
	defer p.Shutdown(context.Background())

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9004")

	svr := chat.NewServer(&ChatServiceImpl{mongoClient: client, liveCli: liveCli},
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "chatservice"}),
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "chatservice",
		}),
		kServer.WithTracer(prometheus.NewServerTracer(":10005", "/metrics")))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
