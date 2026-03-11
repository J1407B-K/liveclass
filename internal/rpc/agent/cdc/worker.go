package cdc

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"

	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
)

func RunFactIndexerWorker(ctx context.Context, r *kafka.Reader, dbm *memory.DBManager) error {
	defer func() {
		if err := r.Close(); err != nil {
			log.Printf("[fact-indexer] close reader failed: %v", err)
		}
	}()

	log.Println("[fact-indexer] worker started")

	for {
		msg, err := r.FetchMessage(ctx)
		if err != nil {
			return err
		}

		log.Printf("[fact-indexer] recv topic=%s partition=%d offset=%d key=%s",
			msg.Topic, msg.Partition, msg.Offset, string(msg.Key))

		var env debeziumEnvelope
		if err := json.Unmarshal(msg.Value, &env); err != nil {
			log.Printf("[fact-indexer] unmarshal envelope failed offset=%d err=%v raw=%s", msg.Offset, err, string(msg.Value))
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		if env.Payload == "" {
			log.Printf("[fact-indexer] empty envelope payload offset=%d raw=%s", msg.Offset, string(msg.Value))
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		var e FactCreatedEvent
		if err := json.Unmarshal([]byte(env.Payload), &e); err != nil {
			log.Printf("[fact-indexer] unmarshal event payload failed offset=%d err=%v payload=%s", msg.Offset, err, env.Payload)
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		if e.FactID <= 0 || e.UserID <= 0 || e.FactType == "" || e.Content == "" {
			log.Printf("[fact-indexer] invalid event offset=%d event=%+v", msg.Offset, e)
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		log.Printf("[fact-indexer] parsed event offset=%d fact_id=%d user_id=%d type=%s",
			msg.Offset, e.FactID, e.UserID, e.FactType)

		vector, err := global.MultiModalEmbedder.EmbedText(ctx, e.Content)
		if err != nil {
			log.Printf("[fact-indexer] embed failed offset=%d fact_id=%d err=%v", msg.Offset, e.FactID, err)
			_ = dbm.UpdateFactIndexStatus(context.Background(), e.FactID, "failed")
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		if len(vector) == 0 {
			log.Printf("[fact-indexer] empty embedding offset=%d fact_id=%d", msg.Offset, e.FactID)
			_ = dbm.UpdateFactIndexStatus(context.Background(), e.FactID, "failed")
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		log.Printf("[fact-indexer] embedding dim=%d fact_id=%d", len(vector), e.FactID)

		err = dbm.UpsertFactVector(
			ctx,
			e.FactID,
			e.UserID,
			e.FactType,
			e.SourceConv,
			e.IsActive,
			memory.Float64To32(vector),
		)
		if err != nil {
			log.Printf("[fact-indexer] qdrant upsert failed offset=%d fact_id=%d err=%v", msg.Offset, e.FactID, err)
			_ = dbm.UpdateFactIndexStatus(context.Background(), e.FactID, "failed")
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		log.Printf("[fact-indexer] qdrant upsert success fact_id=%d", e.FactID)

		if err := dbm.UpdateFactIndexStatus(ctx, e.FactID, "done"); err != nil {
			log.Printf("[fact-indexer] update status done failed offset=%d fact_id=%d err=%v", msg.Offset, e.FactID, err)
			_ = r.CommitMessages(ctx, msg)
			continue
		}

		if err := r.CommitMessages(ctx, msg); err != nil {
			log.Printf("[fact-indexer] commit failed offset=%d fact_id=%d err=%v", msg.Offset, e.FactID, err)
		}
	}
}
