package agent

import (
	"context"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/toolruntime"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

func newReactAgent(ctx context.Context, dbm *memory.DBManager) (lba *compose.Lambda, err error) {
	//创建配置
	maxSteps := 25
	if global.Config != nil && global.Config.AgentRuntime.MaxSteps > 0 {
		maxSteps = global.Config.AgentRuntime.MaxSteps
	}
	config := &react.AgentConfig{
		MaxStep:            maxSteps,
		ToolReturnDirectly: map[string]struct{}{},
	}
	config.MessageModifier = func(ctx context.Context, input []*schema.Message) []*schema.Message {
		tracker := toolruntime.ProgressTrackerFromContext(ctx)
		if tracker == nil || !tracker.ConsumeReminder() {
			return input
		}
		result := append([]*schema.Message(nil), input...)
		return append(result, schema.SystemMessage("复杂任务已连续多步没有计划进展：请更新当前 TaskPlan 步骤状态，或说明该任务不需要计划。普通问答不要创建计划。"))
	}

	config.ToolCallingModel = global.ChatModel

	tools, err := GetTools(ctx, dbm)
	if err != nil {
		return nil, err
	}
	config.ToolsConfig.Tools = tools

	ins, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}

	lba, err = compose.AnyLambda(ins.Generate, ins.Stream, nil, nil)
	if err != nil {
		return nil, err
	}
	return lba, nil
}
