package cdc

import (
	"context"
	"encoding/json"
	"liveclass/internal/rpc/user/dao"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/segmentio/kafka-go"
)

type UserEvent struct {
	Userid int64 `json:"userid"`
}

func RunBloomWorker(ctx context.Context, r *kafka.Reader, rdb *redis.Client) error {
	defer func(r *kafka.Reader) {
		err := r.Close()
		if err != nil {
			log.Printf("Error closing reader: %v", err)
			return
		}
	}(r)

	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			return err
		}

		var e UserEvent
		if err = json.Unmarshal(m.Value, &e); err != nil {
			log.Printf("[bloom-worker] bad message offset=%d err=%v value=%s", m.Offset, err, string(m.Value))
			_ = r.CommitMessages(ctx, m)
			continue
		}

		if _, err = rdb.Do(ctx, "BF.ADD", dao.BloomKey, e.Userid).Result(); err != nil {
			log.Printf("[bloom-worker] BF.ADD failed offset=%d lesson=%d err=%v", m.Offset, e.Userid, err)
			time.Sleep(300 * time.Millisecond)
			continue
		}

		if err = r.CommitMessages(ctx, m); err != nil {
			log.Printf("[bloom-worker] commit failed offset=%d err=%v", m.Offset, err)
		}
	}
}
