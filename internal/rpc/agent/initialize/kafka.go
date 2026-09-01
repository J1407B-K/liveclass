package initialize

import (
	"liveclass/internal/rpc/agent/global"
	"time"

	"github.com/segmentio/kafka-go"
)

func InitKafkaReader() *kafka.Reader {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{global.Config.KafkaBroker},
		Topic:          global.Config.KafkaTopic,
		GroupID:        "fact-indexer-group",
		MinBytes:       1,
		MaxBytes:       10e6,
		MaxWait:        500 * time.Millisecond,
		CommitInterval: 0,
	})
	return reader
}

func InitKafkaWriter() *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(global.Config.KafkaBroker),
		Topic:                  global.Config.KafkaTopic,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
		MaxAttempts:            1,
		WriteTimeout:           2 * time.Second,
		ReadTimeout:            2 * time.Second,
	}
}
