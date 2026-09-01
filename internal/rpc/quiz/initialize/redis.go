package initialize

import (
	"context"
	"liveclass/internal/rpc/quiz/global"
	"time"

	"github.com/go-redis/redis/v8"
)

func InitRedisDB() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:         global.Config.RedisConfig.Addr,
		Password:     global.Config.RedisConfig.Password,
		DB:           global.Config.RedisConfig.DB,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  300 * time.Millisecond,
		WriteTimeout: 300 * time.Millisecond,
		PoolTimeout:  500 * time.Millisecond,
		MaxRetries:   -1,
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}
	return rdb
}
