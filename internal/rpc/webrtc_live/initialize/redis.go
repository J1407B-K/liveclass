package initialize

import (
	"context"
	"github.com/go-redis/redis/v8"
	"liveclass/internal/rpc/webrtc_live/global"
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
