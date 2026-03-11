package agent

import (
	"context"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/mcp"
	"liveclass/internal/rpc/agent/skill"
	"log"

	"github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino/components/tool"
)

func GetTools(ctx context.Context) ([]tool.BaseTool, error) {
	ddgTool, err := newDDGSearch(ctx, nil)
	if err != nil {
		return nil, err
	}

	mcpTools, err := mcp.GetSSETool(ctx, "http://127.0.0.1:12345/sse")
	if err != nil {
		return nil, err
	}

	tools := make([]tool.BaseTool, 0, len(mcpTools)+3)
	tools = append(tools, mcpTools...)

	if global.UserClient != nil {
		if userTool, err := skill.NewUserInfoTool(global.UserClient); err != nil {
			log.Printf("init user info tool failed: %v", err)
		} else {
			tools = append(tools, userTool)
		}
	}

	if global.LessonClient != nil {
		if lessonTool, err := skill.NewLessonInfoTool(global.LessonClient); err != nil {
			log.Printf("init lesson info tool failed: %v", err)
		} else {
			tools = append(tools, lessonTool)
		}
	}

	tools = append(tools, ddgTool)

	return tools, nil
}

func defaultDDGSearchConfig(ctx context.Context) (*duckduckgo.Config, error) {
	config := &duckduckgo.Config{}
	return config, nil
}

func newDDGSearch(ctx context.Context, config *duckduckgo.Config) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultDDGSearchConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	tn, err = duckduckgo.NewTool(ctx, config)
	if err != nil {
		return nil, err
	}
	return tn, nil
}
