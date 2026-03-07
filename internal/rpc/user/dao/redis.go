package dao

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

const BloomKey = "bloom:user"

func (m *DBManager) AddBloom(ctx context.Context, userid int64) error {
	_, err := m.RDB.Do(ctx, "BF.ADD", BloomKey, userid).Result()
	return err
}

func (m *DBManager) BloomMaybeUser(ctx context.Context, userid int64) (maybe bool, err error) {
	v, err := m.RDB.Do(ctx, "BF.EXISTS", BloomKey, userid).Result()
	if err != nil {
		return false, err
	}
	switch vv := v.(type) {
	case int64:
		return vv == 1, nil
	case bool:
		return vv, nil
	default:
		return false, fmt.Errorf("unexpected BF.EXISTS resp=%T", v)
	}
}

func MAddBloom(ctx context.Context, rdb *redis.Client, key string, userids []int64) error {
	args := make([]interface{}, 0, 2+len(userids))
	args = append(args, "BF.MADD", key)
	for _, id := range userids {
		args = append(args, id)
	}
	_, err := rdb.Do(ctx, args...).Result()
	return err
}
