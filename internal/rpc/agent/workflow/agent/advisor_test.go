package agent

import (
	"testing"

	"liveclass/internal/rpc/agent/model"
)

func TestFallbackSkills(t *testing.T) {
	tests := []struct {
		name string
		in   *model.UserMessage
		want string
	}{
		{name: "lesson question", in: &model.UserMessage{Lesson: 9, Query: "这节课讲了什么"}, want: "student_qa"},
		{name: "complex plan", in: &model.UserMessage{Query: "根据薄弱点制定下周分步骤复习计划"}, want: "lesson_plan"},
		{name: "greeting", in: &model.UserMessage{Query: "你好"}, want: "general"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fallbackSkills(tt.in)
			if len(got) != 1 || got[0] != tt.want {
				t.Fatalf("fallbackSkills() = %v, want %s", got, tt.want)
			}
		})
	}
}

func TestNormalizeSkillsUsesLessonScope(t *testing.T) {
	got := normalizeSkills(&model.UserMessage{Lesson: 7}, []string{"general"})
	if len(got) != 1 || got[0] != "student_qa" {
		t.Fatalf("normalizeSkills() = %v", got)
	}
	got = normalizeSkills(&model.UserMessage{}, []string{"general"})
	if len(got) != 1 || got[0] != "general" {
		t.Fatalf("unscoped general route changed: %v", got)
	}
}

func TestParseAdvisorPlanningDecision(t *testing.T) {
	got, err := parseAdvisorResponse(`{"skills":["lesson_plan"],"complexity":"complex","requires_plan":true,"reason":"dependent steps","estimated_steps":4}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Skills) != 1 || got.Skills[0] != "lesson_plan" || !got.RequiresPlan || got.EstimatedSteps != 4 {
		t.Fatalf("unexpected advisor decision: %#v", got)
	}
}
