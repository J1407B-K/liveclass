package agent

import (
	"context"
	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino/components/embedding"
	"liveclass/internal/rpc/agent/global"
)

func newEmbedding(ctx context.Context) (eb embedding.Embedder, err error) {
	//配置文件
	config := &ark.EmbeddingConfig{
		BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		APIKey:  global.Config.APIKey,
		Model:   global.Config.EmbeddingModel,
	}
	//创建实例
	eb, err = ark.NewEmbedder(ctx, config)
	if err != nil {
		return nil, err
	}
	return eb, nil
}
