package rerank

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/model"
	my_prompt "liveclass/internal/rpc/agent/prompt"
	"liveclass/internal/rpc/agent/rag"
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

	resp, err := global.ChatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(my_prompt.RerankSystemPrompt),
		schema.UserMessage(fmt.Sprintf(
			my_prompt.RerankUserPrompt,
			strings.TrimSpace(query),
			candidates,
		)),
	})
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

type docRerankItem struct {
	ChunkID string  `json:"chunk_id"`
	Score   float64 `json:"score"`
	Reason  string  `json:"reason,omitempty"`
}

// Docs orders doc chunks by relevance using the shared LLM.
func Docs(ctx context.Context, query string, chunks []rag.DocChunk, topK int) ([]rag.DocChunk, error) {
	if len(chunks) <= 1 || global.ChatModel == nil || strings.TrimSpace(query) == "" {
		return chunks, nil
	}
	if topK <= 0 {
		topK = 3
	}

	candidates := formatDocCandidates(chunks)
	if candidates == "" {
		return chunks, nil
	}

	resp, err := global.ChatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(my_prompt.DocRerankSystemPrompt),
		schema.UserMessage(fmt.Sprintf(
			my_prompt.DocRerankUserPrompt,
			strings.TrimSpace(query),
			candidates,
		)),
	})
	if err != nil {
		return chunks, err
	}
	if resp == nil {
		return chunks, errors.New("nil doc rerank response")
	}

	items, err := parseDocResp(resp.Content)
	if err != nil || len(items) == 0 {
		return chunks, err
	}

	chunkMap := make(map[string]rag.DocChunk, len(chunks))
	for _, chunk := range chunks {
		if chunk.ID != "" {
			chunkMap[chunk.ID] = chunk
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	result := make([]rag.DocChunk, 0, len(items))
	for _, item := range items {
		if item.Score <= 0 {
			continue
		}
		if chunk, ok := chunkMap[item.ChunkID]; ok {
			chunk.Score = item.Score
			result = append(result, chunk)
		}
		if len(result) >= topK {
			break
		}
	}
	if len(result) == 0 {
		return chunks, nil
	}
	return result, nil
}

func formatDocCandidates(chunks []rag.DocChunk) string {
	var b strings.Builder
	index := 1
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk.Text) == "" {
			continue
		}
		id := chunk.ID
		if id == "" {
			id = fmt.Sprintf("chunk_%d", index)
		}
		fmt.Fprintf(&b, "%d. [chunk_id=%s][source=%s] %s\n",
			index,
			id,
			strings.TrimSpace(chunk.Source),
			strings.TrimSpace(chunk.Text),
		)
		index++
	}
	return strings.TrimSpace(b.String())
}

func parseDocResp(raw string) ([]docRerankItem, error) {
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start == -1 || end == -1 || end <= start {
		return nil, errors.New("doc rerank: no json array")
	}
	var items []docRerankItem
	if err := json.Unmarshal([]byte(raw[start:end+1]), &items); err != nil {
		return nil, err
	}
	return items, nil
}
