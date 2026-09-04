package agent

import (
	"context"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/mcp"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/tool"
	"liveclass/internal/rpc/agent/toolruntime"
	"log"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

func GetTools(ctx context.Context, managers ...*memory.DBManager) ([]einotool.BaseTool, error) {
	mcpTools, err := mcp.GetSSETool(ctx, "http://127.0.0.1:12345/sse")
	if err != nil {
		return nil, err
	}

	registry := toolruntime.NewRegistry(&toolruntime.ModelRepairer{Model: global.ChatModel})
	register := func(base einotool.BaseTool, permission toolruntime.Permission, risk toolruntime.RiskLevel, timeout time.Duration, attempts int, output *openapi3.Schema) {
		err := registry.Register(ctx, base, toolruntime.ToolSpec{
			Permission: permission, RiskLevel: risk, Timeout: timeout,
			OutputSchema: output,
			Retry:        toolruntime.RetryPolicy{Attempts: attempts, Backoff: 50 * time.Millisecond},
		})
		if err != nil {
			log.Printf("register tool failed: %v", err)
		}
	}
	for _, mcpTool := range mcpTools {
		register(mcpTool, toolruntime.PermissionAuthenticated, toolruntime.RiskReadOnly, 3*time.Second, 2, outputSchema[string]())
	}

	if global.UserClient != nil {
		if userTool, err := tool.NewUserInfoTool(global.UserClient); err != nil {
			log.Printf("init user info tool failed: %v", err)
		} else {
			register(userTool, toolruntime.PermissionOwnUser, toolruntime.RiskReadOnly, 800*time.Millisecond, 1, outputSchema[tool.UserInfoResponse]())
		}
	}

	if global.LessonClient != nil {
		if lessonTool, err := tool.NewLessonInfoTool(global.LessonClient); err != nil {
			log.Printf("init lesson info tool failed: %v", err)
		} else {
			register(lessonTool, toolruntime.PermissionLessonMember, toolruntime.RiskReadOnly, 800*time.Millisecond, 1, outputSchema[tool.LessonInfoResponse]())
		}
	}

	if global.MallClient != nil {
		catalogTool, prepareTool, exchangeTool, mallErr := tool.NewMallTools(global.MallClient, []byte(global.Config.MallConfirmSecret))
		if mallErr != nil {
			log.Printf("init mall tools failed: %v", mallErr)
		} else {
			register(catalogTool, toolruntime.PermissionAuthenticated, toolruntime.RiskReadOnly, 800*time.Millisecond, 1, outputSchema[tool.MallCatalogResponse]())
			register(prepareTool, toolruntime.PermissionAuthenticated, toolruntime.RiskLow, 800*time.Millisecond, 1, outputSchema[tool.MallPrepareResponse]())
			register(exchangeTool, toolruntime.PermissionAuthenticated, toolruntime.RiskHigh, 20*time.Second, 1, outputSchema[tool.MallExchangeResponse]())
		}
	}

	if searchTool, err := tool.NewSearchTool(); err != nil {
		log.Printf("init search tool failed: %v", err)
	} else {
		register(searchTool, toolruntime.PermissionAuthenticated, toolruntime.RiskReadOnly, 2*time.Second, 1, outputSchema[[]tool.SearchResult]())
	}

	if formatTool, err := tool.NewFormatTool(global.SkriptsDir); err != nil {
		log.Printf("init format tool failed: %v", err)
	} else {
		register(formatTool, toolruntime.PermissionAuthenticated, toolruntime.RiskLow, 2*time.Second, 1, outputSchema[string]())
	}
	return registry.Tools(), nil
}

func outputSchema[T any]() *openapi3.Schema {
	ref, err := openapi3gen.NewSchemaRefForValue(new(T), nil)
	if err != nil {
		return nil
	}
	return ref.Value
}
