package main

import (
	"context"
	user "liveclass/idl/kitex_gen/user/userservice"
	"liveclass/internal/rpc/user/cdc"
	"liveclass/internal/rpc/user/dao"
	"liveclass/internal/rpc/user/flag"
	"liveclass/internal/rpc/user/initialize"
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
	snowflakeNode, err := initialize.InitSnowflake()
	if err != nil {
		panic(err)
	}
	db := initialize.InitGormDB()
	option := flag.Parse()
	ok := flag.DBOption(db, option)
	if !ok {
		log.Println("未自动建表")
	}

	rdb := initialize.InitRedisDB()
	err = initialize.InitBloom(db, rdb)
	if err != nil {
		log.Fatal(err)
	}

	reader := initialize.InitKafkaReader()

	ctxBloom, cancelBloom := context.WithCancel(context.Background())
	go initialize.InitRuntimeAddMBloom(ctxBloom, db, rdb)
	defer cancelBloom()

	ctxCDC, cancelCDC := context.WithCancel(context.Background())
	go func() {
		err = cdc.RunBloomWorker(ctxCDC, reader, rdb)
		if err != nil {
			log.Println("CDC worker error:", err)
			return
		}
	}()
	defer cancelCDC()

	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("userservice"),
		provider.WithExportEndpoint("localhost:4317"),
		provider.WithInsecure(),
		provider.WithEnableMetrics(false),
	)
	defer p.Shutdown(context.Background())

	r, err := etcd.NewEtcdRegistry([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}

	addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:9001")

	svr := user.NewServer(&UserServiceImpl{DBManager: &dao.DBManager{DB: db, RDB: rdb, Node: snowflakeNode}},
		server.WithSuite(tracing.NewServerSuite()),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "userservice"}),
		server.WithServiceAddr(addr),
		server.WithRegistry(r),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{
			ServiceName: "userservice",
		}),
		kServer.WithTracer(prometheus.NewServerTracer(":10002", "/metrics")))

	err = svr.Run()

	if err != nil {
		log.Println(err.Error())
	}
}
