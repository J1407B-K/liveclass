package flag

import (
	"gorm.io/gorm"
	"liveclass/internal/rpc/quiz/model"
	"log"
)

func MysqlAutoMigrate(db *gorm.DB) {
	err := db.Set("gorm:table_options", "ENGINE=InnoDB").
		AutoMigrate(
			&model.Question{},
			&model.Answer{},
		)
	if err != nil {
		log.Println("建表失败")
		return
	}
	log.Println("建表成功")
}
