package initialize

import (
	"context"
	"liveclass/internal/rpc/quiz/global"

	"github.com/go-redis/redis/v8"
)

func InitRedisDB() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     global.Config.RedisConfig.Addr,
		Password: global.Config.RedisConfig.Password,
		DB:       global.Config.RedisConfig.DB,
	})

	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		panic(err)
	}
	return rdb
}
