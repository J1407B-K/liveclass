package session

import (
	"context"
	"testing"

	"liveclass/internal/rpc/agent/model"
)

type memoryStore struct {
	events []model.TranscriptEvent
	cp     *model.SummaryCheckpoint
}

func (s *memoryStore) AppendTranscriptEvent(_ context.Context, e *model.TranscriptEvent) error {
	s.events = append(s.events, *e)
	return nil
}
func (s *memoryStore) ListTranscriptEvents(_ context.Context, _ string, after int64) ([]model.TranscriptEvent, error) {
	var out []model.TranscriptEvent
	for _, e := range s.events {
		if e.ID > after {
			out = append(out, e)
		}
	}
	return out, nil
}
func (s *memoryStore) LatestSummaryCheckpoint(context.Context, string) (*model.SummaryCheckpoint, error) {
	return s.cp, nil
}
func (s *memoryStore) SaveSummaryCheckpoint(_ context.Context, cp *model.SummaryCheckpoint) error {
	s.cp = cp
	return nil
}

func TestManagerCompactsPrefixAndKeepsTail(t *testing.T) {
	store := &memoryStore{}
	for i := 1; i <= 8; i++ {
		store.events = append(store.events, model.TranscriptEvent{ID: int64(i), UserID: 7, SessionID: "s", RequestID: "r", EventKey: string(rune('a' + i)), EventType: "user_message", Content: "课堂问题内容", EstimatedTokens: 6})
	}
	b := DefaultBudget()
	b.CompactionTrigger = 20
	b.RecentTail = 12
	m := NewManager(store, NewBuilder(b), DeterministicCompactor{MaxTokens: 20})
	ctx, err := m.Recover(context.Background(), "s", BuildInput{})
	if err != nil {
		t.Fatal(err)
	}
	if store.cp == nil || store.cp.SourceEventEndID != 6 {
		t.Fatalf("checkpoint = %#v, want end=6", store.cp)
	}
	if len(ctx.History) != 3 { // checkpoint plus two-event tail
		t.Fatalf("history length = %d, want 3", len(ctx.History))
	}
}
