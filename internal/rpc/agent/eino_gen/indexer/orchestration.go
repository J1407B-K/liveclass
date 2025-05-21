package indexer

import (
	"context"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/compose"
	_const "liveclass/internal/rpc/agent/const"
)

func BuildIndexer(ctx context.Context) (r compose.Runnable[document.Source, []string], err error) {
	g := compose.NewGraph[document.Source, []string]()

	//fileLoader节点
	fileLoaderKeyOfLoader, err := newLoader(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddLoaderNode(_const.FileLoader, fileLoaderKeyOfLoader, compose.WithNodeName("FileLoader"))

	//markdownSplitter节点
	markdownSplitterKeyOfDocumentTransformer, err := newDocumentTransformer(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddDocumentTransformerNode(_const.MarkdownSplitter, markdownSplitterKeyOfDocumentTransformer, compose.WithNodeName("DocumentTransformer"))

	//redisIndexer节点
	redisIndexerKeyOfIndexer, err := newIndexer(ctx)
	if err != nil {
		return nil, err
	}
	_ = g.AddIndexerNode(_const.RedisIndexer, redisIndexerKeyOfIndexer, compose.WithNodeName("RedisIndexer"))

	//START -> fileLoader
	_ = g.AddEdge(compose.START, _const.FileLoader)

	//fileLoader -> markdownSplitter
	_ = g.AddEdge(_const.FileLoader, _const.MarkdownSplitter)

	//markdownSplitter -> redisIndexer
	_ = g.AddEdge(_const.MarkdownSplitter, _const.RedisIndexer)

	//redisIndexer -> END
	_ = g.AddEdge(_const.RedisIndexer, compose.END)

	r, err = g.Compile(ctx, compose.WithGraphName("indexer"), compose.WithNodeTriggerMode(compose.AnyPredecessor))
	if err != nil {
		return nil, err
	}

	return r, nil
}
