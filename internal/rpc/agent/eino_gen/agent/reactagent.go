package agent

import (
	"context"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

func newReactAgent(ctx context.Context) (lba *compose.Lambda, err error) {
	//创建配置
	config := &react.AgentConfig{
		MaxStep:            25,
		ToolReturnDirectly: map[string]struct{}{},
	}

	cm, err := newChatModel(ctx)
	if err != nil {
		return nil, err
	}
	config.ToolCallingModel = cm

	tools, err := GetTools(ctx)
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
