package initialize

import (
	"liveclass/internal/rpc/agent/global"
	"time"

	"github.com/segmentio/kafka-go"
)

func InitKafkaReader() *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{"127.0.0.1:9092"},
		Topic:          global.Config.KafkaTopic,
		GroupID:        "fact-indexer-group",
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 0,
	})
	return reader
}
