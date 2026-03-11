package memory

import (
	"context"
	"errors"
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/model"

	"github.com/qdrant/go-client/qdrant"
)

func (m *DBManager) UpsertFactVector(
	ctx context.Context,
	factID int64,
	userID int64,
	factType string,
	sourceConv string,
	isActive bool,
	vector []float32,
) error {
	point := &qdrant.PointStruct{
		Id:      qdrant.NewIDNum(uint64(factID)),
		Vectors: qdrant.NewVectors(vector...),
		Payload: qdrant.NewValueMap(map[string]any{
			"fact_id":     factID,
			"user_id":     userID,
			"fact_type":   factType,
			"source_conv": sourceConv,
			"is_active":   isActive,
		}),
	}

	_, err := m.QdrantCli.Client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: m.QdrantCli.Collection,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("qdrant upsert fact vector: %w", err)
	}
	return nil
}

func (m *DBManager) SearchRelevantFacts(
	ctx context.Context,
	userID int64,
	queryVector []float32,
	limit uint64,
) ([]*qdrant.ScoredPoint, error) {
	if limit == 0 {
		limit = 5
	}

	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatchInt("user_id", userID),
			qdrant.NewMatchBool("is_active", true),
		},
	}

	resp, err := m.QdrantCli.Client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: m.QdrantCli.Collection,
		Query:          qdrant.NewQuery(queryVector...),
		Filter:         filter,
		Limit:          &limit,
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant search relevant facts: %w", err)
	}

	return resp, nil
}

func (m *DBManager) RetrieveRelevantFacts(
	ctx context.Context,
	userID int64,
	query string,
	limit uint64,
) ([]*model.UserFact, error) {
	if query == "" {
		return nil, nil
	}
	if limit == 0 {
		limit = 5
	}
	if global.MultiModalEmbedder == nil {
		return nil, errors.New("nil multimodal embedder")
	}

	vector, err := global.MultiModalEmbedder.EmbedText(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(vector) == 0 {
		return nil, errors.New("empty query embedding result")
	}

	points, err := m.SearchRelevantFacts(ctx, userID, Float64To32(vector), limit)
	if err != nil {
		return nil, err
	}

	res := make([]*model.UserFact, 0, len(points))
	for _, p := range points {
		if p.Payload == nil {
			continue
		}

		val, ok := p.Payload["fact_id"]
		if !ok || val == nil {
			continue
		}

		factID := val.GetIntegerValue()
		fact, err := m.GetFactByID(ctx, factID)
		if err != nil {
			continue
		}
		if fact != nil {
			res = append(res, fact)
		}
	}

	return res, nil
}
