package initialize

import (
	"liveclass/internal/rpc/chat/global"
	"time"

	"github.com/go-redis/redis/v8"
)

func InitRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         global.Config.RedisAddr,
		Password:     global.Config.RedisPassword,
		DB:           0,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  300 * time.Millisecond,
		WriteTimeout: 300 * time.Millisecond,
		PoolTimeout:  500 * time.Millisecond,
		MaxRetries:   -1,
	})
}
