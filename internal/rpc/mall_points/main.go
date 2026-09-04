package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudwego/kitex/pkg/rpcinfo"
	"github.com/cloudwego/kitex/server"
	registry "github.com/kitex-contrib/registry-etcd"
	"liveclass/idl/kitex_gen/mall_points/pointsservice"
	"liveclass/internal/rpc/mall/domain"
)

func main() {
	db, raw, err := domain.OpenMySQL()
	if err != nil {
		log.Fatal(err)
	}
	if err = domain.MigratePoints(db); err != nil {
		log.Fatal(err)
	}
	if err = domain.EnsureBarrierTable(raw); err != nil {
		log.Fatal(err)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		if callbackErr := domain.RunHTTP(ctx, env("LIVECLASS_MALL_POINTS_CALLBACK_ADDR", "0.0.0.0:19102"), (&domain.BranchServer{DB: raw, Role: "points"}).Handler()); callbackErr != nil {
			log.Printf("points DTM callback: %v", callbackErr)
			cancel()
		}
	}()
	r, err := registry.NewEtcdRegistry([]string{env("LIVECLASS_ETCD_ADDR", "127.0.0.1:2379")})
	if err != nil {
		log.Fatal(err)
	}
	rpcAddr, err := net.ResolveTCPAddr("tcp", env("LIVECLASS_MALL_POINTS_RPC_ADDR", "127.0.0.1:9012"))
	if err != nil {
		log.Fatal(err)
	}
	svr := pointsservice.NewServer(&pointsServiceImpl{db: db}, server.WithServiceAddr(rpcAddr), server.WithRegistry(r), server.WithServerBasicInfo(&rpcinfo.EndpointBasicInfo{ServiceName: "pointsservice"}))
	go func() { <-ctx.Done(); _ = svr.Stop() }()
	if err = svr.Run(); err != nil {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
