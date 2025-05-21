package indexer

import (
	"context"
	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino/components/document"
)

func newLoader(ctx context.Context) (ldr document.Loader, err error) {
	//创建配置
	config := &file.FileLoaderConfig{}
	ldr, err = file.NewFileLoader(ctx, config)
	if err != nil {
		return nil, err
	}
	return ldr, nil
}
