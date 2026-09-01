package cdc

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"

	"liveclass/internal/rpc/agent/memory"
)

func RunOutboxDispatcher(ctx context.Context, dbm *memory.DBManager, writer *kafka.Writer) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := dispatchBatch(ctx, dbm, writer); err != nil {
				log.Printf("[outbox] dispatch batch error: %v", err)
			}
		}
	}
}

func dispatchBatch(ctx context.Context, dbm *memory.DBManager, writer *kafka.Writer) error {
	for {
		events, err := dbm.ListPendingOutbox(ctx, 50)
		if err != nil {
			return err
		}
		if len(events) == 0 {
			return nil
		}

		// 收集可序列化的消息，序列化失败的单独标记 failed
		msgs := make([]kafka.Message, 0, len(events))
		sentIDs := make([]int64, 0, len(events))

		for _, ev := range events {
			value, err := json.Marshal(debeziumEnvelope{Payload: ev.Payload})
			if err != nil {
				_ = dbm.MarkOutboxFailed(ctx, ev.ID, err.Error())
				continue
			}
			msgs = append(msgs, kafka.Message{
				Key:   []byte(ev.BizKey),
				Value: value,
				Headers: []kafka.Header{
					{Key: "event_type", Value: []byte(ev.EventType)},
				},
			})
			sentIDs = append(sentIDs, ev.ID)
		}

		if len(msgs) == 0 {
			continue
		}

		// 批量写入，一次网络往返
		if err := writer.WriteMessages(ctx, msgs...); err != nil {
			// 批量失败：全部标记 failed，下次重试
			for _, id := range sentIDs {
				_ = dbm.MarkOutboxFailed(ctx, id, err.Error())
			}
			log.Printf("[outbox] batch write failed count=%d err=%v", len(msgs), err)
			// 退出本批次，交给外层 2s ticker 再调度，避免 Kafka 故障时形成紧密重试循环。
			return err
		}

		for _, id := range sentIDs {
			if err := dbm.MarkOutboxSent(ctx, id); err != nil {
				log.Printf("[outbox] mark sent failed id=%d err=%v", id, err)
			}
		}
	}
}
