package agent

import (
	"context"
	"errors"
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/model"
	userprofile "liveclass/internal/rpc/agent/profile"
	"liveclass/internal/rpc/agent/rag"
	"liveclass/internal/rpc/agent/rerank"
	"log"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

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

	var (
		factText string
		docText  string
	)

	relevantFacts, err := dbm.RetrieveRelevantFacts(ctx, userID, msg, 5)
	if err == nil && len(relevantFacts) > 0 {
		rankedFacts, rerr := rerank.Facts(ctx, msg, relevantFacts, 5)
		if rerr != nil {
			log.Println("rerank facts failed:", rerr)
		} else if len(rankedFacts) > 0 {
			relevantFacts = rankedFacts
		}
		factText = memory.FormatFactsForPrompt(relevantFacts)
	}

	if lessonID > 0 && docRetriever != nil && embedder != nil {
		vector, embErr := embedder.EmbedText(ctx, msg)
		if embErr != nil {
			log.Printf("doc embed error: %v", embErr)
		} else {
			chunks, retrErr := docRetriever.Search(ctx, lessonID, vector, 6)
			if retrErr != nil {
				log.Printf("doc search error: %v", retrErr)
			} else {
				if reranked, rerr := rerank.Docs(ctx, msg, chunks, 3); rerr == nil && len(reranked) > 0 {
					chunks = reranked
				} else if rerr != nil {
					log.Printf("doc rerank error: %v", rerr)
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
				docText = strings.TrimSpace(builder.String())
			}
		}
	}

	profileSummary, err := userprofile.EnsureUserProfile(ctx, dbm, userID)
	if err != nil {
		log.Println("generate user profile failed:", err)
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

	if agentRunner == nil {
		return "", errors.New("nil agent runner")
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

	go func(userID int64, convID, msg string) {
		if factRunner == nil {
			return
		}

		bgctx := context.Background()

		input := &model.FactExtractInput{
			UserID:  userID,
			ConvID:  convID,
			Message: msg,
		}

		facts, err := factRunner.Invoke(bgctx, input)
		if err != nil {
			log.Println(err)
			return
		}

		fmt.Println(facts)

		for _, f := range facts {
			if f == nil {
				continue
			}
			if f.Confidence < 0.5 {
				continue
			}

			_, _ = dbm.InsertFactWithOutbox(
				bgctx,
				userID,
				f.FactType,
				f.Content,
				f.Confidence,
				convID,
			)
		}
	}(userID, convID, msg)

	return resp.Content, nil
}
