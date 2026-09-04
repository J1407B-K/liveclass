package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/google/uuid"
	"gorm.io/gorm"

	agentapi "liveclass/idl/kitex_gen/agent"
	"liveclass/idl/kitex_gen/agent/agentservice"
	agenteval "liveclass/internal/rpc/agent/eval"
	"liveclass/internal/rpc/agent/initialize"
	"liveclass/internal/rpc/agent/model"
)

func main() {
	casesPath := flag.String("cases", "eval/cases.jsonl", "gold cases JSONL")
	output := flag.String("output", "", "prediction JSONL")
	variant := flag.String("variant", "v2", "variant label used in request IDs")
	runID := flag.String("run-id", "", "optional suffix that isolates replay conversations")
	caseID := flag.String("case", "", "run only one case ID")
	host := flag.String("host", "127.0.0.1:9006", "agent RPC address")
	userID := flag.Int64("user", 0, "replay user ID")
	maxConsecutiveErrors := flag.Int("max-consecutive-rpc-errors", 3, "abort after this many consecutive RPC errors (0 disables)")
	flag.Parse()
	if *output == "" || *userID <= 0 {
		fatal("output and a positive user are required")
	}
	initialize.SetupViper()
	db := initialize.InitPGDB()
	cli, err := agentservice.NewClient("agentservice", client.WithHostPorts(*host), client.WithRPCTimeout(90*time.Second), client.WithMetaHandler(transmeta.ClientTTHeaderHandler))
	if err != nil {
		fatal(err.Error())
	}
	file, err := os.Open(*casesPath)
	if err != nil {
		fatal(err.Error())
	}
	cases, err := agenteval.ReadJSONL[agenteval.Case](file)
	file.Close()
	if err != nil {
		fatal(err.Error())
	}
	if err := agenteval.ValidateCases(cases); err != nil {
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
	w := bufio.NewWriter(out)
	defer w.Flush()
	consecutiveErrors := 0
	for _, c := range cases {
		if *caseID != "" && c.ID != *caseID {
			continue
		}
		namespace := *variant
		if value := strings.TrimSpace(*runID); value != "" {
			namespace += "-" + value
		}
		// Trace/request columns are varchar(64). Case identity already lives in
		// Prediction.CaseID, so keep the transport id compact and collision-safe.
		requestID := newEvalRequestID(*variant)
		started := time.Now()
		callCtx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		callCtx = metainfo.WithPersistentValue(callCtx, "agent-eval-variant", *variant)
		resp, callErr := cli.ChatWithAgent(callCtx, &agentapi.ChatWithAgentReq{
			Userid: *userID, Message: c.Question, RequestId: requestID,
			ConvId: "eval-" + namespace + "-" + c.ID, LessonId: metadataInt64(c.Metadata, "lesson_id"),
		})
		cancel()
		p := agenteval.Prediction{CaseID: c.ID, LatencyMs: float64(time.Since(started).Microseconds()) / 1000}
		if callErr != nil {
			p.ToolErrors = 1
			p.Answer = "[RPC error] " + callErr.Error()
		} else if resp != nil && resp.Resp != nil {
			p.Answer = resp.Resp.Msg
		}
		collectTrace(db, requestID, &p)
		p.CitedDocs = extractCitations(p.Answer)
		raw, _ := json.Marshal(p)
		fmt.Fprintln(w, string(raw))
		if err := w.Flush(); err != nil {
			fatal(err.Error())
		}
		if callErr != nil {
			consecutiveErrors++
		} else {
			consecutiveErrors = 0
		}
		if *maxConsecutiveErrors > 0 && consecutiveErrors >= *maxConsecutiveErrors {
			fatal(fmt.Sprintf("aborting after %d consecutive RPC errors; predictions are incomplete", consecutiveErrors))
		}
	}
}

func newEvalRequestID(variant string) string {
	variant = strings.TrimSpace(variant)
	if len(variant) > 16 {
		variant = variant[:16]
	}
	return fmt.Sprintf("eval-%s-%s", variant, uuid.NewString())
}

var citationPattern = regexp.MustCompile(`\[([^\[\]\n]+\.md#[^\[\]\n]+)\]`)

func extractCitations(answer string) []string {
	matches := citationPattern.FindAllStringSubmatch(answer, -1)
	seen := make(map[string]struct{}, len(matches))
	result := make([]string, 0, len(matches))
	for _, match := range matches {
		citation := strings.TrimSpace(match[1])
		if _, ok := seen[citation]; ok {
			continue
		}
		seen[citation] = struct{}{}
		result = append(result, citation)
	}
	return result
}

func collectTrace(db *gorm.DB, requestID string, p *agenteval.Prediction) {
	var events []model.AgentTraceEvent
	if err := db.Raw("SELECT * FROM agent_trace_events WHERE request_id = ? ORDER BY id", requestID).Scan(&events).Error; err != nil {
		p.ToolErrors++
		return
	}
	for _, event := range events {
		var metadata map[string]any
		_ = json.Unmarshal([]byte(event.Metadata), &metadata)
		switch event.EventType {
		case "skill_route":
			p.Skill = strings.Split(event.Name, ",")[0]
		case "tool_result":
			p.Tools = append(p.Tools, event.Name)
			if event.Status == "error" || event.Status == "denied" {
				p.ToolErrors++
			}
		case "rag_retrieval":
			if docs, ok := metadata["documents"].([]any); ok {
				for _, doc := range docs {
					if value, ok := doc.(string); ok {
						p.RetrievedDocs = append(p.RetrievedDocs, value)
					}
				}
			}
		case "token_usage":
			p.LLMCalls++
			if total, ok := metadata["total"].(float64); ok {
				p.Tokens += int(total)
			}
		case "run_finished":
			if steps, ok := metadata["steps"].(float64); ok {
				p.Steps = int(steps)
			}
		case "retry":
			p.Retries++
		case "fallback":
			p.Fallbacks++
		}
	}
}

func metadataInt64(metadata map[string]any, key string) int64 {
	value, _ := metadata[key].(float64)
	return int64(value)
}

func fatal(message string) { fmt.Fprintln(os.Stderr, message); os.Exit(1) }
