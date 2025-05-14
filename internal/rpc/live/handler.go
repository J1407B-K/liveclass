package main

import (
	"context"
	"github.com/cloudwego/kitex/client"
	"github.com/go-redis/redis/v8"
	etcd "github.com/kitex-contrib/registry-etcd"
	"gorm.io/gorm"
	live "liveclass/idl/kitex_gen/live"
	"liveclass/idl/kitex_gen/user/userservice"
	"log"
)

// LiveServiceImpl implements the last service interface defined in the IDL.
type LiveServiceImpl struct {
	DB      *gorm.DB
	RDB     *redis.Client
	userCli userservice.Client
}

func NewUserClient() (userservice.Client, error) {
	// 使用时请传入真实 etcd 的服务地址，本例中为 127.0.0.1:2379
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	return userservice.NewClient("userservice", client.WithResolver(r)) // 指定 Resolver
}

// CreateLive implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) CreateLive(ctx context.Context, req *live.CreateLiveReq) (resp *live.CreateLiveResp, err error) {

	return
}

// CoseLive implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) CoseLive(ctx context.Context, req *live.CloseLiveReq) (resp *live.CloseLiveResp, err error) {
	// TODO: Your code here...
	return
}

// AddUserInLive implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) AddUserInLive(ctx context.Context, req *live.AddUserInLiveReq) (resp *live.AddUserInLiveResp, err error) {
	// TODO: Your code here...
	return
}

// DelUserInlive implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) DelUserInlive(ctx context.Context, req *live.DelUserInLiveReq) (resp *live.DelUserInLiveResp, err error) {
	// TODO: Your code here...
	return
}
