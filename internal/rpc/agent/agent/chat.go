package agent

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"liveclass/internal/rpc/agent/agentmetrics"
	"liveclass/internal/rpc/agent/dependency"
	agenteval "liveclass/internal/rpc/agent/eval"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/model"
	userprofile "liveclass/internal/rpc/agent/profile"
	"liveclass/internal/rpc/agent/rag"
	"liveclass/internal/rpc/agent/rerank"
	agentsession "liveclass/internal/rpc/agent/session"
	"liveclass/internal/rpc/agent/toolruntime"
	agenttrace "liveclass/internal/rpc/agent/trace"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/gopkg/cloud/metainfo"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"golang.org/x/sync/errgroup"
)

const factExtractTimeout = 30 * time.Second

type AgentRunner interface {
	Invoke(context.Context, *model.UserMessage, ...compose.Option) (*schema.Message, error)
}

type FactRunner interface {
	Invoke(context.Context, *model.FactExtractInput, ...compose.Option) ([]*model.FactCandidate, error)
}

func ChatWithAgent(
	ctx context.Context,
	dbm *memory.DBManager,
	agentRunner AgentRunner,
	factRunner FactRunner,
	docRetriever *rag.DocRetriever,
	embedder global.TextMultiModalEmbedder,
	sessionManager *agentsession.Manager,
	userID int64,
	convID string,
	requestID string,
	msg string,
	lessonID int64,
) (answer string, retErr error) {
	originalMsg := msg
	started := time.Now()
	succeeded := false
	observedSkill := ""
	observedTokens := 0
	run := agenttrace.NewRun(dbm, uuid.NewString(), convID, requestID, userID)
	ctx = agenttrace.WithRun(ctx, run)
	role, _ := metainfo.GetValue(ctx, "user-role")
	if role != "student" && role != "teacher" && role != "admin" {
		role = "student"
	}
	ctx = toolruntime.WithPrincipal(ctx, toolruntime.Principal{UserID: userID, Role: role, LessonID: lessonID, SessionID: convID, RequestID: requestID, AllowPlanning: isComplexPlanningRequest(msg)})
	reminderSteps := 5
	if global.Config != nil && global.Config.AgentRuntime.PlanReminderSteps > 0 {
		reminderSteps = global.Config.AgentRuntime.PlanReminderSteps
	}
	ctx = toolruntime.WithProgressTracker(ctx, toolruntime.NewProgressTracker(reminderSteps))
	run.Record(ctx, "run_started", "agent", "started", 0, map[string]any{"lesson_id": lessonID})
	run.RecordTranscript(ctx, "runtime_event", "run_started", "Agent request started", map[string]any{"lesson_id": lessonID})
	defer func() {
		status := "error"
		if succeeded {
			status = "success"
			agentmetrics.Success.Inc()
		}
		agentmetrics.Requests.WithLabelValues(status).Inc()
		agentmetrics.Latency.Observe(time.Since(started).Seconds())
		agentmetrics.Steps.Observe(float64(run.Steps()))
		run.Record(ctx, "run_finished", "agent", status, time.Since(started), map[string]any{"steps": run.Steps()})
		if retErr != nil {
			safeError := agenttrace.SafeError(retErr)
			run.Record(ctx, "error", "agent", "error", time.Since(started), map[string]any{"error": safeError})
			run.RecordTranscript(ctx, "error", "agent", safeError, map[string]any{"stage": "request"})
		} else {
			run.Record(ctx, "final_result", "agent", status, time.Since(started), map[string]any{"response_chars": len(answer)})
		}
		run.RecordTranscript(ctx, "runtime_event", "run_finished", "Agent request finished", map[string]any{"status": status, "duration_ms": time.Since(started).Milliseconds()})
		duplicates, retries, fallbacks, toolErrors := run.Stats()
		selected, reasons := agenteval.ShouldJudge(agenteval.OnlinePolicy{
			MaxTokens: 12000, MaxSteps: 20, MaxRetries: 2, MaxLatency: 30 * time.Second,
			RandomSampleRate: 0.01,
			AllowedSkills:    map[string]bool{"general": true, "lesson_plan": true, "lesson_summary": true, "quiz_help": true, "student_qa": true},
		}, agenteval.Observation{RequestID: requestID, Skill: observedSkill, Tokens: observedTokens, Steps: run.Steps(), DuplicateTools: duplicates, Retries: retries, Fallbacks: fallbacks, ToolErrors: toolErrors, Latency: time.Since(started)})
		if selected {
			run.Record(ctx, "online_eval_sample", "judge_queue", "selected", 0, map[string]any{"reasons": reasons})
		}
	}()
	if agentRunner == nil {
		return "", errors.New("nil agent runner")
	}

	// 幂等
	existedResp, err := dbm.GetAssistantMessageByRequestID(ctx, convID, requestID)
	if err != nil {
		return "", err
	}
	if existedResp != nil {
		succeeded = true
		return existedResp.Content, nil
	}
	if sessionManager != nil {
		transcriptResp, transcriptErr := dbm.GetTranscriptEvent(ctx, convID, requestID, "assistant")
		if transcriptErr != nil {
			return "", transcriptErr
		}
		if transcriptResp != nil {
			if err = dbm.AppendMessage(ctx, userID, convID, requestID, schema.AssistantMessage(transcriptResp.Content, nil)); err != nil {
				return "", err
			}
			succeeded = true
			return transcriptResp.Content, nil
		}
	}

	var history []*schema.Message
	if sessionManager == nil {
		history, err = dbm.GetRecentMessages(ctx, convID, 6)
		if err != nil {
			return "", err
		}
	}

	// Persist the user input before any model call. Transcript remains the fact
	// source even when generation fails midway; the legacy messages table is
	// retained for API compatibility.
	if err = dbm.AppendMessage(ctx, userID, convID, requestID, schema.UserMessage(msg)); err != nil {
		return "", err
	}
	if sessionManager != nil {
		if err = sessionManager.AppendMessage(ctx, userID, convID, requestID, "user", "user_message", string(schema.User), msg); err != nil {
			return "", err
		}
	}
	// 并行执行三个独立的 IO 操作：facts 检索、doc 检索、profile 生成
	var (
		factText       string
		docText        string
		profileSummary string
		mu             sync.Mutex
	)

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() error {
		memoryStarted := time.Now()
		defer func() {
			agentmetrics.MemoryRetrievalLatency.WithLabelValues("semantic_fact").Observe(time.Since(memoryStarted).Seconds())
		}()
		var facts []*model.UserFact
		var ferr error
		if sessionManager == nil {
			facts, ferr = dbm.RetrieveRelevantFacts(egCtx, userID, msg, 5)
		} else {
			facts, ferr = dbm.RetrieveSemanticFacts(egCtx, userID, msg, 5)
		}
		if ferr != nil {
			dependency.FallbackContext(egCtx, dependency.LongTermMemory, "retrieve_facts")
			facts = nil
		}
		if len(facts) > 0 {
			if ranked, rerr := rerank.Facts(egCtx, msg, facts, 5); rerr == nil && len(ranked) > 0 {
				facts = ranked
			}
		}
		if sessionManager != nil {
			episodeStarted := time.Now()
			episodes, episodeErr := dbm.RetrieveEpisodicMemory(egCtx, userID, msg, time.Now().AddDate(0, -6, 0), time.Now(), 3)
			agentmetrics.MemoryRetrievalLatency.WithLabelValues("episodic").Observe(time.Since(episodeStarted).Seconds())
			if episodeErr == nil {
				facts = append(facts, episodes...)
			}
		}
		mu.Lock()
		factText = memory.FormatFactsForPrompt(facts)
		mu.Unlock()
		return nil
	})

	eg.Go(func() error {
		if lessonID <= 0 || docRetriever == nil || embedder == nil {
			return nil
		}
		vector, embErr := embedder.EmbedText(egCtx, msg)
		if embErr != nil {
			log.Printf("doc embed error: %v", embErr)
			dependency.FallbackContext(egCtx, dependency.Embedding, "embed_docs")
			return nil
		}
		childTopK, parentTopK := 8, 3
		if global.Config != nil {
			if global.Config.RAG.ChildTopK > 0 {
				childTopK = global.Config.RAG.ChildTopK
			}
			if global.Config.RAG.ParentTopK > 0 {
				parentTopK = global.Config.RAG.ParentTopK
			}
		}
		var chunks []rag.DocChunk
		var retrErr error
		if sessionManager == nil {
			chunks, retrErr = docRetriever.SearchHybridChildren(egCtx, lessonID, msg, vector, 6)
		} else {
			chunks, retrErr = docRetriever.SearchHybrid(egCtx, lessonID, msg, vector, childTopK)
		}
		if retrErr != nil {
			log.Printf("doc search error: %v", retrErr)
			dependency.FallbackContext(egCtx, dependency.Qdrant, "search_docs")
			return nil
		}
		if reranked, rerr := rerank.Docs(egCtx, msg, chunks, parentTopK); rerr == nil && len(reranked) > 0 {
			chunks = reranked
		}
		retrieved := make([]string, 0, len(chunks))
		for _, chunk := range chunks {
			retrieved = append(retrieved, rag.CitationID(chunk))
		}
		run.Record(egCtx, "rag_retrieval", "lesson_docs", "success", 0, map[string]any{"documents": retrieved, "lesson_id": lessonID})
		var builder strings.Builder
		for _, chunk := range chunks {
			builder.WriteString("- 来源: ")
			if chunk.Source != "" {
				builder.WriteString(chunk.Source)
			} else {
				builder.WriteString("unknown")
			}
			if strings.TrimSpace(chunk.Heading) != "" {
				builder.WriteString(" ")
				builder.WriteString(chunk.Heading)
			} else {
				builder.WriteString(" #段落")
				builder.WriteString(strconv.FormatInt(int64(chunk.ChunkIdx), 10))
			}
			builder.WriteString("\n")
			builder.WriteString(chunk.Text)
			builder.WriteString("\n")
		}
		mu.Lock()
		docText = strings.TrimSpace(builder.String())
		mu.Unlock()
		return nil
	})

	eg.Go(func() error {
		if isEvalReplay(egCtx) {
			// Canonical cases use a fresh user and evaluate the request itself.
			// Generating/caching a profile here would make later variants inherit
			// state from the first run and consume an extra model call per case.
			return nil
		}
		summary, perr := userprofile.EnsureUserProfile(egCtx, dbm, userID)
		if perr != nil {
			log.Printf("generate user profile failed: %v", perr)
			dependency.FallbackContext(egCtx, dependency.Profile, "load")
			return nil
		}
		mu.Lock()
		profileSummary = summary
		mu.Unlock()
		return nil
	})

	// 等待三个并行操作完成（均为降级处理，不会返回错误）
	_ = eg.Wait()
	if sessionManager != nil {
		workingContext, contextErr := sessionManager.Recover(ctx, convID, agentsession.BuildInput{
			Profile: profileSummary, Facts: factText, Docs: docText, CurrentRequestID: requestID, CurrentRequest: msg,
		})
		if contextErr != nil {
			return "", contextErr
		}
		history = workingContext.History
		profileSummary = workingContext.Profile
		factText = workingContext.Facts
		docText = workingContext.Docs
		msg = workingContext.CurrentRequest
		agentmetrics.ContextTokens.WithLabelValues("total_estimated").Observe(float64(workingContext.EstimatedTokens))
		agentmetrics.ContextTokens.WithLabelValues("profile").Observe(float64(agentsession.EstimateTextTokens(workingContext.Profile)))
		agentmetrics.ContextTokens.WithLabelValues("memory").Observe(float64(agentsession.EstimateTextTokens(workingContext.Facts)))
		agentmetrics.ContextTokens.WithLabelValues("rag").Observe(float64(agentsession.EstimateTextTokens(workingContext.Docs)))
		agentmetrics.ContextTokens.WithLabelValues("current_request").Observe(float64(agentsession.EstimateTextTokens(workingContext.CurrentRequest)))
		historyTokens := 0
		for _, message := range workingContext.History {
			historyTokens += agentsession.EstimateTextTokens(message.Content)
		}
		agentmetrics.ContextTokens.WithLabelValues("history").Observe(float64(historyTokens))
	}

	userMsg := &model.UserMessage{
		ID:      userID,
		Lesson:  lessonID,
		Query:   msg,
		History: history,
		Facts:   factText,
		Profile: profileSummary,
		Docs:    docText,
	}

	// The adaptive runner applies dependency policy per model call. Wrapping the
	// whole Plan-and-Execute loop in one LLM timeout would cancel valid multi-step
	// tasks after the timeout intended for a single generation.
	resp, err := agentRunner.Invoke(ctx, userMsg)
	if err != nil {
		return "", err
	}
	if userMsg.SkillAdvice != nil {
		observedSkill = strings.Join(userMsg.SkillAdvice.Skills, ",")
		run.Record(ctx, "skill_route", strings.Join(userMsg.SkillAdvice.Skills, ","), "selected", 0, nil)
	}
	if resp.ResponseMeta != nil && resp.ResponseMeta.Usage != nil {
		usage := resp.ResponseMeta.Usage
		observedTokens = usage.TotalTokens
		agentmetrics.Tokens.WithLabelValues("prompt").Observe(float64(usage.PromptTokens))
		agentmetrics.Tokens.WithLabelValues("completion").Observe(float64(usage.CompletionTokens))
		run.Record(ctx, "token_usage", "main_llm", "observed", 0, map[string]any{"prompt": usage.PromptTokens, "completion": usage.CompletionTokens, "total": usage.TotalTokens})
	}

	if sessionManager != nil {
		if err = sessionManager.AppendMessage(ctx, userID, convID, requestID, "assistant", "assistant_message", string(schema.Assistant), resp.Content); err != nil {
			return "", err
		}
	}
	if err = dbm.AppendMessage(ctx, userID, convID, requestID, resp); err != nil {
		return "", err
	}

	// Canonical replay cases must be independent. Writing facts from one case
	// back into shared user memory would contaminate every later case and make
	// v1/v2 results depend on execution order.
	if shouldExtractFacts(ctx) {
		// 异步 fact 提取：使用独立超时，避免 goroutine 泄漏
		go func(userID int64, convID, msg string) {
			if factRunner == nil {
				return
			}
			bgCtx, cancel := context.WithTimeout(context.Background(), factExtractTimeout)
			defer cancel()

			facts, err := dependency.Do(bgCtx, dependency.MainLLM, "extract_facts", func(callCtx context.Context) ([]*model.FactCandidate, error) {
				return factRunner.Invoke(callCtx, &model.FactExtractInput{
					UserID:  userID,
					ConvID:  convID,
					Message: msg,
				})
			})
			if err != nil {
				log.Printf("fact extract failed user=%d conv=%s: %v", userID, convID, err)
				return
			}
			for _, f := range facts {
				if f == nil || f.Confidence < 0.5 {
					continue
				}
				if _, err := dbm.ResolveFactWithOutbox(bgCtx, memory.FactWrite{
					UserID: userID, FactType: f.FactType, ConflictKey: f.ConflictKey,
					Content: f.Content, Confidence: f.Confidence, Source: "conversation", SourceConv: convID,
				}); err != nil {
					log.Printf("insert fact failed user=%d type=%s: %v", userID, f.FactType, err)
				}
			}
		}(userID, convID, originalMsg)
	}

	succeeded = true
	return resp.Content, nil
}

func shouldExtractFacts(ctx context.Context) bool {
	return !isEvalReplay(ctx)
}

func isEvalReplay(ctx context.Context) bool {
	evalVariant, _ := metainfo.GetValue(ctx, "agent-eval-variant")
	if evalVariant == "" {
		evalVariant, _ = metainfo.GetPersistentValue(ctx, "agent-eval-variant")
	}
	return strings.TrimSpace(evalVariant) != ""
}

func isComplexPlanningRequest(message string) bool {
	text := strings.ToLower(strings.TrimSpace(message))
	if text == "" {
		return false
	}
	planning := strings.Contains(text, "计划") || strings.Contains(text, "规划") || strings.Contains(text, "安排") || strings.Contains(text, "plan")
	multiStep := strings.Contains(text, "分步骤") || strings.Contains(text, "多步骤") || strings.Contains(text, "本周") || strings.Contains(text, "下周") || strings.Contains(text, "课表") || strings.Contains(text, "薄弱") || strings.Contains(text, "next week") || strings.Contains(text, "multiple step") || strings.Contains(text, "multi-step")
	return planning && multiStep
}
