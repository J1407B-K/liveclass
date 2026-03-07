package initialize

import (
	"liveclass/internal/rpc/chat/global"

	"github.com/segmentio/kafka-go"
)

func InitKafkaWriter() *kafka.Writer {
	w := &kafka.Writer{
		Addr:     kafka.TCP(global.Config.KafkaBroker), // 设置 Kafka broker 地址
		Topic:    global.Config.KafkaTopic,             // 设置 Kafka topic
		Balancer: &kafka.LeastBytes{},                  // 设置负载均衡策略
	}
	return w
}
