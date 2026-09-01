package rerank

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/global"
	"net/http"
	"strings"
	"time"
)

const (
	defaultRerankURL   = "http://127.0.0.1:8000/rerank"
	defaultRerankModel = "bge-reranker-v2-m3"
	defaultRerankFmt   = "documents"
	maxRerankBatch     = 50
)

var rerankHTTPClient = &http.Client{Timeout: 15 * time.Second}

type rerankRequest struct {
	Model     string   `json:"model,omitempty"`
	Query     string   `json:"query"`
	Documents []string `json:"documents,omitempty"`
	Texts     []string `json:"texts,omitempty"`
}

type rerankResponse struct {
	Scores []float64 `json:"scores"`
	Data   struct {
		Scores []float64 `json:"scores"`
	} `json:"data"`
	Results []rerankResult `json:"results"`
}

type rerankResult struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
}

// scoreDocuments calls a local bge-reranker-v2-m3 cross-encoder service and
// returns scores aligned with docs by index. docs must not exceed maxRerankBatch.
func scoreDocuments(ctx context.Context, query string, docs []string) ([]float64, error) {
	if len(docs) == 0 {
		return nil, errors.New("empty rerank documents")
	}
	if len(docs) > maxRerankBatch {
		return nil, fmt.Errorf("rerank batch size %d exceeds limit %d", len(docs), maxRerankBatch)
	}

	bodyBytes, err := json.Marshal(newRerankRequest(strings.TrimSpace(query), docs))
	if err != nil {
		return nil, fmt.Errorf("marshal rerank request failed: %w", err)
	}

	return dependency.Do(ctx, dependency.Reranker, "score", func(callCtx context.Context) ([]float64, error) {
		return scoreDocumentsOnce(callCtx, bodyBytes, len(docs))
	})
}

func scoreDocumentsOnce(ctx context.Context, bodyBytes []byte, docCount int) ([]float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rerankURL(), bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("build rerank request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := rerankHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call local bge rerank failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read local bge rerank response failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &dependency.HTTPStatusError{StatusCode: resp.StatusCode, Body: string(raw)}
	}

	scores, err := parseRerankScores(raw, docCount)
	if err != nil {
		return nil, err
	}
	if len(scores) != docCount {
		return nil, fmt.Errorf("rerank scores length %d mismatch docs %d", len(scores), docCount)
	}

	return scores, nil
}

func newRerankRequest(query string, docs []string) rerankRequest {
	req := rerankRequest{
		Model: rerankModel(),
		Query: query,
	}
	switch rerankFormat() {
	case "tei":
		req.Texts = docs
	default:
		req.Documents = docs
	}
	return req
}

func parseRerankScores(raw []byte, docCount int) ([]float64, error) {
	var parsed rerankResponse
	if err := json.Unmarshal(raw, &parsed); err == nil {
		switch {
		case len(parsed.Scores) > 0:
			return parsed.Scores, nil
		case len(parsed.Data.Scores) > 0:
			return parsed.Data.Scores, nil
		case len(parsed.Results) > 0:
			return scoresFromResults(parsed.Results, docCount)
		}
	}

	var results []rerankResult
	if err := json.Unmarshal(raw, &results); err != nil {
		return nil, fmt.Errorf("unmarshal local bge rerank response failed: %w, body=%s", err, string(raw))
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("empty local bge rerank response: body=%s", string(raw))
	}
	return scoresFromResults(results, docCount)
}

func scoresFromResults(results []rerankResult, docCount int) ([]float64, error) {
	scores := make([]float64, docCount)
	seen := make([]bool, docCount)
	for _, result := range results {
		if result.Index < 0 || result.Index >= docCount {
			return nil, fmt.Errorf("rerank result index %d out of range docs %d", result.Index, docCount)
		}
		scores[result.Index] = result.Score
		seen[result.Index] = true
	}
	for i, ok := range seen {
		if !ok {
			return nil, fmt.Errorf("missing rerank score for doc index %d", i)
		}
	}
	return scores, nil
}

func rerankURL() string {
	if global.Config != nil && strings.TrimSpace(global.Config.RerankURL) != "" {
		return strings.TrimSpace(global.Config.RerankURL)
	}
	return defaultRerankURL
}

func rerankModel() string {
	if global.Config != nil && strings.TrimSpace(global.Config.RerankModel) != "" {
		return strings.TrimSpace(global.Config.RerankModel)
	}
	return defaultRerankModel
}

func rerankFormat() string {
	if global.Config != nil && strings.TrimSpace(global.Config.RerankFormat) != "" {
		return strings.ToLower(strings.TrimSpace(global.Config.RerankFormat))
	}
	return defaultRerankFmt
}
