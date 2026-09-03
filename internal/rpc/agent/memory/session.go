package memory

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"liveclass/internal/rpc/agent/model"
)

func (m *DBManager) AppendTranscriptEvent(ctx context.Context, event *model.TranscriptEvent) error {
	if m == nil || m.DB == nil {
		return errors.New("nil db")
	}
	if event == nil || event.SessionID == "" || event.RequestID == "" || event.EventKey == "" || event.EventType == "" {
		return errors.New("invalid transcript event")
	}
	return postgresWriteError(ctx, "append_transcript", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "session_id"}, {Name: "request_id"}, {Name: "event_key"}},
			DoNothing: true,
		}).Create(event).Error
	})
}

func (m *DBManager) ListTranscriptEvents(ctx context.Context, sessionID string, afterID int64) ([]model.TranscriptEvent, error) {
	if m == nil || m.DB == nil {
		return nil, errors.New("nil db")
	}
	return postgresRead(ctx, "list_transcript", func(callCtx context.Context) ([]model.TranscriptEvent, error) {
		var events []model.TranscriptEvent
		q := m.DB.WithContext(callCtx).Where("session_id = ?", sessionID)
		if afterID > 0 {
			q = q.Where("id > ?", afterID)
		}
		err := q.Order("id asc").Find(&events).Error
		return events, err
	})
}

func (m *DBManager) GetTranscriptEvent(ctx context.Context, sessionID, requestID, eventKey string) (*model.TranscriptEvent, error) {
	if m == nil || m.DB == nil {
		return nil, errors.New("nil db")
	}
	event, err := postgresRead(ctx, "get_transcript_event", func(callCtx context.Context) (model.TranscriptEvent, error) {
		var event model.TranscriptEvent
		err := m.DB.WithContext(callCtx).Where("session_id = ? AND request_id = ? AND event_key = ?", sessionID, requestID, eventKey).First(&event).Error
		return event, err
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &event, nil
}

func (m *DBManager) LatestSummaryCheckpoint(ctx context.Context, sessionID string) (*model.SummaryCheckpoint, error) {
	if m == nil || m.DB == nil {
		return nil, errors.New("nil db")
	}
	cp, err := postgresRead(ctx, "latest_summary_checkpoint", func(callCtx context.Context) (model.SummaryCheckpoint, error) {
		var cp model.SummaryCheckpoint
		err := m.DB.WithContext(callCtx).Where("session_id = ?", sessionID).
			Order("source_event_end_id desc, id desc").First(&cp).Error
		return cp, err
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cp, nil
}

func (m *DBManager) SaveSummaryCheckpoint(ctx context.Context, cp *model.SummaryCheckpoint) error {
	if m == nil || m.DB == nil {
		return errors.New("nil db")
	}
	if cp == nil || cp.SessionID == "" || cp.SourceEventEndID <= 0 {
		return errors.New("invalid summary checkpoint")
	}
	return postgresWriteError(ctx, "save_summary_checkpoint", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "session_id"}, {Name: "source_event_end_id"}},
			DoNothing: true,
		}).Create(cp).Error
	})
}

func (m *DBManager) AppendTraceEvent(ctx context.Context, event *model.AgentTraceEvent) error {
	if m == nil || m.DB == nil {
		return errors.New("nil db")
	}
	if event == nil || event.RunID == "" || event.EventType == "" {
		return errors.New("invalid trace event")
	}
	return postgresWriteError(ctx, "append_agent_trace", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Create(event).Error
	})
}
