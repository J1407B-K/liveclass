package flag

import (
	"liveclass/internal/rpc/agent/model"
	"log"

	"gorm.io/gorm"
)

func PGAutoMigrate(db *gorm.DB) {
	err := db.
		AutoMigrate(
			&model.Conversation{},
			&model.Message{},
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
