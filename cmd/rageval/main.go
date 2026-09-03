package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"liveclass/internal/rpc/agent/dependency"
	agenteval "liveclass/internal/rpc/agent/eval"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/initialize"
	"liveclass/internal/rpc/agent/rag"
	"liveclass/internal/rpc/agent/rerank"
)

func main() {
	casesPath := flag.String("cases", "eval/cases.jsonl", "gold cases JSONL")
	output := flag.String("output", "", "prediction JSONL")
	variant := flag.String("variant", "parent_child_rerank", "baseline|baseline_overlap|parent_child|parent_child_rerank")
	rerankStage := flag.String("rerank-stage", "parent", "for parent_child_rerank: child|parent|two_stage")
	collection := flag.String("collection", "", "Qdrant collection override")
	index := flag.String("index", "", "Elasticsearch index override")
	flag.Parse()
	if *output == "" {
		fatal("output is required")
	}
	if *variant == "parent_child_rerank" && *rerankStage != "child" && *rerankStage != "parent" && *rerankStage != "two_stage" {
		fatal("rerank-stage must be child, parent, or two_stage")
	}
	initialize.SetupViper()
	if *collection != "" {
		global.Config.DocCollection = *collection
	}
	if *index != "" {
		global.Config.DocIndex = *index
	}
	if err := dependency.Configure(global.Config.Resilience); err != nil {
		fatal(err.Error())
	}
	ctx := context.Background()
	embedder, err := initialize.InitMultiModalEmbedder(ctx)
	if err != nil {
		fatal(err.Error())
	}
	global.MultiModalEmbedder = embedder
	docMgr, err := initialize.InitDocQdrant(ctx, 2048)
	if err != nil {
		fatal(err.Error())
	}
	esMgr, _ := initialize.InitDocElasticsearch(ctx)
	retriever, err := rag.NewDocRetriever(docMgr, esMgr)
	if err != nil {
		fatal(err.Error())
	}
	caseFile, err := os.Open(*casesPath)
	if err != nil {
		fatal(err.Error())
	}
	cases, err := agenteval.ReadJSONL[agenteval.Case](caseFile)
	caseFile.Close()
	if err != nil {
		fatal(err.Error())
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fatal(err.Error())
	}
	out, err := os.Create(*output)
	if err != nil {
		fatal(err.Error())
	}
	defer out.Close()
	writer := bufio.NewWriter(out)
	defer writer.Flush()
	for _, c := range cases {
		if len(c.GoldDocs) == 0 {
			continue
		}
		lessonID := metadataInt64(c.Metadata, "lesson_id")
		if lessonID <= 0 {
			continue
		}
		started := time.Now()
		vector, err := embedder.EmbedText(ctx, c.Question)
		if err != nil {
			fatal(err.Error())
		}
		children, err := retriever.SearchHybridChildren(ctx, lessonID, c.Question, vector, global.Config.RAG.ChildTopK)
		if err != nil {
			fatal(err.Error())
		}
		docs := children
		if *variant == "parent_child_rerank" && (*rerankStage == "child" || *rerankStage == "two_stage") {
			childLimit := global.Config.RAG.ParentTopK
			if *rerankStage == "two_stage" {
				childLimit *= 2
			}
			if childLimit <= 0 {
				childLimit = 6
			}
			children, err = rerank.Docs(ctx, c.Question, children, childLimit)
			if err != nil {
				fatal("child reranker unavailable: " + err.Error())
			}
			docs = children
		}
		if *variant == "parent_child" || *variant == "parent_child_rerank" {
			docs = rag.ExpandAndDeduplicateParents(children)
		}
		if *variant == "parent_child_rerank" && (*rerankStage == "parent" || *rerankStage == "two_stage") {
			ranked, rankErr := rerank.Docs(ctx, c.Question, docs, global.Config.RAG.ParentTopK)
			if rankErr != nil {
				fatal("reranker unavailable: " + rankErr.Error())
			}
			docs = ranked
		}
		prediction := agenteval.Prediction{CaseID: c.ID, LatencyMs: float64(time.Since(started).Microseconds()) / 1000}
		for _, doc := range docs {
			prediction.RetrievedDocs = append(prediction.RetrievedDocs, rag.CitationID(doc))
		}
		raw, _ := json.Marshal(prediction)
		fmt.Fprintln(writer, string(raw))
	}
}
func metadataInt64(metadata map[string]any, key string) int64 {
	value, _ := metadata[key].(float64)
	return int64(value)
}
func fatal(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
