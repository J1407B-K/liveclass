package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cloudwego/kitex/client"
	"github.com/go-redis/redis/v8"
	etcd "github.com/kitex-contrib/registry-etcd"
	"gorm.io/gorm"
	"io"
	"liveclass/idl/kitex_gen/common"
	live "liveclass/idl/kitex_gen/live"
	"liveclass/idl/kitex_gen/user"
	"liveclass/idl/kitex_gen/user/userservice"
	"liveclass/internal/rpc/live/dao"
	"liveclass/internal/rpc/live/model"
	"liveclass/internal/utils/cut"
	"log"
	"net/http"
)

// LiveServiceImpl implements the last service interface defined in the IDL.
type LiveServiceImpl struct {
	DB             *gorm.DB
	RDB            *redis.Client
	userCli        userservice.Client
	GetLiveKeyAddr string
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
	//请求userrpc拿到userinfo
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！非老师不能创建课程/直播")
	}

	var datajson model.Livegojson
	//拿到json
	data, err := http.Get(s.GetLiveKeyAddr + cut.CombineLesson(req.Livename, username))
	if err != nil {
		return nil, err
	}
	defer data.Body.Close()

	//读取
	body, err := io.ReadAll(data.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &datajson)
	if err != nil {
		return nil, err
	}

	if datajson.Status != 200 {
		return nil, errors.New("livego 生成key错误")
	}

	err = dao.SaveLesson(s.DB, req, username, datajson.Data)
	if err != nil {
		return nil, err
	}

	return &live.CreateLiveResp{
		Resp: &common.Resp{
			Data: datajson.Data,
		},
	}, nil
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

// CloseLive implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) CloseLive(ctx context.Context, req *live.CloseLiveReq) (resp *live.CloseLiveResp, err error) {
	// TODO: Your code here...
	return
}
