package global

import (
	"liveclass/internal/rpc/chat/config"

	"github.com/go-redis/redis/v8"
)

var (
	Config      *config.Config
	RedisClient *redis.Client
)
