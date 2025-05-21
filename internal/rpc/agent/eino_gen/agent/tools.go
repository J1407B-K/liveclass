package indexer

import (
	"context"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino/components/tool"
)

func GetTools(ctx context.Context) ([]tool.BaseTool, error) {
	toolDDGSearch, err := newDDGSearch(ctx, nil)
	if err != nil {
		return nil, err
	}

	//TODO: MCP

	return []tool.BaseTool{
		toolDDGSearch,
	}, nil
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
