package fact

import (
	"context"
	"encoding/json"
	"liveclass/internal/rpc/agent/model"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func factInputToVars(ctx context.Context, input *model.FactExtractInput, opts ...any) (map[string]any, error) {
	return map[string]any{
		"message": input.Message,
	}, nil
}

func factMessageToCandidates(ctx context.Context, msg *schema.Message, opts ...any) ([]*model.FactCandidate, error) {
	if msg == nil {
		return []*model.FactCandidate{}, nil
	}

	text := strings.TrimSpace(msg.Content)
	if text == "" {
		return []*model.FactCandidate{}, nil
	}

	var facts []*model.FactCandidate
	if err := json.Unmarshal([]byte(text), &facts); err != nil {
		// 解析失败就返回空，避免影响主流程
		return []*model.FactCandidate{}, nil
	}

	// 简单过滤
	res := make([]*model.FactCandidate, 0, len(facts))
	for _, f := range facts {
		if f == nil {
			continue
		}
		f.FactType = strings.TrimSpace(f.FactType)
		f.Content = strings.TrimSpace(f.Content)
		if f.FactType == "" || f.Content == "" {
			continue
		}
		if f.Confidence < 0 {
			f.Confidence = 0
		}
		if f.Confidence > 1 {
			f.Confidence = 1
		}
		res = append(res, f)
	}

	return res, nil
}
