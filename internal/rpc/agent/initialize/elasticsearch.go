package initialize

import (
	"context"
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/rag"
)

func InitDocElasticsearch(ctx context.Context) (*rag.ElasticsearchManager, error) {
	addr := global.Config.ElasticsearchConfig.Addr
	index := global.Config.ElasticsearchConfig.DocIndex
	if addr == "" || index == "" {
		return nil, fmt.Errorf("empty elasticsearch config")
	}

	mgr, err := rag.NewElasticsearchManager([]string{addr}, index)
	if err != nil {
		return nil, err
	}
	if err := mgr.EnsureDocIndex(ctx); err != nil {
		return nil, err
	}
	return mgr, nil
}
