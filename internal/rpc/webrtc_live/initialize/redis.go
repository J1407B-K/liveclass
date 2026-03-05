package initialize

import (
	"context"
	"liveclass/internal/rpc/webrtc_live/dao"
	"liveclass/internal/rpc/webrtc_live/global"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
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

func InitBloom(db *gorm.DB, rdb *redis.Client) error {
	ctx := context.Background()

	exists, err := rdb.Exists(ctx, dao.BloomKey).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		if _, err = rdb.Do(ctx, "BF.RESERVE", dao.BloomKey, 0.01, 100000).Result(); err != nil {
			return err
		}
	}

	ids, err := dao.GetAllLessonIDs(db)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}

	return dao.MAddBloom(ctx, rdb, ids)
}

func InitRuntimeAddMBloom(ctx context.Context, db *gorm.DB, rdb *redis.Client) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshBloom(ctx, db, rdb)
		}
	}
}

func refreshBloom(ctx context.Context, db *gorm.DB, rdb *redis.Client) {
	ids, err := dao.GetAllLessonIDs(db)
	if err != nil {
		log.Printf("GetAllLessonIDs err=%v", err)
		return
	}

	tmpKey := dao.BloomKey + ":tmp"

	rdb.Del(ctx, tmpKey)

	_, err = rdb.Do(ctx, "BF.RESERVE", tmpKey, 0.01, 100000).Result()
	if err != nil {
		log.Println("BF.RESERVE tmp failed:", err)
		return
	}

	const batch = 2000
	for i := 0; i < len(ids); i += batch {
		end := i + batch
		if end > len(ids) {
			end = len(ids)
		}

		if err := dao.MAddBloom(ctx, rdb, ids[i:end]); err != nil {
			log.Println("MAddBloom failed:", err)
			return
		}
	}

	if err = rdb.Rename(ctx, tmpKey, dao.BloomKey).Err(); err != nil {
		log.Println("Rename bloom failed:", err)
	}
}
