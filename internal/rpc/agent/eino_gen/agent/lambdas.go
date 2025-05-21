package indexer

import (
	"context"
	_type "liveclass/internal/rpc/agent/eino_gen/agent/type"
	"time"
)

func newInputToQuery(ctx context.Context, input *_type.UserMessage, opts ...any) (output string, err error) {
	return input.Query, nil
}

func newInputToHistory(ctx context.Context, input *_type.UserMessage, opts ...any) (output map[string]any, err error) {
	return map[string]any{
		"content": input.Query,
		"history": input.History,
		"date":    time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}
