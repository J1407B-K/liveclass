package kafka

import (
	"context"
	"encoding/json"
	"liveclass/internal/api/utils/filter"
	"liveclass/internal/rpc/chat/global"
	"liveclass/internal/rpc/chat/model"
	"time"

	"github.com/segmentio/kafka-go"
)

func CloseKafkaWriter() error {
	if global.Writer != nil {
		return global.Writer.Close()
	}
	return nil
}

func FilterMessage(message string) (string, time.Time) {
	cleanedContent := filter.FilterSensitiveWords(filter.CleanMessage(message))
	return cleanedContent, time.Now()
}

func ProduceFilteredMessage(userid, targetLesson int64, msg model.Message) error {
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return global.Writer.WriteMessages(ctx,
		kafka.Message{
			Value: messageBytes,
		},
	)
}
