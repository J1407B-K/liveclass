package memory

import (
	"context"
	"errors"
	"fmt"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/model"
	"time"

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

	_, err := dependency.Do(ctx, dependency.Qdrant, "upsert_fact", func(callCtx context.Context) (*qdrant.UpdateResult, error) {
		return m.QdrantCli.Client.Upsert(callCtx, &qdrant.UpsertPoints{
			CollectionName: m.QdrantCli.Collection,
			Points:         []*qdrant.PointStruct{point},
		})
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
	return m.SearchRelevantFactsWithOptions(ctx, userID, queryVector, FactRetrievalOptions{TopK: limit})
}

type FactRetrievalOptions struct {
	TopK           uint64
	ScoreThreshold float32
	TokenBudget    int
	FactTypes      []string
	Since          time.Time
	Until          time.Time
}

func (m *DBManager) SearchRelevantFactsWithOptions(ctx context.Context, userID int64, queryVector []float32, options FactRetrievalOptions) ([]*qdrant.ScoredPoint, error) {
	limit := options.TopK
	if limit == 0 {
		limit = 5
	}

	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			qdrant.NewMatchInt("user_id", userID),
			qdrant.NewMatchBool("is_active", true),
		},
	}
	if len(options.FactTypes) == 1 {
		filter.Must = append(filter.Must, qdrant.NewMatchKeyword("fact_type", options.FactTypes[0]))
	} else if len(options.FactTypes) > 1 {
		filter.Must = append(filter.Must, qdrant.NewMatchKeywords("fact_type", options.FactTypes...))
	}

	resp, err := dependency.Do(ctx, dependency.Qdrant, "search_facts", func(callCtx context.Context) ([]*qdrant.ScoredPoint, error) {
		return m.QdrantCli.Client.Query(callCtx, &qdrant.QueryPoints{
			CollectionName: m.QdrantCli.Collection,
			Query:          qdrant.NewQuery(queryVector...),
			Filter:         filter,
			Limit:          &limit,
			ScoreThreshold: scoreThresholdPointer(options.ScoreThreshold),
			WithPayload:    qdrant.NewWithPayload(true),
		})
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant search relevant facts: %w", err)
	}

	return resp, nil
}

func scoreThresholdPointer(value float32) *float32 {
	if value <= 0 {
		return nil
	}
	return &value
}

func (m *DBManager) RetrieveRelevantFacts(
	ctx context.Context,
	userID int64,
	query string,
	limit uint64,
) ([]*model.UserFact, error) {
	return m.RetrieveRelevantFactsWithOptions(ctx, userID, query, FactRetrievalOptions{TopK: limit, ScoreThreshold: 0.2, TokenBudget: 2000})
}

func (m *DBManager) RetrieveRelevantFactsWithOptions(ctx context.Context, userID int64, query string, options FactRetrievalOptions) ([]*model.UserFact, error) {
	limit := options.TopK
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

	options.TopK = limit
	points, err := m.SearchRelevantFactsWithOptions(ctx, userID, Float64To32(vector), options)
	if err != nil {
		return nil, err
	}

	res := make([]*model.UserFact, 0, len(points))
	seen := make(map[string]struct{})
	usedTokens := 0
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
		if fact != nil && fact.IsActive {
			if !options.Since.IsZero() && fact.CreatedAt.Before(options.Since) {
				continue
			}
			if !options.Until.IsZero() && fact.CreatedAt.After(options.Until) {
				continue
			}
			key := fact.FactType + "\x00" + fact.Content
			if _, exists := seen[key]; exists {
				continue
			}
			cost := estimateFactTokens(fact.Content)
			if options.TokenBudget > 0 && usedTokens+cost > options.TokenBudget {
				continue
			}
			seen[key] = struct{}{}
			usedTokens += cost
			res = append(res, fact)
		}
	}

	return res, nil
}

func (m *DBManager) RetrieveSemanticFacts(ctx context.Context, userID int64, query string, limit uint64) ([]*model.UserFact, error) {
	return m.RetrieveRelevantFactsWithOptions(ctx, userID, query, FactRetrievalOptions{
		TopK: limit, ScoreThreshold: 0.2, TokenBudget: 1600,
		FactTypes: []string{"project", "identity", "preference", "habit", "goal", "background", "skill"},
	})
}

func (m *DBManager) RetrieveEpisodicMemory(ctx context.Context, userID int64, query string, since, until time.Time, limit uint64) ([]*model.UserFact, error) {
	return m.RetrieveRelevantFactsWithOptions(ctx, userID, query, FactRetrievalOptions{
		TopK: limit, ScoreThreshold: 0.2, TokenBudget: 800, FactTypes: []string{"episodic"}, Since: since, Until: until,
	})
}

func estimateFactTokens(value string) int {
	n := len([]rune(value))
	if n == 0 {
		return 0
	}
	return (n + 1) / 2
}
