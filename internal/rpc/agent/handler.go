package main

import (
	"context"
	"github.com/cloudwego/kitex/client"
	etcd "github.com/kitex-contrib/registry-etcd"
	agent "liveclass/idl/kitex_gen/agent"
	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/user/userservice"
	my_agent "liveclass/internal/rpc/agent/agent"
	_const "liveclass/internal/rpc/agent/const"
	"log"
)

// AgentServiceImpl implements the last service interface defined in the IDL.
type AgentServiceImpl struct {
	userCli userservice.Client
}

func NewUserClient() (userservice.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	return userservice.NewClient("userservice", client.WithResolver(r)) // 指定 Resolver
}

// ChatWithAgent implements the AgentServiceImpl interface.
func (s *AgentServiceImpl) ChatWithAgent(ctx context.Context, req *agent.ChatWithAgentReq) (resp *agent.ChatWithAgentResp, err error) {
	agentResp, err := my_agent.ChatWithAgent(ctx, req.Userid, req.Message)
	if err != nil {
		var respAgain string
		for i := 0; i < _const.MAXRETRY; i++ {
			respAgain, err = my_agent.ChatWithAgent(ctx, req.Userid, req.Message)
			if err == nil {
				break
			}
		}
		agentResp = respAgain
	}

	return &agent.ChatWithAgentResp{Resp: &common.Resp{Data: agentResp}}, err
}
