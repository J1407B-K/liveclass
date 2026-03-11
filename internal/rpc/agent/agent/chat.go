package agent

import (
	"context"
	"errors"
	"fmt"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/model"
	userprofile "liveclass/internal/rpc/agent/profile"
	"liveclass/internal/rpc/agent/rerank"
	"log"

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
	userID int64,
	convID string,
	requestID string,
	msg string,
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

	var factText string
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

	profileSummary, err := userprofile.EnsureUserProfile(ctx, dbm, userID)
	if err != nil {
		log.Println("generate user profile failed:", err)
	}

	userMsg := &model.UserMessage{
		ID:      userID,
		Query:   msg,
		History: history,
		Facts:   factText,
		Profile: profileSummary,
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
