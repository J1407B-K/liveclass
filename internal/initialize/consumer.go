package initialize

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/segmentio/kafka-go"
	"liveclass/internal/global"
	"liveclass/internal/model"
	"log"
)

func ConsumeKafkaMessages() {
	reader := initKafkaReader()
	defer reader.Close()

	for {
		// 使用 context.Background()
		msg, err := reader.ReadMessage(context.Background())
		if err != nil {
			continue // 继续读取下一条消息
		}

		// 解析 JSON 消息
		var message model.Message
		err = json.Unmarshal(msg.Value, &message)
		if err != nil {
			log.Println("解析 Kafka 消息失败:", err)
			continue
		}

		log.Printf("收到 Kafka 消息: %s\n", string(msg.Value))

		global.Mux.Lock()
		defer global.Mux.Unlock()
		for conn, lid := range global.WsConnsChat {
			if lid == message.LessonID {
				err = conn.WriteMessage(websocket.TextMessage, msg.Value)
				if err != nil {
					log.Fatal("广播错误")
				}
			}
		}
	}
}

func initKafkaReader() *kafka.Reader {
	// 创建 Kafka 消费者（reader）
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{global.KafkaBroker},
		Topic:   global.KafkaTopic,
		GroupID: "chat-group",
	})
	return reader
}
