package initialize

import (
	"context"
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	"log"

	"github.com/qdrant/go-client/qdrant"
)

func InitQdrant(ctx context.Context, vectorSize uint64) (*memory.QdrantManager, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   global.Config.QdrantConfig.Host,
		Port:   global.Config.QdrantConfig.GrpcPort,
		APIKey: global.Config.QdrantConfig.ApiKey,
		UseTLS: false,
	})
	if err != nil {
		return nil, fmt.Errorf("init qdrant client: %w", err)
	}

	collection := global.Config.QdrantConfig.Collection

	mgr := &memory.QdrantManager{
		Client:     client,
		Collection: collection,
	}

	exists, err := client.CollectionExists(ctx, collection)
	if err != nil {
		return nil, fmt.Errorf("check qdrant collection exists: %w", err)
	}

	if !exists {
		err = client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: collection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     vectorSize,
				Distance: qdrant.Distance_Cosine,
			}),
		})
		if err != nil {
			return nil, fmt.Errorf("create qdrant collection: %w", err)
		}
		log.Printf("qdrant collection %s created\n", collection)

		// 常用过滤字段建 payload index
		fieldTypeInt := qdrant.FieldType_FieldTypeInteger

		_, err = client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: collection,
			FieldName:      "user_id",
			FieldType:      &fieldTypeInt,
		})
		fieldTypeKeyword := qdrant.FieldType_FieldTypeKeyword

		_, err = client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: collection,
			FieldName:      "fact_type",
			FieldType:      &fieldTypeKeyword,
		})

		fieldTypeBool := qdrant.FieldType_FieldTypeBool

		_, err = client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: collection,
			FieldName:      "is_active",
			FieldType:      &fieldTypeBool,
		})
	}

	return mgr, nil
}
