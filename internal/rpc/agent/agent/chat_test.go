package agent

import "testing"

func TestIsComplexPlanningRequest(t *testing.T) {
	tests := []struct {
		message string
		want    bool
	}{
		{message: "根据我的薄弱知识点和下周课表，制定一份分步骤复习计划", want: true},
		{message: "帮我制定计划", want: false},
		{message: "这节课讲了什么", want: false},
		{message: "plan my next week in multiple steps", want: true},
	}
	for _, tt := range tests {
		if got := isComplexPlanningRequest(tt.message); got != tt.want {
			t.Errorf("isComplexPlanningRequest(%q) = %v, want %v", tt.message, got, tt.want)
		}
	}
}
