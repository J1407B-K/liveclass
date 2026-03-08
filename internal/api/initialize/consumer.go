package initialize

import (
	"liveclass/internal/api/global"

	"github.com/segmentio/kafka-go"
)

func InitChatKafkaReader(groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{global.KafkaBroker},
		Topic:    global.KafkaTopic,
		GroupID:  groupID,
		MinBytes: 1e3,
		MaxBytes: 10e6,
	})
}
