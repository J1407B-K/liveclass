package memory

import (
	"context"
	"errors"
	"time"

	"liveclass/internal/rpc/agent/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultProfileLimit = 12

// GetUserProfile returns cached profile summary for a user.
func (m *DBManager) GetUserProfile(ctx context.Context, userID int64) (*model.UserProfile, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}

	profile, err := postgresRead(ctx, "get_user_profile", func(callCtx context.Context) (model.UserProfile, error) {
		var profile model.UserProfile
		err := m.DB.WithContext(callCtx).Where("user_id = ?", userID).First(&profile).Error
		return profile, err
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}

// UpsertUserProfile stores/updates user profile summary with the latest timestamp.
func (m *DBManager) UpsertUserProfile(ctx context.Context, userID int64, summary string) error {
	if m.DB == nil {
		return errors.New("nil db")
	}
	if userID == 0 {
		return errors.New("invalid user id")
	}

	now := time.Now()
	profile := model.UserProfile{
		UserID:    userID,
		Summary:   summary,
		UpdatedAt: now,
		CreatedAt: now,
	}

	return postgresWriteError(ctx, "upsert_user_profile", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}},
			DoUpdates: clause.Assignments(map[string]any{
				"summary":    summary,
				"updated_at": gorm.Expr("NOW()"),
			}),
		}).Create(&profile).Error
	})
}

// ListFactsForProfile fetches recent high-confidence facts for building profile summary.
func (m *DBManager) ListFactsForProfile(
	ctx context.Context,
	userID int64,
	limit int,
	minConfidence float64,
) ([]*model.UserFact, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}

	if limit <= 0 {
		limit = defaultProfileLimit
	}

	facts, err := postgresRead(ctx, "list_facts_for_profile", func(callCtx context.Context) ([]*model.UserFact, error) {
		query := m.DB.WithContext(callCtx).Where("user_id = ? AND is_active = ?", userID, true)
		if minConfidence > 0 {
			query = query.Where("confidence >= ?", minConfidence)
		}
		var facts []*model.UserFact
		err := query.Order("updated_at DESC, confidence DESC").Limit(limit).Find(&facts).Error
		return facts, err
	})
	if err != nil {
		return nil, err
	}

	return facts, nil
}
