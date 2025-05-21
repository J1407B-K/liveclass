package indexer

import (
	"context"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"liveclass/internal/rpc/agent/global"
)

func newChatModel(ctx context.Context) (cm *ark.ChatModel, err error) {
	//创建配置
	config := &ark.ChatModelConfig{
		Model:  global.Config.ChatModel,
		APIKey: global.Config.APIKey,
	}

	cm, err = ark.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	return cm, nil
}
