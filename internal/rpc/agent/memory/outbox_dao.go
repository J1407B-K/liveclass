package memory

import (
	"context"
	"errors"
	"liveclass/internal/rpc/agent/model"
	"time"

	"gorm.io/gorm"
)

// CreateOutboxEvent inserts a new outbox event.
func (m *DBManager) CreateOutboxEvent(ctx context.Context, ev model.OutboxEvent) error {
	if m.DB == nil {
		return errors.New("nil db")
	}
	return postgresWriteError(ctx, "create_outbox_event", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Create(&ev).Error
	})
}

// MarkOutboxSent updates event status to sent.
func (m *DBManager) MarkOutboxSent(ctx context.Context, id int64) error {
	if m.DB == nil {
		return errors.New("nil db")
	}
	return postgresWriteError(ctx, "mark_outbox_sent", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Model(&model.OutboxEvent{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":     1,
			"last_error": "",
			"updated_at": time.Now(),
		}).Error
	})
}

// MarkOutboxFailed updates event status and error message.
func (m *DBManager) MarkOutboxFailed(ctx context.Context, id int64, errMsg string) error {
	if m.DB == nil {
		return errors.New("nil db")
	}
	return postgresWriteError(ctx, "mark_outbox_failed", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Model(&model.OutboxEvent{}).Where("id = ?", id).Updates(map[string]interface{}{
			"status":      2,
			"last_error":  errMsg,
			"retry_count": gorm.Expr("retry_count + 1"),
			"updated_at":  time.Now(),
		}).Error
	})
}

// ListPendingOutbox returns events with pending/failed status, limited by size.
func (m *DBManager) ListPendingOutbox(ctx context.Context, limit int) ([]model.OutboxEvent, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}
	if limit <= 0 {
		limit = 100
	}
	return postgresRead(ctx, "list_pending_outbox", func(callCtx context.Context) ([]model.OutboxEvent, error) {
		var events []model.OutboxEvent
		err := m.DB.WithContext(callCtx).Where("status IN (?)", []int32{0, 2}).
			Order("id ASC").Limit(limit).Find(&events).Error
		return events, err
	})
}
