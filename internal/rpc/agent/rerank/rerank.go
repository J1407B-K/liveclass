package rerank

import (
	"context"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/model"
	"liveclass/internal/rpc/agent/rag"
	"sort"
	"strings"
)

const (
	defaultTopK       = 5
	minScoreThreshold = 0.2
	defaultDocTopK    = 3
	minDocScore       = 0.0
)

type scoredIndex struct {
	idx   int
	score float64
}

func Facts(ctx context.Context, query string, facts []*model.UserFact, topK int) ([]*model.UserFact, error) {
	if len(facts) <= 1 || global.Config == nil || strings.TrimSpace(query) == "" {
		return facts, nil
	}
	if topK <= 0 {
		topK = defaultTopK
	}

	idxMap := make([]int, 0, len(facts))
	docs := make([]string, 0, len(facts))
	for i, f := range facts {
		if f == nil || strings.TrimSpace(f.Content) == "" {
			continue
		}
		idxMap = append(idxMap, i)
		docs = append(docs, strings.TrimSpace(f.Content))
	}
	if len(docs) <= 1 {
		return facts, nil
	}

	scores, err := scoreDocuments(ctx, strings.TrimSpace(query), docs)
	if err != nil {
		return facts, err
	}

	ranked := sortByScore(scores)

	result := make([]*model.UserFact, 0, topK)
	for _, r := range ranked {
		if r.score < minScoreThreshold {
			continue
		}
		result = append(result, facts[idxMap[r.idx]])
		if len(result) >= topK {
			break
		}
	}
	if len(result) == 0 {
		return facts, nil
	}
	return result, nil
}

func Docs(ctx context.Context, query string, chunks []rag.DocChunk, topK int) ([]rag.DocChunk, error) {
	if len(chunks) <= 1 || global.Config == nil || strings.TrimSpace(query) == "" {
		return chunks, nil
	}
	if topK <= 0 {
		topK = defaultDocTopK
	}

	idxMap := make([]int, 0, len(chunks))
	docs := make([]string, 0, len(chunks))
	for i, c := range chunks {
		if strings.TrimSpace(c.Text) == "" {
			continue
		}
		idxMap = append(idxMap, i)
		docs = append(docs, strings.TrimSpace(c.Text))
	}
	if len(docs) <= 1 {
		return chunks, nil
	}

	scores, err := scoreDocuments(ctx, strings.TrimSpace(query), docs)
	if err != nil {
		return chunks, err
	}

	ranked := sortByScore(scores)

	result := make([]rag.DocChunk, 0, topK)
	for _, r := range ranked {
		if r.score <= minDocScore {
			continue
		}
		chunk := chunks[idxMap[r.idx]]
		chunk.Score = r.score
		result = append(result, chunk)
		if len(result) >= topK {
			break
		}
	}
	if len(result) == 0 {
		return chunks, nil
	}
	return result, nil
}

func sortByScore(scores []float64) []scoredIndex {
	ranked := make([]scoredIndex, len(scores))
	for i, s := range scores {
		ranked[i] = scoredIndex{idx: i, score: s}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})
	return ranked
}
