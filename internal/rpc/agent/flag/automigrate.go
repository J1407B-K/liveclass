package flag

import (
	"liveclass/internal/rpc/agent/model"
	"log"

	"gorm.io/gorm"
)

func PGAutoMigrate(db *gorm.DB) {
	if err := prepareLegacyAgentSchema(db); err != nil {
		log.Printf("迁移前兼容处理失败: %v", err)
		return
	}
	err := db.
		AutoMigrate(
			&model.Conversation{},
			&model.Message{},
			&model.TranscriptEvent{},
			&model.SummaryCheckpoint{},
			&model.AgentTraceEvent{},
			&model.TaskPlan{},
			&model.TaskStep{},
			&model.UserFact{},
			&model.UserProfile{},
			&model.OutboxEvent{},
		)
	if err != nil {
		log.Println("建表失败")
		return
	}
	log.Println("建表成功")
}

func prepareLegacyAgentSchema(db *gorm.DB) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	if !db.Migrator().HasTable(&model.OutboxEvent{}) || db.Migrator().HasColumn(&model.OutboxEvent{}, "UpdatedAt") {
		return nil
	}
	statements := []string{
		`ALTER TABLE outbox_events ADD COLUMN IF NOT EXISTS updated_at timestamptz`,
		`UPDATE outbox_events SET updated_at = COALESCE(created_at, NOW()) WHERE updated_at IS NULL`,
		`ALTER TABLE outbox_events ALTER COLUMN updated_at SET NOT NULL`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return err
		}
	}
	return nil
}
