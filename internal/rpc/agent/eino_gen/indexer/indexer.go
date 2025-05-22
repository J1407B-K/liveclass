package indexer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino-ext/components/indexer/redis"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	redisCli "github.com/redis/go-redis/v9"
	_const "liveclass/internal/rpc/agent/const"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/initialize"
)

func init() {
	err := initialize.InitRedisStackIndex()
	if err != nil {
		panic(err)
	}
}

func newIndexer(ctx context.Context) (idr indexer.Indexer, err error) {
	rClient := redisCli.NewClient(&redisCli.Options{
		Addr:     global.Config.RedisAddr,
		Protocol: 2,
	})

	//配置文件
	config := &redis.IndexerConfig{
		Client:           rClient,
		KeyPrefix:        _const.RedisPrefix,
		BatchSize:        1,
		DocumentToHashes: documentToHashesHandler,
	}

	embedding, err := newEmbedding(ctx)
	if err != nil {
		return nil, err
	}
	config.Embedding = embedding
	idr, err = redis.NewIndexer(ctx, config)
	if err != nil {
		return nil, err
	}

	return idr, nil
}

func documentToHashesHandler(ctx context.Context, doc *schema.Document) (*redis.Hashes, error) {
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	key := doc.ID

	metadataBytes, err := json.Marshal(doc.MetaData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return &redis.Hashes{
		Key: key,
		Field2Value: map[string]redis.FieldValue{
			_const.ContentField:  {Value: doc.Content, EmbedKey: _const.VectorField},
			_const.MetadataField: {Value: metadataBytes},
		},
	}, nil
}
