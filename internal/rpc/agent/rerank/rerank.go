package rerank

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/model"
	my_prompt "liveclass/internal/rpc/agent/prompt"
	"sort"
	"strings"

	"github.com/cloudwego/eino/schema"
)

type rerankItem struct {
	FactID int64   `json:"fact_id"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason,omitempty"`
}

const (
	defaultTopK       = 5
	minScoreThreshold = 0.2
)

// Facts selects the most relevant facts for the query using the shared chat model.
// If rerank fails, it falls back to the original ordering.
func Facts(ctx context.Context, query string, facts []*model.UserFact, topK int) ([]*model.UserFact, error) {
	if len(facts) <= 1 || global.ChatModel == nil || strings.TrimSpace(query) == "" {
		return facts, nil
	}
	if topK <= 0 {
		topK = defaultTopK
	}

	candidates := formatCandidates(facts)
	if candidates == "" {
		return facts, nil
	}

	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: my_prompt.RerankSystemPrompt,
		},
		{
			Role: schema.User,
			Content: fmt.Sprintf(
				my_prompt.RerankUserPrompt,
				strings.TrimSpace(query),
				candidates,
			),
		},
	}

	resp, err := global.ChatModel.Generate(ctx, messages)
	if err != nil {
		return facts, err
	}
	if resp == nil {
		return facts, errors.New("nil rerank response")
	}

	items, err := parseResp(resp.Content)
	if err != nil || len(items) == 0 {
		return facts, err
	}

	result := make([]*model.UserFact, 0, len(items))
	factMap := make(map[int64]*model.UserFact, len(facts))
	for _, f := range facts {
		if f != nil {
			factMap[f.ID] = f
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	for _, item := range items {
		if item.Score < minScoreThreshold {
			continue
		}
		if fact := factMap[item.FactID]; fact != nil {
			result = append(result, fact)
		}
		if len(result) >= topK {
			break
		}
	}

	if len(result) == 0 {
		return facts, nil
	}
	return result, nil
}

func formatCandidates(facts []*model.UserFact) string {
	var b strings.Builder
	index := 1
	for _, f := range facts {
		if f == nil || strings.TrimSpace(f.Content) == "" {
			continue
		}
		fmt.Fprintf(&b, "%d. [fact_id=%d][type=%s] %s\n",
			index,
			f.ID,
			strings.TrimSpace(f.FactType),
			strings.TrimSpace(f.Content),
		)
		index++
	}
	return strings.TrimSpace(b.String())
}

func parseResp(raw string) ([]rerankItem, error) {
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, errors.New("no json array in rerank response")
	}

	var items []rerankItem
	if err := json.Unmarshal([]byte(raw[start:end+1]), &items); err != nil {
		return nil, err
	}
	return items, nil
}
