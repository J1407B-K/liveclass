package main

import (
	"context"
	agent "liveclass/idl/kitex_gen/agent/agentservice"
	"liveclass/internal/rpc/agent/cdc"
	"liveclass/internal/rpc/agent/flag"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/initialize"
	"liveclass/internal/rpc/agent/mcp"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/rag"
	agent2 "liveclass/internal/rpc/agent/workflow/agent"
	"liveclass/internal/rpc/agent/workflow/fact"
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

	db := initialize.InitPGDB()
	option := flag.Parse()
	ok := flag.DBOption(db, option)
	if !ok {
		log.Println("未自动建表")
	}

	qdrantCli, err := initialize.InitQdrant(context.Background(), 2048)
	if err != nil {
		log.Println("qdrant init err:", err)
		return
	}

	dbm := &memory.DBManager{
		DB:        db,
		QdrantCli: &memory.QdrantManager{Client: qdrantCli.Client, Collection: qdrantCli.Collection},
	}

	//err = gotenv.Load("coze.env")
	//if err != nil {
	//	panic(err.Error())
	//}

	ctx := context.Background()

	go func() {
		if err := mcp.StartMCPServer(); err != nil {
			log.Fatalf("start mcp server failed: %v", err)
		}
	}()

	global.ChatModel, err = initialize.InitChatModel(ctx)
	if err != nil {
		panic(err.Error())
	}

	global.MultiModalEmbedder, err = initialize.InitMultiModalEmbedder(ctx)
	if err != nil {
		panic(err.Error())
	}

	global.AgentRunner, err = agent2.BuildAgent(ctx)
	if err != nil {
		panic(err.Error())
	}

	global.FactExtractorRunner, err = fact.BuildFactExtractor(ctx)
	if err != nil {
		panic(err.Error())
	}

	docMgr, err := initialize.InitDocQdrant(ctx, 2048)
	if err != nil {
		panic(err.Error())
	}
	docRetriever, err := rag.NewDocRetriever(docMgr)
	if err != nil {
		panic(err.Error())
	}

	if cli, err := initialize.InitUserClient(); err != nil {
		log.Printf("init user client failed: %v", err)
	} else {
		global.UserClient = cli
	}

	if cli, err := initialize.InitLessonClient(); err != nil {
		log.Printf("init lesson client failed: %v", err)
	} else {
		global.LessonClient = cli
	}

	reader := initialize.InitKafkaReader()
	go func() {
		if err := cdc.RunFactIndexerConsumer(ctx, reader, dbm); err != nil {
			log.Printf("fact indexer worker exited: %v", err)
		}
	}()

	writer := initialize.InitKafkaWriter()
	defer writer.Close()
	go func() {
		if err := cdc.RunOutboxDispatcher(ctx, dbm, writer); err != nil {
			log.Printf("outbox dispatcher exited: %v", err)
		}
	}()

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("agentservice"),
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
	svr := agent.NewServer(&AgentServiceImpl{
		DBManager:    dbm,
		agentRunner:  global.AgentRunner,
		factRunner:   global.FactExtractorRunner,
		docRetriever: docRetriever,
		embedder:     global.MultiModalEmbedder,
	},
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "agentservice"}),
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "agentservice",
		}),
		kServer.WithTracer(prometheus.NewServerTracer(global.Config.PrometheusPort, "/metrics")))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
