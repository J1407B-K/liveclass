package agent

import (
	"context"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/mcp"
	"liveclass/internal/rpc/agent/tool"
	"log"

	einotool "github.com/cloudwego/eino/components/tool"
)

func GetTools(ctx context.Context) ([]einotool.BaseTool, error) {
	mcpTools, err := mcp.GetSSETool(ctx, "http://127.0.0.1:12345/sse")
	if err != nil {
		return nil, err
	}

	tools := make([]einotool.BaseTool, 0, len(mcpTools)+4)
	tools = append(tools, mcpTools...)

	if global.UserClient != nil {
		if userTool, err := tool.NewUserInfoTool(global.UserClient); err != nil {
			log.Printf("init user info tool failed: %v", err)
		} else {
			tools = append(tools, userTool)
		}
	}

	if global.LessonClient != nil {
		if lessonTool, err := tool.NewLessonInfoTool(global.LessonClient); err != nil {
			log.Printf("init lesson info tool failed: %v", err)
		} else {
			tools = append(tools, lessonTool)
		}
	}

	if searchTool, err := tool.NewSearchTool(); err != nil {
		log.Printf("init search tool failed: %v", err)
	} else {
		tools = append(tools, searchTool)
	}

	if formatTool, err := tool.NewFormatTool(global.SkriptsDir); err != nil {
		log.Printf("init format tool failed: %v", err)
	} else {
		tools = append(tools, formatTool)
	}

	return tools, nil
}
