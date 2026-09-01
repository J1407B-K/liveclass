package memory

import (
	"context"
	"errors"
	"liveclass/internal/rpc/agent/model"
	"strings"

	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
)

func (m *DBManager) GetMessageByRequestIDAndRole(
	ctx context.Context,
	convID string,
	requestID string,
	role string,
) (*model.Message, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}
	if convID == "" {
		return nil, errors.New("empty convID")
	}
	if requestID == "" {
		return nil, errors.New("empty requestID")
	}

	msg, err := postgresRead(ctx, "get_message_by_request", func(callCtx context.Context) (model.Message, error) {
		var msg model.Message
		err := m.DB.WithContext(callCtx).
			Where("conv_id = ? AND request_id = ? AND role = ?", convID, requestID, role).
			First(&msg).Error
		return msg, err
	})

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (m *DBManager) GetAssistantMessageByRequestID(
	ctx context.Context,
	convID string,
	requestID string,
) (*schema.Message, error) {
	record, err := m.GetMessageByRequestIDAndRole(ctx, convID, requestID, string(schema.Assistant))
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, nil
	}

	return &schema.Message{
		Role:    schema.Assistant,
		Content: record.Content,
	}, nil
}

func Float64To32(v []float64) []float32 {
	out := make([]float32, len(v))
	for i := range v {
		out[i] = float32(v[i])
	}
	return out
}

func FormatFactsForPrompt(facts []*model.UserFact) string {
	if len(facts) == 0 {
		return ""
	}

	var b strings.Builder
	for _, f := range facts {
		if f == nil || f.Content == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(f.Content)
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String())
}
