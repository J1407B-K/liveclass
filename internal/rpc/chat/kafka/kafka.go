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

func initKafkaWriter() *kafka.Writer {
	writer := &kafka.Writer{
		Addr:     kafka.TCP(global.Config.KafkaBroker), // 设置 Kafka broker 地址
		Topic:    global.Config.KafkaTopic,             // 设置 Kafka topic
		Balancer: &kafka.LeastBytes{},                  // 设置负载均衡策略
	}
	return writer
}

func ProduceMessage(userid, targetLesson string, timestamp time.Time, message []byte) error {
	// 创建 Kafka 生产者（writer），写完后关闭
	writer := initKafkaWriter()
	defer writer.Close() // 使用完关闭

	// 过滤敏感词 & 清理非法字符
	cleanedContent := filter.FilterSensitiveWords(filter.CleanMessage(string(message)))

	// 组织消息数据
	msg := model.Message{
		Sender:    userid,
		LessonID:  targetLesson,
		Content:   cleanedContent,
		Timestamp: timestamp,
	}

	// JSON 编码
	messageBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// 将消息写入 Kafka
	return writer.WriteMessages(context.Background(),
		kafka.Message{
			Value: messageBytes, // 消息内容
		},
	)
}
