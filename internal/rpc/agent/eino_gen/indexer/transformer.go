package indexer

import (
	"context"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
)

func newDocumentTransformer(ctx context.Context) (tfr document.Transformer, err error) {
	//创建配置
	config := &markdown.HeaderConfig{
		Headers: map[string]string{
			"#": "title",
		},
		//是否包含头部
		TrimHeaders: false,
	}

	tfr, err = markdown.NewHeaderSplitter(ctx, config)
	if err != nil {
		return nil, err
	}
	return tfr, nil

}
