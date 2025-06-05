package main

import (
	"context"
	"github.com/cloudwego/kitex/client"
	"github.com/coze-dev/cozeloop-go"
	etcd "github.com/kitex-contrib/registry-etcd"
	agent "liveclass/idl/kitex_gen/agent"
	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/user/userservice"
	myagent "liveclass/internal/rpc/agent/agent"
	_const "liveclass/internal/rpc/agent/const"
	"liveclass/internal/rpc/agent/global"
	"log"
	"os"
	"strings"
)

// AgentServiceImpl implements the last service interface defined in the IDL.
type AgentServiceImpl struct {
	userCli userservice.Client

	cozeloopClient cozeloop.Client
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
	input := map[string]interface{}{
		"message": req.Message,
	}

	cozeCtx, root := s.cozeloopClient.StartSpan(context.Background(), req.Userid, "graph")
	root.SetInput(cozeCtx, input)

	agentResp, err := myagent.ChatWithAgent(ctx, req.Userid, req.Message)
	if err != nil {
		var respAgain string
		for i := 0; i < _const.MAXRETRY; i++ {
			respAgain, err = myagent.ChatWithAgent(ctx, req.Userid, req.Message)
			if err == nil {
				break
			}
		}
		agentResp = respAgain
	}

	root.SetOutput(cozeCtx, agentResp)
	root.Finish(cozeCtx)

	s.cozeloopClient.Close(cozeCtx)

	return &agent.ChatWithAgentResp{Resp: &common.Resp{Data: agentResp}}, nil
}

// ListAllUserConv implements the AgentServiceImpl interface.
func (s *AgentServiceImpl) ListAllUserConv(ctx context.Context, req *agent.ListAllUserConvReq) (resp *agent.ListAllUserConvResp, err error) {
	convsf := global.Mem.ListConversations()

	for _, convf := range convsf {
		if !strings.Contains(convf, req.Userid) {
			continue
		}

		bytes, err := os.ReadFile("data/memory/" + convf + ".jsonl")
		if err != nil {
			return nil, err
		}

		return &agent.ListAllUserConvResp{Resp: &common.Resp{Data: string(bytes)}}, nil
	}
	return nil, nil
}

// DelAllUserConv implements the AgentServiceImpl interface.
func (s *AgentServiceImpl) DelAllUserConv(ctx context.Context, req *agent.DelAllUserConvReq) (resp *agent.DelAllUserConvResp, err error) {
	convsf := global.Mem.ListConversations()

	for _, convf := range convsf {
		if !strings.Contains(convf, req.Userid) {
			continue
		}

		err := os.Remove("data/memory/" + convf + ".jsonl")
		if err != nil {
			return nil, err
		}

		return &agent.DelAllUserConvResp{
			Resp: &common.Resp{
				Data: "success",
			},
		}, nil
	}
	return &agent.DelAllUserConvResp{
		Resp: &common.Resp{
			Data: "not found file",
		},
	}, nil
}
