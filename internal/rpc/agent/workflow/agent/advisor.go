package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/model"
	my_prompt "liveclass/internal/rpc/agent/prompt"
	"log"
	"strings"

	"github.com/cloudwego/eino/schema"
)

func runAdvisor(ctx context.Context, input *model.UserMessage, _ ...any) (*model.UserMessage, error) {
	index, err := my_prompt.LoadSkillIndex()
	if err != nil || len(index) == 0 {
		log.Printf("advisor: failed to load tool index: %v, falling back to general", err)
		dependency.Fallback(dependency.AdvisorLLM, "select_skill")
		return applySkills(input, []string{"general"})
	}

	if global.ChatModel == nil {
		dependency.Fallback(dependency.AdvisorLLM, "select_skill")
		return applySkills(input, []string{"general"})
	}

	userContent := input.Query
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
		schema.SystemMessage(my_prompt.BuildAdvisorSystemPrompt(index)),
		schema.UserMessage(userContent),
	}

	resp, err := dependency.Do(ctx, dependency.AdvisorLLM, "select_skill", func(callCtx context.Context) (*schema.Message, error) {
		return global.ChatModel.Generate(callCtx, msgs)
	})
	if err != nil {
		log.Printf("advisor LLM call failed: %v, falling back to general", err)
		dependency.Fallback(dependency.AdvisorLLM, "select_skill")
		return applySkills(input, []string{"general"})
	}

	skills, err := parseAdvisorResponse(resp.Content)
	if err != nil || len(skills) == 0 {
		log.Printf("advisor parse failed: %v (raw: %q), falling back to general", err, resp.Content)
		dependency.Fallback(dependency.AdvisorLLM, "select_skill")
		return applySkills(input, []string{"general"})
	}

	return applySkills(input, skills)
}

func applySkills(input *model.UserMessage, skills []string) (*model.UserMessage, error) {
	var parts []string
	for _, name := range skills {
		content, err := my_prompt.LoadSkillContent(name)
		if err != nil {
			log.Printf("advisor: tool %q not found, skipping: %v", name, err)
			continue
		}
		parts = append(parts, content)
	}
	if len(parts) == 0 {
		content, _ := my_prompt.LoadSkillContent("general")
		parts = []string{content}
		skills = []string{"general"}
	}
	input.SkillAdvice = &model.SkillAdvice{
		Skills:   skills,
		Guidance: strings.Join(parts, "\n\n---\n\n"),
	}
	return input, nil
}

func parseAdvisorResponse(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, fmt.Errorf("no JSON found in: %q", raw)
	}
	var result struct {
		Skills []string `json:"skills"`
	}
	if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err != nil {
		return nil, err
	}
	return result.Skills, nil
}
