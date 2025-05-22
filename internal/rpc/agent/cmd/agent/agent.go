package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/schema"
	"liveclass/internal/rpc/agent/eino_gen/agent"
	_type "liveclass/internal/rpc/agent/eino_gen/agent/type"
	"liveclass/internal/rpc/agent/mcp"
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

func main() {
	go mcp.StartMCPServer()

	resp, err := ChatWithAgent(context.Background(), "1", "根据网上所说的方法，给我提供学习方法")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp)
}
