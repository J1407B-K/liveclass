package agent

import (
	"context"
	"fmt"
	"liveclass/internal/rpc/agent/model"
	"time"
)

func newInputToTemplateVars(ctx context.Context, input *model.UserMessage, opts ...any) (map[string]any, error) {
	facts := input.Facts
	if facts == "" {
		facts = "暂无相关长期记忆。"
	}
	profile := input.Profile
	if profile == "" {
		profile = "暂无画像数据。"
	}

	return map[string]any{
		"content": input.Query,
		"history": input.History,
		"facts":   facts,
		"profile": profile,
		"user_id": fmt.Sprintf("%d", input.ID),
		"date":    time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}
