package main

import (
	"context"
	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	kServer "github.com/cloudwego/kitex/server"
	"github.com/coze-dev/cozeloop-go"
	prometheus "github.com/kitex-contrib/monitor-prometheus"
	"github.com/kitex-contrib/obs-opentelemetry/provider"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	"github.com/subosito/gotenv"
	agent "liveclass/idl/kitex_gen/agent/agentservice"
	"liveclass/internal/rpc/agent/mcp"
	"log"
	"net"
)

func main() {
	err := gotenv.Load("coze.env")
	if err != nil {
		panic(err.Error())
	}

	go mcp.StartMCPServer()

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("agentservice"),
		provider.WithExportEndpoint("localhost:4317"),
		provider.WithInsecure(),
		provider.WithEnableMetrics(false),
	)
	defer p.Shutdown(context.Background())

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	cozeloopClient, _ := cozeloop.NewClient(cozeloop.WithPromptTrace(true))

	userCli, err := NewUserClient()
	if err != nil {
		log.Fatal(err)
	}
	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9006")
	svr := agent.NewServer(&AgentServiceImpl{userCli: userCli, cozeloopClient: cozeloopClient},
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "agentservice"}),
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "agentservice",
		}),
		kServer.WithTracer(prometheus.NewServerTracer(":10007", "/metrics")))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
