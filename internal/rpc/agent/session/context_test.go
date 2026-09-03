package session

import (
	"strings"
	"testing"

	"liveclass/internal/rpc/agent/model"
)

func TestEstimateTextTokens(t *testing.T) {
	if got := EstimateTextTokens("abcdefgh"); got != 2 {
		t.Fatalf("ascii estimate = %d, want 2", got)
	}
	if got := EstimateTextTokens("课堂聊天"); got != 4 {
		t.Fatalf("CJK estimate = %d, want 4", got)
	}
}

func TestBuilderUsesCheckpointAndNewestEventsWithinBudget(t *testing.T) {
	b := NewBuilder(Budget{ModelContext: 100, SystemReserve: 10, OutputReserve: 10, RAG: 10, Memory: 10, Conversation: 60, CompactionTrigger: 45, RecentTail: 15, MaxToolResult: 5})
	ctx := b.Build(BuildInput{
		Checkpoint: &model.SummaryCheckpoint{Summary: "用户正在复习 Go 并决定先学习并发。"},
		Events: []model.TranscriptEvent{
			{ID: 1, EventType: "user_message", Content: strings.Repeat("旧", 50)},
			{ID: 2, EventType: "assistant_message", Content: "最近回答"},
			{ID: 3, EventType: "user_message", Content: "最近问题"},
		},
	})
	if len(ctx.History) < 3 {
		t.Fatalf("history length = %d, want checkpoint plus recent pair", len(ctx.History))
	}
	if ctx.History[len(ctx.History)-1].Content != "最近问题" {
		t.Fatalf("newest event was not retained: %#v", ctx.History)
	}
	if !ctx.Truncated {
		t.Fatal("expected oversized old event to be excluded")
	}
}

func TestToolResultIsBounded(t *testing.T) {
	b := NewBuilder(DefaultBudget())
	b.Budget.MaxToolResult = 3
	ctx := b.Build(BuildInput{Events: []model.TranscriptEvent{{ID: 1, EventType: "tool_result", EventKey: "search", Content: strings.Repeat("结果", 20)}}})
	if len(ctx.History) != 1 || !strings.Contains(ctx.History[0].Content, "截断") {
		t.Fatalf("tool result was not bounded: %#v", ctx.History)
	}
}

func TestCurrentRequestIsBoundedBeforeHistory(t *testing.T) {
	b := NewBuilder(Budget{ModelContext: 100, SystemReserve: 10, OutputReserve: 10, RAG: 10, Memory: 10, Conversation: 60, CompactionTrigger: 45, RecentTail: 15, MaxToolResult: 5})
	ctx := b.Build(BuildInput{CurrentRequest: strings.Repeat("问", 100), Events: []model.TranscriptEvent{{ID: 1, EventType: "assistant_message", Content: "old"}}})
	if !ctx.Truncated || EstimateTextTokens(ctx.CurrentRequest) > 60 {
		t.Fatalf("current request was not bounded: tokens=%d", EstimateTextTokens(ctx.CurrentRequest))
	}
	if len(ctx.History) != 0 {
		t.Fatalf("history should yield to oversized current request: %#v", ctx.History)
	}
}

func TestBuilderInjectsRecoveredTaskState(t *testing.T) {
	b := NewBuilder(DefaultBudget())
	ctx := b.Build(BuildInput{TaskState: "目标：复习并发（running）\n- [done] review: 回顾\n- [running] practice: 练习"})
	if len(ctx.History) != 1 || !strings.Contains(ctx.History[0].Content, "当前任务计划状态") || !strings.Contains(ctx.History[0].Content, "practice") {
		t.Fatalf("task state was not injected: %#v", ctx.History)
	}
}
