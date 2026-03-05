package model

import "github.com/go-redis/redis/v8"

type DBManager struct {
	RDB *redis.Client
}
