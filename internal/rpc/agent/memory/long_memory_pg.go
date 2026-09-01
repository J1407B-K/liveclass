package memory

import (
	"context"
	"errors"

	"liveclass/internal/rpc/agent/model"
)

func (m *DBManager) InsertFact(
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

	f := &model.UserFact{
		UserID:     userID,
		FactType:   factType,
		Content:    content,
		Confidence: confidence,
		SourceConv: sourceConv,
		IsActive:   true,
	}

	if err := postgresWriteError(ctx, "insert_fact", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Create(f).Error
	}); err != nil {
		return nil, err
	}
	return f, nil
}

func (m *DBManager) GetFactByID(ctx context.Context, id int64) (*model.UserFact, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}

	f, err := postgresRead(ctx, "get_fact_by_id", func(callCtx context.Context) (model.UserFact, error) {
		var f model.UserFact
		err := m.DB.WithContext(callCtx).First(&f, id).Error
		return f, err
	})
	if err != nil {
		return nil, err
	}
	return &f, nil
}

func (m *DBManager) UpdateFactIndexStatus(ctx context.Context, factID int64, status string) error {
	if m.DB == nil {
		return errors.New("nil db")
	}

	return postgresWriteError(ctx, "update_fact_index_status", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Model(&model.UserFact{}).
			Where("id = ?", factID).Update("index_status", status).Error
	})
}
