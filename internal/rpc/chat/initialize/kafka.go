package initialize

import (
	"liveclass/internal/rpc/chat/global"

	"github.com/segmentio/kafka-go"
)

func InitKafkaWriter() *kafka.Writer {
	return &kafka.Writer{
		Addr:     kafka.TCP(global.Config.KafkaBroker),
		Topic:    global.Config.KafkaTopic,
		Balancer: &kafka.LeastBytes{},
	}
}

func InitKafkaReader(groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{global.Config.KafkaBroker},
		Topic:    global.Config.KafkaTopic,
		GroupID:  groupID,
		MinBytes: 1e3,
		MaxBytes: 10e6,
	})
}
