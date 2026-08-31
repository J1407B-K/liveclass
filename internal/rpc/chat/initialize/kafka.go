package initialize

import (
	"liveclass/internal/rpc/chat/global"
	"time"

	"github.com/segmentio/kafka-go"
)

func InitKafkaWriter() *kafka.Writer {
	return &kafka.Writer{
		Addr:         kafka.TCP(global.Config.KafkaBroker),
		Topic:        global.Config.KafkaTopic,
		Balancer:     &kafka.Hash{},
		BatchSize:    100,
		BatchTimeout: 10 * time.Millisecond,
	}
}
