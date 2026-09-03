package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"liveclass/internal/rpc/agent/model"

	"gorm.io/gorm"
)

type FactWrite struct {
	UserID                         int64
	FactType, ConflictKey, Content string
	Confidence                     float64
	Source, SourceConv             string
}

func (m *DBManager) InsertFactWithOutbox(
	ctx context.Context,
	userID int64,
	factType string,
	content string,
	confidence float64,
	sourceConv string,
) (*model.UserFact, error) {
	return m.ResolveFactWithOutbox(ctx, FactWrite{UserID: userID, FactType: factType, Content: content, Confidence: confidence, Source: "conversation", SourceConv: sourceConv})
}

func (m *DBManager) ResolveFactWithOutbox(ctx context.Context, input FactWrite) (*model.UserFact, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}
	input.FactType = strings.ToLower(strings.TrimSpace(input.FactType))
	input.ConflictKey = strings.ToLower(strings.TrimSpace(input.ConflictKey))
	if input.UserID <= 0 || input.FactType == "" || strings.TrimSpace(input.Content) == "" {
		return nil, errors.New("invalid fact")
	}
	if input.Confidence < 0 || input.Confidence > 1 || !allowedFactType(input.FactType) {
		return nil, errors.New("invalid fact type or confidence")
	}
	if input.Source == "" {
		input.Source = "conversation"
	}

	var fact model.UserFact

	factResult, err := postgresWrite(ctx, "insert_fact_with_outbox", func(callCtx context.Context) (model.UserFact, error) {
		err := m.DB.WithContext(callCtx).Transaction(func(tx *gorm.DB) error {
			var superseded *model.UserFact
			active := true
			if input.ConflictKey != "" && (input.FactType == "preference" || input.FactType == "habit") {
				lockKey := input.FactType + ":" + input.ConflictKey
				if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtextextended(?, ?))", lockKey, input.UserID).Error; err != nil {
					return err
				}
				var old model.UserFact
				findErr := tx.Where("user_id = ? AND fact_type = ? AND is_active = true AND metadata->>'conflict_key' = ?", input.UserID, input.FactType, input.ConflictKey).
					Order("confidence desc, updated_at desc, id desc").First(&old).Error
				if findErr == nil {
					if shouldSupersede(old, input) {
						superseded = &old
					} else {
						active = false
					}
				} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
					return findErr
				}
			}
			metadataBytes, _ := json.Marshal(map[string]any{"conflict_key": input.ConflictKey})
			fact = model.UserFact{
				UserID:      input.UserID,
				FactType:    input.FactType,
				Content:     strings.TrimSpace(input.Content),
				Confidence:  input.Confidence,
				Source:      input.Source,
				SourceConv:  input.SourceConv,
				IsActive:    active,
				Metadata:    string(metadataBytes),
				IndexStatus: "pending",
			}
			if superseded != nil {
				fact.Supersedes = &superseded.ID
			}

			if err := tx.Create(&fact).Error; err != nil {
				return err
			}

			if err := createFactOutbox(tx, fact); err != nil {
				return err
			}
			if superseded != nil {
				if err := tx.Model(&model.UserFact{}).Where("id = ?", superseded.ID).Updates(map[string]any{"is_active": false, "index_status": "pending"}).Error; err != nil {
					return err
				}
				superseded.IsActive = false
				if err := createFactOutbox(tx, *superseded); err != nil {
					return err
				}
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

func allowedFactType(value string) bool {
	switch value {
	case "project", "identity", "preference", "habit", "goal", "background", "skill", "episodic":
		return true
	default:
		return false
	}
}

func shouldSupersede(old model.UserFact, input FactWrite) bool {
	if input.Confidence < 0.7 {
		return false
	}
	if input.Confidence >= old.Confidence {
		return true
	}
	return sourcePriority(input.Source) > sourcePriority(old.Source)
}

func sourcePriority(source string) int {
	switch strings.ToLower(source) {
	case "user_explicit":
		return 3
	case "conversation":
		return 2
	case "inferred":
		return 1
	default:
		return 0
	}
}

func createFactOutbox(tx *gorm.DB, fact model.UserFact) error {
	payload := map[string]interface{}{"fact_id": fact.ID, "user_id": fact.UserID, "fact_type": fact.FactType, "content": fact.Content, "source_conv": fact.SourceConv, "is_active": fact.IsActive}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	event := model.OutboxEvent{EventType: "user_fact_upserted", AggregateType: "user_fact", AggregateID: strconv.FormatInt(fact.ID, 10), BizKey: fmt.Sprintf("fact:%d:%t", fact.ID, fact.IsActive), Payload: string(bytes), Status: 0}
	return tx.Create(&event).Error
}
