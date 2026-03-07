package dao

import (
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type DBManager struct {
	DB  *gorm.DB
	RDB *redis.Client
}
