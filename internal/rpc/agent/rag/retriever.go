package rag

import (
	"context"
	"errors"
	"fmt"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/memory"
	"strings"

	"github.com/qdrant/go-client/qdrant"
)

type DocRetriever struct {
	mgr *memory.QdrantManager
	es  *ElasticsearchManager
}

type DocChunk struct {
	ID         string
	ParentID   string
	Text       string
	ParentText string
	Heading    string
	LessonID   int64
	Source     string
	ChunkIdx   int32
	ChildIdx   int32
	Score      float64
}

func NewDocRetriever(mgr *memory.QdrantManager, es ...*ElasticsearchManager) (*DocRetriever, error) {
	if mgr == nil {
		return nil, errors.New("nil qdrant manager")
	}
	r := &DocRetriever{mgr: mgr}
	if len(es) > 0 {
		r.es = es[0]
	}
	return r, nil
}

func (r *DocRetriever) Search(ctx context.Context, lessonID int64, vector []float64, limit int) ([]DocChunk, error) {
	return r.SearchHybrid(ctx, lessonID, "", vector, limit)
}

func (r *DocRetriever) SearchHybrid(ctx context.Context, lessonID int64, query string, vector []float64, limit int) ([]DocChunk, error) {
	children, err := r.SearchHybridChildren(ctx, lessonID, query, vector, limit)
	if err != nil {
		return nil, err
	}
	return ExpandAndDeduplicateParents(children), nil
}

func (r *DocRetriever) SearchHybridChildren(ctx context.Context, lessonID int64, query string, vector []float64, limit int) ([]DocChunk, error) {
	vectorChunks, err := r.SearchVector(ctx, lessonID, vector, limit)
	if err != nil {
		return nil, err
	}
	if r == nil || r.es == nil || strings.TrimSpace(query) == "" {
		return vectorChunks, nil
	}

	bm25Chunks, err := r.es.SearchDocs(ctx, lessonID, query, limit)
	if err != nil {
		dependency.Fallback(dependency.Elasticsearch, "search_docs")
		return vectorChunks, nil
	}
	return mergeDocChunks(vectorChunks, bm25Chunks), nil
}

func CitationID(chunk DocChunk) string {
	heading := strings.TrimSpace(strings.TrimLeft(chunk.Heading, "#"))
	if heading != "" {
		return chunk.Source + "#" + heading
	}
	return fmt.Sprintf("%s#%d", chunk.Source, chunk.ChunkIdx)
}

func (r *DocRetriever) SearchVector(ctx context.Context, lessonID int64, vector []float64, limit int) ([]DocChunk, error) {
	if r == nil || r.mgr == nil {
		return nil, errors.New("nil retriever")
	}
	if lessonID == 0 {
		return nil, errors.New("lesson id required")
	}
	if len(vector) == 0 {
		return nil, errors.New("empty query vector")
	}
	if limit <= 0 {
		limit = 3
	}

	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatchInt("lesson_id", lessonID),
		},
	}

	limit64 := uint64(limit)
	points, err := dependency.Do(ctx, dependency.Qdrant, "search_docs", func(callCtx context.Context) ([]*qdrant.ScoredPoint, error) {
		return r.mgr.Client.Query(callCtx, &qdrant.QueryPoints{
			CollectionName: r.mgr.Collection,
			Query:          qdrant.NewQuery(memory.Float64To32(vector)...),
			Filter:         filter,
			Limit:          &limit64,
			WithPayload:    qdrant.NewWithPayload(true),
		})
	})
	if err != nil {
		return nil, err
	}

	results := make([]DocChunk, 0, len(points))
	for _, point := range points {
		if point == nil || point.Payload == nil {
			continue
		}
		payload := point.Payload
		chunk := DocChunk{
			LessonID: lessonID,
			Score:    float64(point.Score),
		}
		if point.Id != nil {
			if uuid := point.Id.GetUuid(); uuid != "" {
				chunk.ID = uuid
			} else {
				chunk.ID = fmt.Sprintf("%d", point.Id.GetNum())
			}
		}
		if text := payload["text"]; text != nil {
			chunk.Text = text.GetStringValue()
		}
		if value := payload["parent_id"]; value != nil {
			chunk.ParentID = value.GetStringValue()
		}
		if value := payload["parent_text"]; value != nil {
			chunk.ParentText = value.GetStringValue()
		}
		if value := payload["heading"]; value != nil {
			chunk.Heading = value.GetStringValue()
		}
		if value := payload["child_idx"]; value != nil {
			chunk.ChildIdx = int32(value.GetIntegerValue())
		}
		if source := payload["source"]; source != nil {
			chunk.Source = source.GetStringValue()
		}
		if idx := payload["chunk_idx"]; idx != nil {
			chunk.ChunkIdx = int32(idx.GetIntegerValue())
		}
		if chunk.Text == "" {
			continue
		}
		results = append(results, chunk)
	}
	return results, nil
}

func ExpandAndDeduplicateParents(children []DocChunk) []DocChunk {
	seen := make(map[string]struct{})
	parents := make([]DocChunk, 0, len(children))
	for _, child := range children {
		if child.ParentID == "" || child.ParentText == "" {
			parents = append(parents, child)
			continue
		}
		if _, ok := seen[child.ParentID]; ok {
			continue
		}
		seen[child.ParentID] = struct{}{}
		child.ID = child.ParentID
		child.Text = child.ParentText
		parents = append(parents, child)
	}
	return parents
}

func mergeDocChunks(groups ...[]DocChunk) []DocChunk {
	seen := make(map[string]struct{})
	merged := make([]DocChunk, 0)
	for _, group := range groups {
		for _, chunk := range group {
			key := docChunkKey(chunk)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			merged = append(merged, chunk)
		}
	}
	return merged
}

func docChunkKey(chunk DocChunk) string {
	if strings.TrimSpace(chunk.ID) != "" {
		return chunk.ID
	}
	return fmt.Sprintf("%d:%s:%d:%s", chunk.LessonID, chunk.Source, chunk.ChunkIdx, chunk.Text)
}
