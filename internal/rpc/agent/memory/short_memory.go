package memory

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"liveclass/internal/rpc/agent/model"
)

const defaultWindow = 6

func BuildConvID(userID int64, convID string) string {
	convID = strings.TrimSpace(convID)
	if convID != "" {
		return convID
	}
	return fmt.Sprintf("user_%d_default", userID)
}

func BuildRequestID(requestID string) string {
	requestID = strings.TrimSpace(requestID)
	return requestID
}

// EnsureConversation 确保会话存在；不存在则创建，存在则刷新 updated_at
func (m *DBManager) EnsureConversation(ctx context.Context, userID int64, convID string) error {
	if m.DB == nil {
		return errors.New("nil db")
	}
	if convID == "" {
		return errors.New("empty convID")
	}

	conv := model.Conversation{
		UserID: userID,
		ConvID: convID,
	}

	return postgresWriteError(ctx, "ensure_conversation", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "conv_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"updated_at": gorm.Expr("NOW()"),
			}),
		}).
			Create(&conv).Error
	})
}

// AppendMessage 幂等追加一条消息到短期记忆
func (m *DBManager) AppendMessage(
	ctx context.Context,
	userID int64,
	convID string,
	requestID string,
	msg *schema.Message,
) error {
	if m.DB == nil {
		return errors.New("nil db")
	}
	if msg == nil {
		return errors.New("nil message")
	}
	if convID == "" {
		return errors.New("empty convID")
	}
	if requestID == "" {
		return errors.New("empty requestID")
	}

	return postgresWriteError(ctx, "append_message", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Transaction(func(tx *gorm.DB) error {
			conv := model.Conversation{
				UserID: userID,
				ConvID: convID,
			}

			if err := tx.
				Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "conv_id"}},
					DoUpdates: clause.Assignments(map[string]interface{}{
						"updated_at": gorm.Expr("NOW()"),
					}),
				}).
				Create(&conv).Error; err != nil {
				return err
			}

			record := model.Message{
				UserID:    userID,
				ConvID:    convID,
				RequestID: requestID,
				Role:      string(msg.Role),
				Content:   msg.Content,
			}

			return tx.
				Clauses(clause.OnConflict{
					Columns: []clause.Column{
						{Name: "conv_id"},
						{Name: "request_id"},
						{Name: "role"},
					},
					DoNothing: true,
				}).
				Create(&record).Error
		})
	})
}

// ExistsMessage 用于判断某个 request_id + role 是否已存在
func (m *DBManager) ExistsMessage(
	ctx context.Context,
	convID string,
	requestID string,
	role schema.RoleType,
) (bool, error) {
	if m.DB == nil {
		return false, errors.New("nil db")
	}
	if convID == "" {
		return false, errors.New("empty convID")
	}
	if requestID == "" {
		return false, errors.New("empty requestID")
	}

	count, err := postgresRead(ctx, "exists_message", func(callCtx context.Context) (int64, error) {
		var count int64
		err := m.DB.WithContext(callCtx).Model(&model.Message{}).
			Where("conv_id = ? AND request_id = ? AND role = ?", convID, requestID, string(role)).
			Count(&count).Error
		return count, err
	})
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetRecentMessages 获取最近 window 条消息，返回顺序为 旧 -> 新
func (m *DBManager) GetRecentMessages(
	ctx context.Context,
	convID string,
	window int,
) ([]*schema.Message, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}
	if convID == "" {
		return nil, errors.New("empty convID")
	}
	if window <= 0 {
		window = defaultWindow
	}

	rows, err := postgresRead(ctx, "get_recent_messages", func(callCtx context.Context) ([]model.Message, error) {
		var rows []model.Message
		err := m.DB.WithContext(callCtx).Where("conv_id = ?", convID).
			Order("created_at desc, id desc").Limit(window).Find(&rows).Error
		return rows, err
	})
	if err != nil {
		return nil, err
	}

	res := make([]*schema.Message, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		res = append(res, &schema.Message{
			Role:    schema.RoleType(rows[i].Role),
			Content: rows[i].Content,
		})
	}

	return res, nil
}

// GetFullMessages 获取整个会话的完整消息，返回顺序为 旧 -> 新
func (m *DBManager) GetFullMessages(
	ctx context.Context,
	convID string,
) ([]*schema.Message, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}
	if convID == "" {
		return nil, errors.New("empty convID")
	}

	rows, err := postgresRead(ctx, "get_full_messages", func(callCtx context.Context) ([]model.Message, error) {
		var rows []model.Message
		err := m.DB.WithContext(callCtx).Where("conv_id = ?", convID).
			Order("created_at asc, id asc").Find(&rows).Error
		return rows, err
	})
	if err != nil {
		return nil, err
	}

	res := make([]*schema.Message, 0, len(rows))
	for _, row := range rows {
		res = append(res, &schema.Message{
			Role:    schema.RoleType(row.Role),
			Content: row.Content,
		})
	}

	return res, nil
}

// ListConversations 列出某个用户的所有会话 conv_id，按最近更新时间倒序
func (m *DBManager) ListConversations(
	ctx context.Context,
	userID int64,
) ([]string, error) {
	if m.DB == nil {
		return nil, errors.New("nil db")
	}

	rows, err := postgresRead(ctx, "list_conversations", func(callCtx context.Context) ([]model.Conversation, error) {
		var rows []model.Conversation
		err := m.DB.WithContext(callCtx).Where("user_id = ?", userID).
			Order("updated_at desc, id desc").Find(&rows).Error
		return rows, err
	})
	if err != nil {
		return nil, err
	}

	res := make([]string, 0, len(rows))
	for _, row := range rows {
		res = append(res, row.ConvID)
	}
	return res, nil
}

// DeleteConversation 删除一个会话及其全部消息
func (m *DBManager) DeleteConversation(
	ctx context.Context,
	userID int64,
	convID string,
) error {
	if m.DB == nil {
		return errors.New("nil db")
	}
	if convID == "" {
		return errors.New("empty convID")
	}

	return postgresWriteError(ctx, "delete_conversation", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Transaction(func(tx *gorm.DB) error {
			if err := tx.
				Where("conv_id = ?", convID).
				Delete(&model.Message{}).Error; err != nil {
				return err
			}

			if err := tx.
				Where("user_id = ? AND conv_id = ?", userID, convID).
				Delete(&model.Conversation{}).Error; err != nil {
				return err
			}

			return nil
		})
	})
}
