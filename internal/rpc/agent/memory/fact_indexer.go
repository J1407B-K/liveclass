package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"liveclass/internal/rpc/agent/model"

	"gorm.io/gorm"
)

func (m *DBManager) InsertFactWithOutbox(
	ctx context.Context,
	userID int64,
	factType string,
	content string,
	confidence float64,
	sourceConv string,
) (*model.UserFact, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}

	var fact model.UserFact

	factResult, err := postgresWrite(ctx, "insert_fact_with_outbox", func(callCtx context.Context) (model.UserFact, error) {
		err := m.DB.WithContext(callCtx).Transaction(func(tx *gorm.DB) error {
			fact = model.UserFact{
				UserID:      userID,
				FactType:    factType,
				Content:     content,
				Confidence:  confidence,
				SourceConv:  sourceConv,
				IsActive:    true,
				IndexStatus: "pending",
			}

			if err := tx.Create(&fact).Error; err != nil {
				return err
			}

			payload := map[string]interface{}{
				"fact_id":     fact.ID,
				"user_id":     fact.UserID,
				"fact_type":   fact.FactType,
				"content":     fact.Content,
				"source_conv": fact.SourceConv,
				"is_active":   fact.IsActive,
			}

			bytes, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			event := model.OutboxEvent{
				EventType:     "user_fact_created",
				AggregateType: "user_fact",
				AggregateID:   strconv.FormatInt(fact.ID, 10),
				BizKey:        fmt.Sprintf("fact:%d", fact.ID),
				Payload:       string(bytes),
				Status:        0,
			}

			if err = tx.Create(&event).Error; err != nil {
				return err
			}

			return nil
		})
		return fact, err
	})

	if err != nil {
		return nil, err
	}

	return &factResult, nil
}
