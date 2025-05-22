package initialize

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
	_const "liveclass/internal/rpc/agent/const"
	"liveclass/internal/rpc/agent/global"
	"strings"
)

// 采用redis hash存储
func InitRedisStackIndex() error {
	ctx := context.Background()
	rClient := redis.NewClient(&redis.Options{
		Addr:     global.Config.RedisAddr,
		Protocol: 2,
	})

	if err := rClient.Ping(ctx).Err(); err != nil {
		return err
	}

	fullIndex := _const.RedisPrefix + _const.IndexName

	//先check index
	e, err := rClient.Do(ctx, "FT.INFO", fullIndex).Result()
	if err != nil {
		if !strings.Contains(err.Error(), "Unknown index name") {
			return fmt.Errorf("check index failed: %w", err)
		}
		err = nil
	} else if e != nil {
		return nil
	}

	//args参数
	args := []interface{}{
		"FT.CREATE", fullIndex, //创建全文索引
		"ON", "HASH", //索引数据为HASH
		"PREFIX", "1", _const.RedisPrefix, //校验prefix，仅为规范prefix建立索引
		"SCHEMA",                    //字段模式 (希腊奶
		_const.ContentField, "TEXT", //TEXT字段
		_const.MetadataField, "TEXT", // TEXT字段
		_const.VectorField, "VECTOR", "FLAT", //向量字段，使用FLAT索引类型
		"6",               //FLAT索引参数数量(一共6个)
		"TYPE", "FLOAT32", //向量数据类型float32
		"DIM", _const.Dimension, //向量维度
		"DISTANCE_METRIC", "COSINE", //距离度量方式为余弦相似度
	}

	if err := rClient.Do(ctx, args...).Err(); err != nil {
		return fmt.Errorf("create index failed: %w", err)
	}

	if _, err = rClient.Do(ctx, "FT.INFO", fullIndex).Result(); err != nil {
		return fmt.Errorf("failed to verify index creation: %w", err)
	}

	return nil
}
