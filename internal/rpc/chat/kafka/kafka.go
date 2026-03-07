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

func ProduceMessage(userid, targetLesson int64, timestamp time.Time, message []byte) error {
	cleanedContent := filter.FilterSensitiveWords(filter.CleanMessage(string(message)))

	msg := model.Message{
		Sender:    userid,
		LessonID:  targetLesson,
		Content:   cleanedContent,
		Timestamp: timestamp,
	}

	messageBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return global.Writer.WriteMessages(context.Background(),
		kafka.Message{
			Value: messageBytes,
		},
	)
}
