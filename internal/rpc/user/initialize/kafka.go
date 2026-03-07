package initialize

import (
	"time"

	"github.com/segmentio/kafka-go"
)

func InitKafkaReader() *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{"127.0.0.1:9092"},
		Topic:          "outbox.event.User",
		GroupID:        "bloom-worker",
		MinBytes:       1e3,  // 1KB
		MaxBytes:       10e6, // 10MB
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 0, // 0=手动 commit（配合 FetchMessage）
	})
	return reader
}
