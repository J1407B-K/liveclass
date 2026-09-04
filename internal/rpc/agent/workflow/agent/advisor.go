package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"liveclass/internal/rpc/agent/agentmetrics"
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
		dependency.FallbackContext(ctx, dependency.AdvisorLLM, "select_skill")
		return applySkills(input, fallbackSkills(input))
	}

	if global.ChatModel == nil {
		dependency.FallbackContext(ctx, dependency.AdvisorLLM, "select_skill")
		return applySkills(input, fallbackSkills(input))
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
		dependency.FallbackContext(ctx, dependency.AdvisorLLM, "select_skill")
		return applySkills(input, fallbackSkills(input))
	}
	recordModelUsage(ctx, "advisor", resp)

	advice, err := parseAdvisorResponse(resp.Content)
	if err != nil || len(advice.Skills) == 0 {
		log.Printf("advisor parse failed: %v (raw: %q), falling back to general", err, resp.Content)
		dependency.FallbackContext(ctx, dependency.AdvisorLLM, "select_skill")
		return applySkills(input, fallbackSkills(input))
	}

	advice.Skills = normalizeSkills(input, advice.Skills)
	return applyAdvice(input, advice)
}

func normalizeSkills(input *model.UserMessage, skills []string) []string {
	if input != nil && input.Lesson > 0 && len(skills) == 1 && skills[0] == "general" {
		return []string{"student_qa"}
	}
	return skills
}

func fallbackSkills(input *model.UserMessage) []string {
	if input == nil {
		return []string{"general"}
	}
	if runtimeRequiresPlan(input) {
		return []string{"lesson_plan"}
	}
	if input.Lesson > 0 {
		return []string{"student_qa"}
	}
	return []string{"general"}
}

func applySkills(input *model.UserMessage, skills []string) (*model.UserMessage, error) {
	return applyAdvice(input, advisorDecision{Skills: skills})
}

func applyAdvice(input *model.UserMessage, advice advisorDecision) (*model.UserMessage, error) {
	var parts []string
	for _, name := range advice.Skills {
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
		advice.Skills = []string{"general"}
	}
	input.SkillAdvice = &model.SkillAdvice{
		Skills:         advice.Skills,
		Guidance:       strings.Join(parts, "\n\n---\n\n"),
		RequiresPlan:   advice.RequiresPlan,
		Complexity:     advice.Complexity,
		Reason:         advice.Reason,
		EstimatedSteps: advice.EstimatedSteps,
	}
	for _, skill := range advice.Skills {
		agentmetrics.SkillRoutes.WithLabelValues(boundedSkillLabel(skill), "selected").Inc()
	}
	return input, nil
}

func boundedSkillLabel(skill string) string {
	switch skill {
	case "general", "lesson_plan", "lesson_summary", "quiz_help", "student_qa":
		return skill
	default:
		return "unknown"
	}
}

type advisorDecision struct {
	Skills         []string `json:"skills"`
	Complexity     string   `json:"complexity"`
	RequiresPlan   bool     `json:"requires_plan"`
	Reason         string   `json:"reason"`
	EstimatedSteps int      `json:"estimated_steps"`
}

func parseAdvisorResponse(raw string) (advisorDecision, error) {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start == -1 || end == -1 || end <= start {
		return advisorDecision{}, fmt.Errorf("no JSON found in: %q", raw)
	}
	var result advisorDecision
	if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err != nil {
		return advisorDecision{}, err
	}
	return result, nil
}
