package initialize

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"liveclass/internal/rpc/live/global"
	"log"
)

func InitGormDB() *gorm.DB {
	dsn := global.Config.MysqlConfig.Username + ":" + global.Config.MysqlConfig.Password + "@tcp(" + global.Config.MysqlConfig.Addr + ")/" + global.Config.MysqlConfig.DB + "?charset=utf8mb4&collation=utf8mb4_unicode_ci&parseTime=True&loc=Local"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("mysql init success")
	return db
}
