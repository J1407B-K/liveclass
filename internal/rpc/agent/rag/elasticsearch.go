package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"liveclass/internal/rpc/agent/dependency"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
)

type ElasticsearchManager struct {
	Client *elasticsearch.Client
	Index  string
}

func NewElasticsearchManager(addrs []string, index string) (*ElasticsearchManager, error) {
	if len(addrs) == 0 || strings.TrimSpace(addrs[0]) == "" {
		return nil, fmt.Errorf("empty elasticsearch address")
	}
	if strings.TrimSpace(index) == "" {
		return nil, fmt.Errorf("empty elasticsearch index")
	}
	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: addrs})
	if err != nil {
		return nil, err
	}
	return &ElasticsearchManager{Client: client, Index: strings.TrimSpace(index)}, nil
}

func (m *ElasticsearchManager) EnsureDocIndex(ctx context.Context) error {
	if m == nil || m.Client == nil {
		return fmt.Errorf("nil elasticsearch manager")
	}

	resp, err := m.Client.Indices.Exists([]string{m.Index}, m.Client.Indices.Exists.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		return nil
	}
	if resp.StatusCode != 404 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("check elasticsearch index failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	mapping := map[string]any{
		"mappings": map[string]any{
			"properties": map[string]any{
				"lesson_id": map[string]any{"type": "long"},
				"source":    map[string]any{"type": "keyword"},
				"chunk_idx": map[string]any{"type": "integer"},
				"text": map[string]any{
					"type":     "text",
					"analyzer": "standard",
				},
			},
		},
	}
	body, err := json.Marshal(mapping)
	if err != nil {
		return err
	}
	resp, err = m.Client.Indices.Create(
		m.Index,
		m.Client.Indices.Create.WithContext(ctx),
		m.Client.Indices.Create.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create elasticsearch index failed: status=%d body=%s", resp.StatusCode, string(raw))
	}
	return nil
}

func (m *ElasticsearchManager) BulkUpsertDocChunks(ctx context.Context, chunks []DocChunk) error {
	if m == nil || m.Client == nil || len(chunks) == 0 {
		return nil
	}

	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	for _, chunk := range chunks {
		if strings.TrimSpace(chunk.ID) == "" || strings.TrimSpace(chunk.Text) == "" {
			continue
		}
		if err := enc.Encode(map[string]any{
			"index": map[string]any{
				"_index": m.Index,
				"_id":    chunk.ID,
			},
		}); err != nil {
			return err
		}
		if err := enc.Encode(map[string]any{
			"lesson_id": chunk.LessonID,
			"source":    chunk.Source,
			"chunk_idx": chunk.ChunkIdx,
			"text":      chunk.Text,
		}); err != nil {
			return err
		}
	}
	if body.Len() == 0 {
		return nil
	}

	bodyBytes := append([]byte(nil), body.Bytes()...)
	_, err := dependency.Do(ctx, dependency.Elasticsearch, "bulk_upsert_docs", func(callCtx context.Context) (struct{}, error) {
		return struct{}{}, m.bulkUpsertOnce(callCtx, bodyBytes)
	})
	return err
}

func (m *ElasticsearchManager) bulkUpsertOnce(ctx context.Context, body []byte) error {
	resp, err := m.Client.Bulk(bytes.NewReader(body), m.Client.Bulk.WithContext(ctx))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		raw, _ := io.ReadAll(resp.Body)
		return &dependency.HTTPStatusError{StatusCode: resp.StatusCode, Body: string(raw)}
	}

	var parsed struct {
		Errors bool `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return err
	}
	if parsed.Errors {
		return fmt.Errorf("bulk upsert elasticsearch docs contains item errors")
	}
	return nil
}

func (m *ElasticsearchManager) SearchDocs(ctx context.Context, lessonID int64, query string, limit int) ([]DocChunk, error) {
	if m == nil || m.Client == nil {
		return nil, fmt.Errorf("nil elasticsearch manager")
	}
	if lessonID == 0 {
		return nil, fmt.Errorf("lesson id required")
	}
	if strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 3
	}

	body, err := json.Marshal(map[string]any{
		"size": limit,
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{"term": map[string]any{"lesson_id": lessonID}},
				},
				"must": []any{
					map[string]any{
						"match": map[string]any{
							"text": map[string]any{"query": strings.TrimSpace(query)},
						},
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return dependency.Do(ctx, dependency.Elasticsearch, "search_docs", func(callCtx context.Context) ([]DocChunk, error) {
		return m.searchDocsOnce(callCtx, body)
	})
}

func (m *ElasticsearchManager) searchDocsOnce(ctx context.Context, body []byte) ([]DocChunk, error) {
	resp, err := m.Client.Search(
		m.Client.Search.WithContext(ctx),
		m.Client.Search.WithIndex(m.Index),
		m.Client.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		raw, _ := io.ReadAll(resp.Body)
		return nil, &dependency.HTTPStatusError{StatusCode: resp.StatusCode, Body: string(raw)}
	}

	var parsed struct {
		Hits struct {
			Hits []struct {
				ID     string  `json:"_id"`
				Score  float64 `json:"_score"`
				Source struct {
					LessonID int64  `json:"lesson_id"`
					Source   string `json:"source"`
					ChunkIdx int32  `json:"chunk_idx"`
					Text     string `json:"text"`
				} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	chunks := make([]DocChunk, 0, len(parsed.Hits.Hits))
	for _, hit := range parsed.Hits.Hits {
		if strings.TrimSpace(hit.Source.Text) == "" {
			continue
		}
		chunks = append(chunks, DocChunk{
			ID:       hit.ID,
			Text:     hit.Source.Text,
			LessonID: hit.Source.LessonID,
			Source:   hit.Source.Source,
			ChunkIdx: hit.Source.ChunkIdx,
			Score:    hit.Score,
		})
	}
	return chunks, nil
}
