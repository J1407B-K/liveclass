package main

import (
	"context"
	agent "liveclass/idl/kitex_gen/agent/agentservice"
	"liveclass/internal/resilience"
	agentruntime "liveclass/internal/rpc/agent/agent"
	"liveclass/internal/rpc/agent/agentmetrics"
	"liveclass/internal/rpc/agent/cdc"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/flag"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/initialize"
	"liveclass/internal/rpc/agent/mcp"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/rag"
	agentsession "liveclass/internal/rpc/agent/session"
	agent2 "liveclass/internal/rpc/agent/workflow/agent"
	"liveclass/internal/rpc/agent/workflow/fact"
	"log"
	"net"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	kServer "github.com/cloudwego/kitex/server"
	prometheus "github.com/kitex-contrib/monitor-prometheus"
	"github.com/kitex-contrib/obs-opentelemetry/provider"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	clientprometheus "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

func main() {
	initialize.SetupViper()
	if err := dependency.Configure(global.Config.Resilience); err != nil {
		log.Fatalf("configure dependency resilience: %v", err)
	}

	db := initialize.InitPGDB()
	option := flag.Parse()
	ok := flag.DBOption(db, option)
	if !ok {
		log.Println("未自动建表")
	}

	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer initCancel()
	qdrantCli, err := initialize.InitQdrant(initCtx, 2048)
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := mcp.StartMCPServer(); err != nil {
			log.Fatalf("start mcp server failed: %v", err)
		}
	}()

	global.ChatModel, err = initialize.InitChatModel(initCtx)
	if err != nil {
		panic(err.Error())
	}

	global.MultiModalEmbedder, err = initialize.InitMultiModalEmbedder(initCtx)
	if err != nil {
		panic(err.Error())
	}

	global.AgentRunner, err = agent2.BuildAgent(initCtx, dbm)
	if err != nil {
		panic(err.Error())
	}

	global.FactExtractorRunner, err = fact.BuildFactExtractor(initCtx)
	if err != nil {
		panic(err.Error())
	}

	runtimeCfg := global.Config.AgentRuntime
	sessionBudget := agentsession.Budget{
		ModelContext: runtimeCfg.ModelContextTokens, SystemReserve: runtimeCfg.SystemReserveTokens,
		OutputReserve: runtimeCfg.OutputReserveTokens, RAG: runtimeCfg.RAGBudgetTokens,
		Memory: runtimeCfg.MemoryBudgetTokens, Conversation: runtimeCfg.ConversationBudgetTokens,
		CompactionTrigger: runtimeCfg.CompactionTriggerTokens, RecentTail: runtimeCfg.RecentTailTokens,
		MaxToolResult: runtimeCfg.MaxToolResultTokens,
	}
	sessionManager := agentsession.NewManager(dbm, agentsession.NewBuilder(sessionBudget), &agentsession.ModelCompactor{
		Model: global.ChatModel, RepairAttempts: 2,
		Fallback: agentsession.DeterministicCompactor{MaxTokens: runtimeCfg.ConversationBudgetTokens / 3},
	})

	docMgr, err := initialize.InitDocQdrant(initCtx, 2048)
	if err != nil {
		panic(err.Error())
	}
	var docES *rag.ElasticsearchManager
	if mgr, esErr := initialize.InitDocElasticsearch(initCtx); esErr != nil {
		log.Printf("init doc elasticsearch failed, fallback to vector-only retrieval: %v", esErr)
	} else {
		docES = mgr
	}
	docRetriever, err := rag.NewDocRetriever(docMgr, docES)
	if err != nil {
		panic(err.Error())
	}
	agentRuntime := agentruntime.NewAgentRuntime(dbm, global.AgentRunner, global.FactExtractorRunner, docRetriever, global.MultiModalEmbedder, sessionManager)

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
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = p.Shutdown(shutdownCtx)
	}()

	r, err := etcd.NewEtcdRegistry([]string{global.Config.EtcdAddr})
	if err != nil {
		log.Fatalf("Failed to create etcd registry: %v", err)
		return
	}

	addr, _ := net.ResolveTCPAddr("tcp", global.Config.ServiceAddr)
	registry := clientprometheus.NewRegistry()
	registry.MustRegister(collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	registry.MustRegister(resilience.Collectors()...)
	registry.MustRegister(agentmetrics.Collectors()...)
	svr := agent.NewServer(&AgentServiceImpl{
		DBManager:    dbm,
		agentRuntime: agentRuntime,
	},
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "agentservice"}),
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		kServer.WithTracer(prometheus.NewServerTracer(global.Config.PrometheusPort, "/metrics", prometheus.WithRegistry(registry))))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
