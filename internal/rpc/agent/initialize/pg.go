package initialize

import (
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitPGDB() *gorm.DB {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=%s TimeZone=%s",
		global.Config.PostgresConfig.Host,
		global.Config.PostgresConfig.User,
		global.Config.PostgresConfig.Password,
		global.Config.PostgresConfig.DB,
		global.Config.PostgresConfig.Port,
		global.Config.PostgresConfig.SSLMode,
		global.Config.PostgresConfig.TimeZone,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("postgres init success")

	return db
}
