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

		for _, ev := range events {
			env := debeziumEnvelope{Payload: ev.Payload}
			value, err := json.Marshal(env)
			if err != nil {
				_ = dbm.MarkOutboxFailed(ctx, ev.ID, err.Error())
				continue
			}

			msg := kafka.Message{
				Key:   []byte(ev.BizKey),
				Value: value,
				Headers: []kafka.Header{
					{
						Key:   "event_type",
						Value: []byte(ev.EventType),
					},
				},
			}

			if err := writer.WriteMessages(ctx, msg); err != nil {
				_ = dbm.MarkOutboxFailed(ctx, ev.ID, err.Error())
				continue
			}

			if err := dbm.MarkOutboxSent(ctx, ev.ID); err != nil {
				log.Printf("[outbox] mark sent failed id=%d err=%v", ev.ID, err)
			}
		}
	}
}
