package agent

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	redisCli "github.com/redis/go-redis/v9"
	_const "liveclass/internal/rpc/agent/const"
	"liveclass/internal/rpc/agent/global"
	"strconv"

	"github.com/cloudwego/eino-ext/components/retriever/redis"
)

func newRetriever(ctx context.Context) (rtr retriever.Retriever, err error) {
	rClient := redisCli.NewClient(&redisCli.Options{
		Addr:     global.Config.RedisAddr,
		Protocol: 2,
	})

	//创建配置
	config := &redis.RetrieverConfig{
		Client: rClient,
		//索引全名
		Index: fmt.Sprintf("%s%s", _const.RedisPrefix, _const.IndexName),
		//命令方言
		Dialect: 2,
		//检索回哪些字段
		ReturnFields: []string{_const.ContentField, _const.MetadataField, _const.DistanceField},
		//每次返回TOPK条
		TopK: 8,
		//向量字段名
		VectorField: _const.VectorField,
		//文档转换器(document -> schema.Document)(多的distance会帮忙算出相似度得分)
		DocumentConverter: func(ctx context.Context, doc redisCli.Document) (*schema.Document, error) {
			resp := &schema.Document{
				ID:       doc.ID,
				Content:  "",
				MetaData: map[string]any{},
			}
			for field, v := range doc.Fields {
				if field == _const.ContentField {
					resp.Content = v
				}
				if field == _const.MetadataField {
					resp.MetaData[field] = v
				}
				if field == _const.DistanceField {
					distance, err := strconv.ParseFloat(v, 64)
					if err != nil {
						continue
					}
					resp.WithScore(1 - distance)
				}
			}

			return resp, nil
		},
	}
	embedding, err := newEmbedding(ctx)
	if err != nil {
		return nil, err
	}
	config.Embedding = embedding

	rtr, err = redis.NewRetriever(ctx, config)
	if err != nil {
		return nil, err
	}
	return rtr, nil
}
