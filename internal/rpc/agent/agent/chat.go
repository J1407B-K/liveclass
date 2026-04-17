package agent

import (
	"context"
	"errors"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/model"
	userprofile "liveclass/internal/rpc/agent/profile"
	"liveclass/internal/rpc/agent/rag"
	"liveclass/internal/rpc/agent/rerank"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

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
	userID int64,
	convID string,
	requestID string,
	msg string,
	lessonID int64,
) (string, error) {
	if agentRunner == nil {
		return "", errors.New("nil agent runner")
	}

	// 幂等
	existedResp, err := dbm.GetAssistantMessageByRequestID(ctx, convID, requestID)
	if err != nil {
		return "", err
	}
	if existedResp != nil {
		return existedResp.Content, nil
	}

	history, err := dbm.GetRecentMessages(ctx, convID, 6)
	if err != nil {
		return "", err
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
		facts, ferr := dbm.RetrieveRelevantFacts(egCtx, userID, msg, 5)
		if ferr != nil || len(facts) == 0 {
			return nil // 降级：无 facts 不影响主流程
		}
		if ranked, rerr := rerank.Facts(egCtx, msg, facts, 5); rerr == nil && len(ranked) > 0 {
			facts = ranked
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
			return nil
		}
		chunks, retrErr := docRetriever.Search(egCtx, lessonID, vector, 6)
		if retrErr != nil {
			log.Printf("doc search error: %v", retrErr)
			return nil
		}
		if reranked, rerr := rerank.Docs(egCtx, msg, chunks, 3); rerr == nil && len(reranked) > 0 {
			chunks = reranked
		}
		var builder strings.Builder
		for _, chunk := range chunks {
			builder.WriteString("- 来源: ")
			if chunk.Source != "" {
				builder.WriteString(chunk.Source)
			} else {
				builder.WriteString("unknown")
			}
			builder.WriteString(" #段落")
			builder.WriteString(strconv.FormatInt(int64(chunk.ChunkIdx), 10))
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
		summary, perr := userprofile.EnsureUserProfile(egCtx, dbm, userID)
		if perr != nil {
			log.Printf("generate user profile failed: %v", perr)
			return nil
		}
		mu.Lock()
		profileSummary = summary
		mu.Unlock()
		return nil
	})

	// 等待三个并行操作完成（均为降级处理，不会返回错误）
	_ = eg.Wait()

	userMsg := &model.UserMessage{
		ID:      userID,
		Lesson:  lessonID,
		Query:   msg,
		History: history,
		Facts:   factText,
		Profile: profileSummary,
		Docs:    docText,
	}

	resp, err := agentRunner.Invoke(ctx, userMsg)
	if err != nil {
		return "", err
	}

	if err = dbm.AppendMessage(ctx, userID, convID, requestID, schema.UserMessage(msg)); err != nil {
		return "", err
	}
	if err = dbm.AppendMessage(ctx, userID, convID, requestID, resp); err != nil {
		return "", err
	}

	// 异步 fact 提取：使用独立超时，避免 goroutine 泄漏
	go func(userID int64, convID, msg string) {
		if factRunner == nil {
			return
		}
		bgCtx, cancel := context.WithTimeout(context.Background(), factExtractTimeout)
		defer cancel()

		facts, err := factRunner.Invoke(bgCtx, &model.FactExtractInput{
			UserID:  userID,
			ConvID:  convID,
			Message: msg,
		})
		if err != nil {
			log.Printf("fact extract failed user=%d conv=%s: %v", userID, convID, err)
			return
		}
		for _, f := range facts {
			if f == nil || f.Confidence < 0.5 {
				continue
			}
			if _, err := dbm.InsertFactWithOutbox(bgCtx, userID, f.FactType, f.Content, f.Confidence, convID); err != nil {
				log.Printf("insert fact failed user=%d type=%s: %v", userID, f.FactType, err)
			}
		}
	}(userID, convID, msg)

	return resp.Content, nil
}
