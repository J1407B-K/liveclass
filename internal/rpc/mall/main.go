package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"liveclass/idl/kitex_gen/mall/mallservice"
	"liveclass/internal/rpc/mall/domain"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	registry "github.com/kitex-contrib/registry-etcd"
)

func main() {
	catalogDB, _, err := domain.OpenMySQL(domain.MallDatabase)
	if err != nil {
		log.Fatal(err)
	}
	orderDB, orderRaw, err := domain.OpenMySQL(domain.OrderDatabase)
	if err != nil {
		log.Fatal(err)
	}
	if err = domain.MigrateMall(catalogDB); err != nil {
		log.Fatal(err)
	}
	if err = domain.MigrateOrder(orderDB); err != nil {
		log.Fatal(err)
	}
	if err = domain.EnsureBarrierTable(orderRaw); err != nil {
		log.Fatal(err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	callbackAddr := env("LIVECLASS_MALL_ORDER_CALLBACK_ADDR", "0.0.0.0:19100")
	go func() {
		if runErr := domain.RunHTTP(ctx, callbackAddr, (&domain.BranchServer{DB: orderRaw, Role: "order"}).Handler()); runErr != nil {
			log.Printf("order DTM callback server: %v", runErr)
			cancel()
		}
	}()

	coordinator := &domain.Coordinator{CatalogDB: catalogDB, OrderDB: orderDB, Saga: domain.DTMSagaSubmitter{
		ServerURL:    env("LIVECLASS_DTM_SERVER", "http://127.0.0.1:36789/api/dtmsvr"),
		OrderURL:     env("LIVECLASS_MALL_ORDER_CALLBACK_URL", "http://host.docker.internal:19100"),
		InventoryURL: env("LIVECLASS_MALL_INVENTORY_CALLBACK_URL", "http://host.docker.internal:19101"),
		PointsURL:    env("LIVECLASS_MALL_POINTS_CALLBACK_URL", "http://host.docker.internal:19102"),
		Timeout:      15 * time.Second,
	}}

	etcdRegistry, err := registry.NewEtcdRegistry([]string{env("LIVECLASS_ETCD_ADDR", "127.0.0.1:2379")})
	if err != nil {
		log.Fatal(err)
	}
	addr, err := net.ResolveTCPAddr("tcp", env("LIVECLASS_MALL_RPC_ADDR", "127.0.0.1:9010"))
	if err != nil {
		log.Fatal(err)
	}
	svr := mallservice.NewServer(&MallServiceImpl{coordinator: coordinator, catalogDB: catalogDB, orderDB: orderDB},
		server.WithServiceAddr(addr),
		server.WithRegistry(etcdRegistry),
		server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "mallservice"}),
	)
	go func() {
		<-ctx.Done()
		_ = svr.Stop()
	}()
	if err = svr.Run(); err != nil {
		log.Print(err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
