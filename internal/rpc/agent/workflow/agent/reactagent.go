package agent

import (
	"context"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

func newReactAgentWithMaxSteps(ctx context.Context, dbm *memory.DBManager, maxSteps int) (lba *compose.Lambda, err error) {
	configured := 25
	if global.Config != nil && global.Config.AgentRuntime.MaxSteps > 0 {
		configured = global.Config.AgentRuntime.MaxSteps
	}
	if maxSteps > 0 {
		configured = maxSteps
	}
	config := &react.AgentConfig{MaxStep: configured, ToolReturnDirectly: map[string]struct{}{}, ToolCallingModel: global.ChatModel}
	tools, err := GetTools(ctx, dbm)
	if err != nil {
		return nil, err
	}
	config.ToolsConfig.Tools = tools
	ins, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}
	return compose.AnyLambda(ins.Generate, ins.Stream, nil, nil)
}
