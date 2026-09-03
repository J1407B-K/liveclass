package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"

	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/initialize"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/rag"
)

const (
	batchSize  = 32
	vectorSize = 2048
)

func main() {
	var (
		filePath   string
		lessonID   int64
		source     string
		parentSize int
		childSize  int
		overlap    int
		collection string
		index      string
	)
	flag.StringVar(&filePath, "path", "", "markdown file path")
	flag.Int64Var(&lessonID, "lesson", 0, "lesson id")
	flag.StringVar(&source, "source", "", "document source identifier")
	flag.IntVar(&parentSize, "parent-size", 0, "parent chunk characters (default from config)")
	flag.IntVar(&childSize, "child-size", 0, "child chunk characters (default from config)")
	flag.IntVar(&overlap, "overlap", -1, "child overlap characters (default from config)")
	flag.StringVar(&collection, "collection", "", "Qdrant collection override")
	flag.StringVar(&index, "index", "", "Elasticsearch index override")
	flag.Parse()

	if filePath == "" {
		fmt.Println("path is required")
		os.Exit(1)
	}
	if lessonID == 0 {
		fmt.Println("lesson id is required")
		os.Exit(1)
	}

	if source == "" {
		source = filepath.Base(filePath)
	}

	initialize.SetupViper()
	if collection != "" {
		global.Config.DocCollection = collection
	}
	if index != "" {
		global.Config.DocIndex = index
	}
	if err := dependency.Configure(global.Config.Resilience); err != nil {
		panic(err)
	}

	ctx := context.Background()

	multiEmbedder, err := initialize.InitMultiModalEmbedder(ctx)
	if err != nil {
		panic(err)
	}

	qdrantMgr, err := initialize.InitDocQdrant(ctx, vectorSize)
	if err != nil {
		panic(err)
	}
	esMgr, err := initialize.InitDocElasticsearch(ctx)
	if err != nil {
		fmt.Printf("elasticsearch init failed, skip bm25 index: %v\n", err)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		panic(err)
	}

	if parentSize <= 0 {
		parentSize = global.Config.RAG.ParentSize
	}
	if childSize <= 0 {
		childSize = global.Config.RAG.ChildSize
	}
	if overlap < 0 {
		overlap = global.Config.RAG.Overlap
	}
	parents, err := rag.ChunkMarkdown(string(content), rag.ChunkConfig{ParentSize: parentSize, ChildSize: childSize, Overlap: overlap})
	if err != nil {
		panic(err)
	}
	if len(parents) == 0 {
		fmt.Println("no content chunks generated")
		return
	}

	points := make([]*qdrant.PointStruct, 0, batchSize)
	esChunks := make([]rag.DocChunk, 0, batchSize)
	childCount := 0
	for _, parent := range parents {
		parentID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("%s:%d:%s:parent:%d", qdrantMgr.Collection, lessonID, source, parent.Index))).String()
		for _, child := range parent.Children {
			vector, err := multiEmbedder.EmbedText(ctx, child.Text)
			if err != nil {
				panic(err)
			}
			chunkID := uuid.NewSHA1(uuid.NameSpaceURL, []byte(fmt.Sprintf("%s:%d:%s:parent:%d:child:%d", qdrantMgr.Collection, lessonID, source, parent.Index, child.Index))).String()
			payload := map[string]any{"lesson_id": lessonID, "source": source, "parent_id": parentID, "parent_text": parent.Text, "heading": parent.Heading, "chunk_idx": parent.Index, "child_idx": child.Index, "text": child.Text}
			points = append(points, buildPoint(chunkID, memory.Float64To32(vector), payload))
			esChunks = append(esChunks, rag.DocChunk{ID: chunkID, ParentID: parentID, Text: child.Text, ParentText: parent.Text, Heading: parent.Heading, LessonID: lessonID, Source: source, ChunkIdx: int32(parent.Index), ChildIdx: int32(child.Index)})
			childCount++
			if len(points) >= batchSize {
				if err := upsertPoints(ctx, qdrantMgr, points); err != nil {
					panic(err)
				}
				if esMgr != nil {
					if err := esMgr.BulkUpsertDocChunks(ctx, esChunks); err != nil {
						panic(err)
					}
				}
				points = points[:0]
				esChunks = esChunks[:0]
			}
		}
	}
	if len(points) > 0 {
		if err := upsertPoints(ctx, qdrantMgr, points); err != nil {
			panic(err)
		}
		if esMgr != nil {
			if err := esMgr.BulkUpsertDocChunks(ctx, esChunks); err != nil {
				panic(err)
			}
		}
	}

	fmt.Printf("Indexed %d parent chunks and %d child chunks into collection %s\n", len(parents), childCount, qdrantMgr.Collection)
}

func buildPoint(id string, vector []float32, payload map[string]any) *qdrant.PointStruct {
	return &qdrant.PointStruct{
		Id:      qdrant.NewID(id),
		Vectors: qdrant.NewVectors(vector...),
		Payload: qdrant.NewValueMap(payload),
	}
}

func upsertPoints(ctx context.Context, mgr *memory.QdrantManager, points []*qdrant.PointStruct) error {
	_, err := dependency.Do(ctx, dependency.Qdrant, "bulk_upsert_docs", func(callCtx context.Context) (*qdrant.UpdateResult, error) {
		return mgr.Client.Upsert(callCtx, &qdrant.UpsertPoints{
			CollectionName: mgr.Collection,
			Points:         points,
		})
	})
	return err
}
