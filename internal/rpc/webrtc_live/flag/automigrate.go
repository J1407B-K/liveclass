package flag

import (
	"gorm.io/gorm"
	"liveclass/internal/rpc/webrtc_live/model"
	"log"
)

func MysqlAutoMigrate(db *gorm.DB) {
	err := db.Set("gorm:table_options", "ENGINE=InnoDB").
		AutoMigrate(
			&model.WebrtcLesson{},
		)
	if err != nil {
		log.Println("建表失败")
		return
	}
	log.Println("建表成功")
}
