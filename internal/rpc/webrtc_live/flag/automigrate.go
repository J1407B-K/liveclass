package flag

import (
	"liveclass/internal/rpc/webrtc_live/model"
	"log"

	"gorm.io/gorm"
)

func MysqlAutoMigrate(db *gorm.DB) {
	err := db.Set("gorm:table_options", "ENGINE=InnoDB").
		AutoMigrate(
			&model.WebrtcLesson{},
			&model.SignIn{},
			&model.ExcalidrawDoc{},
			&model.LessonStudent{},
			&model.OutboxEvent{},
		)
	if err != nil {
		log.Println("建表失败")
		return
	}
	log.Println("建表成功")
}
