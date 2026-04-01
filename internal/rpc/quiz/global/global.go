package global

import (
	"liveclass/internal/rpc/quiz/config"

	"github.com/go-redis/redis/v8"
)

var (
	Config      *config.Config
	RedisClient *redis.Client
)
