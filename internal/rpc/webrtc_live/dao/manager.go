package dao

import (
	"github.com/bwmarrin/snowflake"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

type DBManager struct {
	Node *snowflake.Node
	DB   *gorm.DB
	RDB  *redis.Client
}
