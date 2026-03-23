package rag

import (
	"context"
	"errors"
	"fmt"
	"liveclass/internal/rpc/agent/memory"

	"github.com/qdrant/go-client/qdrant"
)

type DocRetriever struct {
	mgr *memory.QdrantManager
}

type DocChunk struct {
	ID       string
	Text     string
	LessonID int64
	Source   string
	ChunkIdx int32
	Score    float64
}

func NewDocRetriever(mgr *memory.QdrantManager) (*DocRetriever, error) {
	if mgr == nil {
		return nil, errors.New("nil qdrant manager")
	}
	return &DocRetriever{mgr: mgr}, nil
}

func (r *DocRetriever) Search(ctx context.Context, lessonID int64, vector []float64, limit int) ([]DocChunk, error) {
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
	points, err := r.mgr.Client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: r.mgr.Collection,
		Query:          qdrant.NewQuery(memory.Float64To32(vector)...),
		Filter:         filter,
		Limit:          &limit64,
		WithPayload:    qdrant.NewWithPayload(true),
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
