package initialize

import (
	"liveclass/internal/rpc/chat/global"

	"github.com/go-redis/redis/v8"
)

func InitRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     global.Config.RedisAddr,
		Password: global.Config.RedisPassword,
		DB:       0,
	})
}
