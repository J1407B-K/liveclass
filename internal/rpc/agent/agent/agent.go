package agent

import (
	"context"
	"github.com/cloudwego/eino/schema"
	"liveclass/internal/rpc/agent/eino_gen/agent"
	_type "liveclass/internal/rpc/agent/eino_gen/agent/type"
	"liveclass/internal/rpc/agent/memory"
)

var mem = memory.GetDefaultMemory()

func ChatWithAgent(ctx context.Context, id, msg string) (string, error) {
	runner, err := agent.BuildAgent(ctx)
	if err != nil {
		return "", err
	}

	conv := mem.GetConversation(id, true)
	userMsg := &_type.UserMessage{ID: id, Query: msg, History: conv.GetMessages()}

	resp, err := runner.Invoke(ctx, userMsg)
	if err != nil {
		return "", err
	}

	conv.Append(schema.UserMessage(msg))
	conv.Append(resp)

	return resp.Content, nil
}
