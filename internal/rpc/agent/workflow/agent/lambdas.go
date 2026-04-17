package agent

import (
	"context"
	"fmt"
	"liveclass/internal/rpc/agent/model"
	my_prompt "liveclass/internal/rpc/agent/prompt"
	"time"
)

func newInputToTemplateVars(_ context.Context, input *model.UserMessage, opts ...any) (map[string]any, error) {
	facts := input.Facts
	if facts == "" {
		facts = "暂无相关长期记忆。"
	}
	profile := input.Profile
	if profile == "" {
		profile = "暂无画像数据。"
	}
	docs := input.Docs
	if docs == "" {
		docs = "暂无课程资料。"
	}

	// 从 Advisor 获取技能指引，未命中时降级为 general
	skillGuidance := my_prompt.SkillPrompts[my_prompt.SkillGeneral]
	if input.SkillAdvice != nil && input.SkillAdvice.Guidance != "" {
		skillGuidance = input.SkillAdvice.Guidance
	}

	return map[string]any{
		"content":        input.Query,
		"history":        input.History,
		"facts":          facts,
		"profile":        profile,
		"docs":           docs,
		"user_id":        fmt.Sprintf("%d", input.ID),
		"date":           time.Now().Format("2006-01-02 15:04:05"),
		"skill_guidance": skillGuidance,
	}, nil
}
