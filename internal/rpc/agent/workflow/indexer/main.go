package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"

	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/initialize"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/rag"
)

const (
	defaultChunkSize = 800
	batchSize        = 32
	vectorSize       = 2048
)

func main() {
	var (
		filePath string
		lessonID int64
		source   string
	)
	flag.StringVar(&filePath, "path", "", "markdown file path")
	flag.Int64Var(&lessonID, "lesson", 0, "lesson id")
	flag.StringVar(&source, "source", "", "document source identifier")
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

	chunks := splitMarkdown(string(content), defaultChunkSize)
	if len(chunks) == 0 {
		fmt.Println("no content chunks generated")
		return
	}

	points := make([]*qdrant.PointStruct, 0, batchSize)
	esChunks := make([]rag.DocChunk, 0, batchSize)
	for idx, chunk := range chunks {
		vector, err := multiEmbedder.EmbedText(ctx, chunk)
		if err != nil {
			panic(err)
		}
		chunkID := uuid.NewString()
		payload := map[string]any{
			"lesson_id": lessonID,
			"source":    source,
			"chunk_idx": idx,
			"text":      chunk,
		}

		points = append(points, buildPoint(chunkID, memory.Float64To32(vector), payload))
		esChunks = append(esChunks, rag.DocChunk{
			ID:       chunkID,
			Text:     chunk,
			LessonID: lessonID,
			Source:   source,
			ChunkIdx: int32(idx),
		})
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

	fmt.Printf("Indexed %d chunks into collection %s\n", len(chunks), qdrantMgr.Collection)
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

func splitMarkdown(content string, maxChunkLen int) []string {
	reader := bufio.NewReader(strings.NewReader(content))
	var (
		builder strings.Builder
		chunks  []string
	)

	writeChunk := func() {
		text := strings.TrimSpace(builder.String())
		if text != "" {
			chunks = append(chunks, text)
		}
		builder.Reset()
	}

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if builder.Len()+len(line) > maxChunkLen {
				writeChunk()
			}
			builder.WriteString(line)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
	}
	writeChunk()

	merged := make([]string, 0, len(chunks))
	var tmp strings.Builder
	for _, chunk := range chunks {
		if tmp.Len()+len(chunk) <= maxChunkLen {
			if tmp.Len() > 0 {
				tmp.WriteString("\n")
			}
			tmp.WriteString(chunk)
			continue
		}
		merged = append(merged, tmp.String())
		tmp.Reset()
		tmp.WriteString(chunk)
	}
	if tmp.Len() > 0 {
		merged = append(merged, tmp.String())
	}
	return merged
}
