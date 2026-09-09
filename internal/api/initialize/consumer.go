package initialize

import (
	"liveclass/internal/api/global"
	"time"

	"github.com/segmentio/kafka-go"
)

func InitChatKafkaReader(groupID string) *kafka.Reader {
	return kafka.NewReader(chatKafkaReaderConfig(groupID))
}

func chatKafkaReaderConfig(groupID string) kafka.ReaderConfig {
	return kafka.ReaderConfig{
		Brokers:     []string{global.Config.ChatKafka.Broker},
		Topic:       global.Config.ChatKafka.Topic,
		GroupID:     groupID,
		StartOffset: kafka.LastOffset,
		MinBytes:    1,
		MaxBytes:    10e6,
		MaxWait:     50 * time.Millisecond,
		// A positive interval makes CommitMessages enqueue offsets and lets the
		// reader commit the highest offset per partition in one broker request.
		// Processing remains broadcast-before-commit, so failures are replayable.
		CommitInterval: global.Config.ChatKafka.CommitInterval,
	}
}
