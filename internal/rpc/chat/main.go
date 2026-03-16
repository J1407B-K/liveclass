package main

import (
	"context"
	chat "liveclass/idl/kitex_gen/chat/chatservice"
	"liveclass/internal/rpc/chat/global"
	"liveclass/internal/rpc/chat/initialize"
	"liveclass/internal/rpc/chat/kafka"
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
	client := initialize.InitMongo()

	global.Writer = initialize.InitKafkaWriter()
	defer kafka.CloseKafkaWriter()

	ctx := context.Background()

	groupID := global.Config.KafkaGroup
	if groupID == "" {
		groupID = "chat-mongo-writer"
	}
	reader := initialize.InitKafkaReader(groupID)
	go func() {
		if err := kafka.RunMongoWriter(ctx, reader, client); err != nil {
			log.Printf("chat mongo writer exited: %v", err)
		}
	}()

	webrtcliveCli, err := NewWebRTCLiveClient()
	if err != nil {
		log.Fatal(err)
	}

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("chatservice"),
		provider.WithExportEndpoint("localhost:4317"),
		provider.WithInsecure(),
		provider.WithEnableMetrics(false),
	)
	defer p.Shutdown(context.Background())

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9004")

	svr := chat.NewServer(&ChatServiceImpl{mongoClient: client, webrtcCli: webrtcliveCli},
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
