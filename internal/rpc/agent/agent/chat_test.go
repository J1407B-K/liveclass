package agent

import (
	"context"
	"testing"

	"github.com/bytedance/gopkg/cloud/metainfo"
)

func TestShouldExtractFactsSkipsEvalReplay(t *testing.T) {
	if !shouldExtractFacts(context.Background()) {
		t.Fatal("normal requests must extract facts")
	}
	ctx := metainfo.WithPersistentValue(context.Background(), "agent-eval-variant", "v2")
	if shouldExtractFacts(ctx) {
		t.Fatal("eval replay must not write facts shared by later cases")
	}
}

func TestIsEvalReplay(t *testing.T) {
	if isEvalReplay(context.Background()) {
		t.Fatal("normal request detected as eval")
	}
	ctx := metainfo.WithPersistentValue(context.Background(), "agent-eval-variant", "v1")
	if !isEvalReplay(ctx) {
		t.Fatal("eval metadata was not detected")
	}
}

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
