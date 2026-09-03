package main

import (
	"context"
	"fmt"
	agent "liveclass/idl/kitex_gen/agent"
	"liveclass/idl/kitex_gen/common"
	myagent "liveclass/internal/rpc/agent/agent"
	"liveclass/internal/rpc/agent/memory"
	"os"
	"strings"

	"github.com/bytedance/gopkg/cloud/metainfo"
	uuid2 "github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

// AgentServiceImpl implements the last service interface defined in the IDL.
type AgentServiceImpl struct {
	DBManager    *memory.DBManager
	agentRuntime *myagent.AgentRuntime

	//cozeloopClient cozeloop.Client

	sfAgent singleflight.Group
}

// ChatWithAgent implements the AgentServiceImpl interface.
func (s *AgentServiceImpl) ChatWithAgent(ctx context.Context, req *agent.ChatWithAgentReq) (*agent.ChatWithAgentResp, error) {
	if req == nil || strings.TrimSpace(req.Message) == "" {
		return &agent.ChatWithAgentResp{
			Resp: &common.Resp{Msg: "empty message"},
		}, nil
	}

	convID := memory.BuildConvID(req.Userid, req.ConvId)
	requestID := strings.TrimSpace(req.RequestId)
	if requestID == "" {
		requestID = uuid2.NewString()
	}

	//input := map[string]interface{}{
	//	"userid":     req.Userid,
	//	"message":    req.Message,
	//	"conv_id":    convID,
	//	"request_id": requestID,
	//}

	//cozeCtx, root := s.cozeloopClient.StartSpan(ctx, strconv.FormatInt(req.Userid, 10), "graph")
	//defer root.Finish(cozeCtx)
	//
	//root.SetInput(cozeCtx, input)

	sfKey := fmt.Sprintf("%d:%s:%s", req.Userid, convID, requestID)

	v, err, _ := s.sfAgent.Do(sfKey, func() (interface{}, error) {
		if s.agentRuntime == nil {
			return nil, fmt.Errorf("nil agent runtime")
		}
		variant, _ := metainfo.GetValue(ctx, "agent-eval-variant")
		if variant == "v1" && os.Getenv("AGENT_EVAL_ALLOW_V1") == "true" {
			return s.agentRuntime.RunVariant(ctx, "v1", req.Userid, convID, requestID, req.Message, req.LessonId)
		}
		return s.agentRuntime.Run(ctx, req.Userid, convID, requestID, req.Message, req.LessonId)
	})

	if err != nil {
		//root.SetOutput(cozeCtx, map[string]interface{}{
		//	"error":      err.Error(),
		//	"conv_id":    convID,
		//	"request_id": requestID,
		//	"shared":     shared,
		//})
		return &agent.ChatWithAgentResp{
			Resp: &common.Resp{Msg: "agent service temporarily unavailable"},
		}, err
	}

	agentResp, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("singleflight result type assertion failed")
	}

	//root.SetOutput(cozeCtx, map[string]interface{}{
	//	"reply":      agentResp,
	//	"conv_id":    convID,
	//	"request_id": requestID,
	//	"shared":     shared,
	//})

	return &agent.ChatWithAgentResp{
		Resp: &common.Resp{Msg: agentResp},
	}, nil
}

// ListAllUserConv implements the AgentServiceImpl interface.
func (s *AgentServiceImpl) ListAllUserConv(ctx context.Context, req *agent.ListAllUserConvReq) (*agent.ListAllUserConvResp, error) {
	convIDs, err := s.DBManager.ListConversations(ctx, req.Userid)
	if err != nil {
		return nil, err
	}

	if len(convIDs) == 0 {
		return &agent.ListAllUserConvResp{
			Resp: &common.Resp{Msg: "no conversations"},
		}, nil
	}

	return &agent.ListAllUserConvResp{
		Resp: &common.Resp{Msg: strings.Join(convIDs, "\n")},
	}, nil
}

// DelAllUserConv implements the AgentServiceImpl interface.
func (s *AgentServiceImpl) DelAllUserConv(ctx context.Context, req *agent.DelAllUserConvReq) (*agent.DelAllUserConvResp, error) {
	convID := memory.BuildConvID(req.Userid, req.ConvId)

	err := s.DBManager.DeleteConversation(ctx, req.Userid, convID)
	if err != nil {
		return nil, err
	}

	return &agent.DelAllUserConvResp{
		Resp: &common.Resp{
			Msg: "success",
		},
	}, nil
}
