package main

import (
	"context"
	agent "liveclass/idl/kitex_gen/agent"
)

// AgentServiceImpl implements the last service interface defined in the IDL.
type AgentServiceImpl struct{}

// ChatWithAgent implements the AgentServiceImpl interface.
func (s *AgentServiceImpl) ChatWithAgent(ctx context.Context, req *agent.ChatWithAgentReq) (resp *agent.ChatWithAgentResp, err error) {
	// TODO: Your code here...
	return
}
