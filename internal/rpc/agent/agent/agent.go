package agent

import (
	"context"
	"github.com/cloudwego/eino/schema"
	"liveclass/internal/rpc/agent/eino_gen/agent"
	_type "liveclass/internal/rpc/agent/eino_gen/agent/type"
	"liveclass/internal/rpc/agent/global"
)

// 直接用的Invoke，没用Stream，因为感觉所有流处理都很麻烦(hhhh
func ChatWithAgent(ctx context.Context, id, msg string) (string, error) {
	runner, err := agent.BuildAgent(ctx)
	if err != nil {
		return "", err
	}

	conv := global.Mem.GetConversation(id, true)
	userMsg := &_type.UserMessage{ID: id, Query: msg, History: conv.GetMessages()}

	resp, err := runner.Invoke(ctx, userMsg)
	if err != nil {
		return "", err
	}

	conv.Append(schema.UserMessage(msg))
	conv.Append(resp)

	return resp.Content, nil
}
