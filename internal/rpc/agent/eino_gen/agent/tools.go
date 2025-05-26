package agent

import (
	"context"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino/components/tool"
	"liveclass/internal/rpc/agent/mcp"
)

func GetTools(ctx context.Context) ([]tool.BaseTool, error) {
	ddgTool, err := newDDGSearch(ctx, nil)
	if err != nil {
		return nil, err
	}

	mcpTools := mcp.GetSSETool(ctx, "http://localhost:12345/sse")

	var tools []tool.BaseTool

	tools = mcpTools
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
