package profile

import (
	"context"
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/model"
	my_prompt "liveclass/internal/rpc/agent/prompt"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
)

const (
	profileTTL           = 12 * time.Hour
	profileFactLimit     = 12
	profileMinConfidence = 0.4
)

func EnsureUserProfile(ctx context.Context, dbm *memory.DBManager, userID int64) (string, error) {
	profile, err := dbm.GetUserProfile(ctx, userID)
	if err != nil {
		return "", err
	}
	if profile != nil && profile.Summary != "" && time.Since(profile.UpdatedAt) < profileTTL {
		return profile.Summary, nil
	}

	facts, err := dbm.ListFactsForProfile(ctx, userID, profileFactLimit, profileMinConfidence)
	if err != nil {
		return "", err
	}

	if len(facts) == 0 {
		if profile != nil {
			return profile.Summary, nil
		}
		return "", nil
	}

	summary, err := summarizeFacts(ctx, facts)
	if err != nil {
		if profile != nil {
			// Return stale profile if generation failed.
			return profile.Summary, nil
		}
		return "", err
	}

	summary = strings.TrimSpace(summary)
	if summary == "" {
		if profile != nil {
			return profile.Summary, nil
		}
		return "", nil
	}

	if err := dbm.UpsertUserProfile(ctx, userID, summary); err != nil {
		return summary, err
	}

	return summary, nil
}

func summarizeFacts(ctx context.Context, facts []*model.UserFact) (string, error) {
	block := formatFactsBlock(facts)
	if block == "" {
		return "", nil
	}

	if global.ChatModel == nil {
		return block, nil
	}

	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: my_prompt.ProfileSystemPrompt,
		},
		{
			Role:    schema.User,
			Content: fmt.Sprintf(my_prompt.ProfileUserPrompt, block),
		},
	}

	resp, err := global.ChatModel.Generate(ctx, messages)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Content), nil
}

func formatFactsBlock(facts []*model.UserFact) string {
	var b strings.Builder
	counter := 1
	for _, f := range facts {
		if f == nil || f.Content == "" {
			continue
		}
		factType := strings.TrimSpace(f.FactType)
		if factType == "" {
			factType = "general"
		}
		fmt.Fprintf(&b, "%d. [%s] %s\n", counter, factType, strings.TrimSpace(f.Content))
		counter++
	}

	return strings.TrimSpace(b.String())
}
