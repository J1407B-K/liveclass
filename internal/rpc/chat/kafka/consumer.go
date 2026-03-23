package kafka

import (
	"context"
	"encoding/json"
	"liveclass/internal/rpc/chat/dao"
	"liveclass/internal/rpc/chat/model"
	"log"

	"github.com/segmentio/kafka-go"
	"go.mongodb.org/mongo-driver/mongo"
)

func RunMongoWriter(ctx context.Context, reader *kafka.Reader, client *mongo.Client) error {
	defer reader.Close()

	for {
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		var msg model.Message
		if err := json.Unmarshal(m.Value, &msg); err != nil {
			log.Printf("chat mongo writer unmarshal failed offset=%d err=%v", m.Offset, err)
			_ = reader.CommitMessages(ctx, m)
			continue
		}

		coll := dao.ChooseCollection(msg.LessonID, client)
		if err := dao.InsertMongo(ctx, coll, msg); err != nil {
			log.Printf("chat mongo writer insert failed lesson=%d err=%v", msg.LessonID, err)
		}

		if err := reader.CommitMessages(ctx, m); err != nil {
			log.Printf("chat mongo writer commit failed offset=%d err=%v", m.Offset, err)
		}
	}
}
