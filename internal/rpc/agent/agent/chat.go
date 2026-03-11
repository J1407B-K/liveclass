package agent

import (
	"context"
	"liveclass/internal/rpc/agent/model"

	_type "liveclass/internal/rpc/agent/eino_gen/agent/type"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"

	"github.com/cloudwego/eino/schema"
)

func ChatWithAgent(
	ctx context.Context,
	dbm *memory.DBManager,
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

	userMsg := &_type.UserMessage{
		ID:      userID,
		Query:   msg,
		History: history,
	}

	resp, err := global.AgentRunner.Invoke(ctx, userMsg)
	if err != nil {
		return "", err
	}

	if err = dbm.AppendMessage(ctx, userID, convID, requestID, schema.UserMessage(msg)); err != nil {
		return "", err
	}

	if err = dbm.AppendMessage(ctx, userID, convID, requestID, resp); err != nil {
		return "", err
	}

	go func() {
		bgctx := context.Background()

		facts, err := ExtractFacts(bgctx, msg)
		if err != nil {
			return
		}

		for _, f := range facts {
			_, _ = dbm.InsertFactWithOutbox(
				bgctx,
				userID,
				f.FactType,
				f.Content,
				f.Confidence,
				convID,
			)
		}
	}()

	return resp.Content, nil
}

func ExtractFacts(ctx context.Context, msg string) ([]model.FactCandidate, error) {
	//TODO
	return nil, nil
}
