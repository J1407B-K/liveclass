package profile

import (
	"context"
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/model"
	my_prompt "liveclass/internal/rpc/agent/prompt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/schema"
	"golang.org/x/sync/singleflight"
)

var sfGroup singleflight.Group

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

	// 缓存有效：直接返回
	if profile != nil && profile.Summary != "" && time.Since(profile.UpdatedAt) < profileTTL {
		return profile.Summary, nil
	}

	// 缓存过期但有旧值：先返回旧值，后台异步刷新，不阻塞请求热路径
	if profile != nil && profile.Summary != "" {
		go refreshProfile(dbm, userID)
		return profile.Summary, nil
	}

	// 首次生成：同步执行，singleflight 防并发重复 LLM 调用
	key := fmt.Sprintf("profile:%d", userID)
	v, err, _ := sfGroup.Do(key, func() (interface{}, error) {
		return generateAndSaveProfile(ctx, dbm, userID)
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func refreshProfile(dbm *memory.DBManager, userID int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := generateAndSaveProfile(ctx, dbm, userID); err != nil {
		log.Printf("async profile refresh failed user=%d: %v", userID, err)
	}
}

func generateAndSaveProfile(ctx context.Context, dbm *memory.DBManager, userID int64) (string, error) {
	facts, err := dbm.ListFactsForProfile(ctx, userID, profileFactLimit, profileMinConfidence)
	if err != nil {
		return "", err
	}
	if len(facts) == 0 {
		return "", nil
	}

	summary, err := summarizeFacts(ctx, facts)
	if err != nil || strings.TrimSpace(summary) == "" {
		return "", err
	}

	summary = strings.TrimSpace(summary)
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
