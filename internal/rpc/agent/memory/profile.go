package memory

import (
	"context"
	"errors"
	"liveclass/internal/rpc/agent/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultProfileLimit = 12

// GetUserProfile returns cached profile summary for a user.
func (m *DBManager) GetUserProfile(ctx context.Context, userID int64) (*model.UserProfile, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}

	var profile model.UserProfile
	err := m.DB.WithContext(ctx).Where("user_id = ?", userID).First(&profile).Error
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

	return m.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"summary":    summary,
			"updated_at": gorm.Expr("NOW()"),
		}),
	}).Create(&profile).Error
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

	query := m.DB.WithContext(ctx).
		Where("user_id = ? AND is_active = ?", userID, true)

	if minConfidence > 0 {
		query = query.Where("confidence >= ?", minConfidence)
	}

	var facts []*model.UserFact
	if err := query.
		Order("updated_at DESC, confidence DESC").
		Limit(limit).
		Find(&facts).Error; err != nil {
		return nil, err
	}

	return facts, nil
}
