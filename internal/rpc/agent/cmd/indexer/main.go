package main

import (
	"context"
	"fmt"
	"github.com/cloudwego/eino/components/document"
	"io/fs"
	"liveclass/internal/rpc/agent/eino_gen/indexer"
	"path/filepath"
	"strings"
)

func main() {
	ctx := context.Background()

	err := indexMarkdownFiles(ctx, "./data")
	if err != nil {
		panic(err)
	}

	fmt.Println("index success")
}

func indexMarkdownFiles(ctx context.Context, dir string) error {
	runner, err := indexer.BuildIndexer(ctx)
	if err != nil {
		return err
	}

	//range
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir failed: %w", err)
		}
		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			fmt.Printf("%s not a md\n", path)
			return nil
		}

		fmt.Printf("Start indexing markdown file:%s \n", path)

		ids, err := runner.Invoke(ctx, document.Source{URI: path})
		if err != nil {
			return fmt.Errorf("index failed: %w", err)
		}

		fmt.Printf("Finish indexing markdown file:%s,len of path:%d \n", path, len(ids))

		return nil
	})

	return err
}
