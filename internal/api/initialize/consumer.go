package initialize

import (
	"liveclass/internal/api/global"
	"time"

	"github.com/segmentio/kafka-go"
)

func InitChatKafkaReader(groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{global.Config.ChatKafka.Broker},
		Topic:       global.Config.ChatKafka.Topic,
		GroupID:     groupID,
		StartOffset: kafka.LastOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     50 * time.Millisecond,
	})
}
