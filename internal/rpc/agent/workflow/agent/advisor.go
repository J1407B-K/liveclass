package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/model"
	my_prompt "liveclass/internal/rpc/agent/prompt"
	"log"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// runAdvisor 调用 LLM 对用户意图进行分类，返回技能类型和执行指引。
// 这是一次轻量级 LLM 调用，不走完整的 React Agent 流程。
func runAdvisor(ctx context.Context, input *model.UserMessage, _ ...any) (*model.UserMessage, error) {
	if global.ChatModel == nil {
		// 降级：没有模型时直接走 general 技能
		input.SkillAdvice = &model.SkillAdvice{
			Skill:    my_prompt.SkillGeneral,
			Guidance: my_prompt.SkillPrompts[my_prompt.SkillGeneral],
		}
		return input, nil
	}

	userContent := input.Query
	// 附带最近一轮完整对话（user+assistant）避免断章取义
	if len(input.History) >= 2 {
		prev := input.History[len(input.History)-2]
		last := input.History[len(input.History)-1]
		userContent = "（上一轮对话：\n" + string(prev.Role) + ": " + prev.Content +
			"\n" + string(last.Role) + ": " + last.Content + "）\n当前消息：" + userContent
	} else if len(input.History) == 1 {
		h := input.History[0]
		userContent = "（上一条消息：" + string(h.Role) + ": " + h.Content + "）\n当前消息：" + userContent
	}

	msgs := []*schema.Message{
		schema.SystemMessage(my_prompt.AdvisorSystemPrompt),
		schema.UserMessage(userContent),
	}

	resp, err := global.ChatModel.Generate(ctx, msgs)
	if err != nil {
		log.Printf("advisor LLM call failed: %v, falling back to general", err)
		input.SkillAdvice = fallbackAdvice()
		return input, nil
	}

	advice, err := parseAdvisorResponse(resp.Content)
	if err != nil {
		log.Printf("advisor parse failed: %v (raw: %q), falling back to general", err, resp.Content)
		input.SkillAdvice = fallbackAdvice()
		return input, nil
	}

	// 用 Skill Prompt 库里的完整 SOP 替换 LLM 给的简短 guidance，
	// LLM 给的 guidance 只作为补充说明追加在后面。
	skillPrompt, ok := my_prompt.SkillPrompts[advice.Skill]
	if !ok {
		skillPrompt = my_prompt.SkillPrompts[my_prompt.SkillGeneral]
	}
	if advice.Guidance != "" {
		skillPrompt = skillPrompt + "\n**本次任务补充说明：** " + advice.Guidance
	}

	input.SkillAdvice = &model.SkillAdvice{
		Skill:    advice.Skill,
		Guidance: skillPrompt,
	}
	return input, nil
}

func parseAdvisorResponse(raw string) (*model.SkillAdvice, error) {
	// 尝试从输出中提取 JSON（LLM 有时会带前后缀文字）
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("advisor: no JSON object found in response: %q", raw)
	}
	raw = raw[start : end+1]

	var advice model.SkillAdvice
	if err := json.Unmarshal([]byte(raw), &advice); err != nil {
		return nil, err
	}
	if advice.Skill == "" {
		advice.Skill = my_prompt.SkillGeneral
	}
	return &advice, nil
}

func fallbackAdvice() *model.SkillAdvice {
	return &model.SkillAdvice{
		Skill:    my_prompt.SkillGeneral,
		Guidance: my_prompt.SkillPrompts[my_prompt.SkillGeneral],
	}
}
