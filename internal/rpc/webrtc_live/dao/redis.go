package dao

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

const BloomKey = "bloom:lesson"

func HandsKey(lessonID int64) string {
	return fmt.Sprintf("live:room:%d:hands", lessonID)
}

func LiveCountKey(lessonID int64) string   { return fmt.Sprintf("live:room:%d:count", lessonID) }
func LiveMembersKey(lessonID int64) string { return fmt.Sprintf("live:room:%d:members", lessonID) }

func (m *DBManager) AddBloom(ctx context.Context, lessonID int64) error {
	_, err := m.RDB.Do(ctx, "BF.ADD", BloomKey, lessonID).Result()
	return err
}

func (m *DBManager) BloomMaybeLesson(ctx context.Context, lessonID int64) (maybe bool, err error) {
	v, err := m.RDB.Do(ctx, "BF.EXISTS", BloomKey, lessonID).Result()
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
func MAddBloom(ctx context.Context, rdb *redis.Client, lessonIDs []int64) error {
	args := make([]interface{}, 0, 2+len(lessonIDs))
	args = append(args, "BF.MADD", BloomKey)
	for _, id := range lessonIDs {
		args = append(args, id)
	}
	_, err := rdb.Do(ctx, args...).Result()
	return err
}
