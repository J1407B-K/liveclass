package initialize

import (
	"context"
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	"log"

	"github.com/qdrant/go-client/qdrant"
)

type fieldIndex struct {
	Name string
	Type qdrant.FieldType
}

func InitQdrant(ctx context.Context, vectorSize uint64) (*memory.QdrantManager, error) {
	indexes := []fieldIndex{
		{Name: "user_id", Type: qdrant.FieldType_FieldTypeInteger},
		{Name: "fact_type", Type: qdrant.FieldType_FieldTypeKeyword},
		{Name: "lesson_id", Type: qdrant.FieldType_FieldTypeInteger},
		{Name: "is_active", Type: qdrant.FieldType_FieldTypeBool},
	}
	return initQdrantCollection(ctx, vectorSize, global.Config.QdrantConfig.Collection, indexes)
}

func InitDocQdrant(ctx context.Context, vectorSize uint64) (*memory.QdrantManager, error) {
	collection := global.Config.QdrantConfig.DocCollection
	if collection == "" {
		collection = global.Config.QdrantConfig.Collection + "_docs"
	}
	indexes := []fieldIndex{
		{Name: "lesson_id", Type: qdrant.FieldType_FieldTypeInteger},
		{Name: "source", Type: qdrant.FieldType_FieldTypeKeyword},
	}
	return initQdrantCollection(ctx, vectorSize, collection, indexes)
}

func initQdrantCollection(ctx context.Context, vectorSize uint64, collection string, indexes []fieldIndex) (*memory.QdrantManager, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   global.Config.QdrantConfig.Host,
		Port:   global.Config.QdrantConfig.GrpcPort,
		APIKey: global.Config.QdrantConfig.ApiKey,
		UseTLS: false,
	})
	if err != nil {
		return nil, fmt.Errorf("init qdrant client: %w", err)
	}

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

		for _, idx := range indexes {
			ft := idx.Type
			_, err = client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
				CollectionName: collection,
				FieldName:      idx.Name,
				FieldType:      &ft,
			})
			if err != nil {
				log.Printf("create field index %s failed: %v", idx.Name, err)
			}
		}
	}

	return mgr, nil
}
