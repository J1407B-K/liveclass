package trace

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"liveclass/internal/rpc/agent/model"
)

type Store interface {
	AppendTraceEvent(context.Context, *model.AgentTraceEvent) error
}

type transcriptStore interface {
	AppendTranscriptEvent(context.Context, *model.TranscriptEvent) error
}

type Run struct {
	RunID, SessionID, RequestID string
	UserID                      int64
	store                       Store
	mu                          sync.Mutex
	toolCalls                   map[string]int
	steps                       int
	errors, retries, fallbacks  int
	sequence                    int
}

func NewRun(store Store, runID, sessionID, requestID string, userID int64) *Run {
	return &Run{store: store, RunID: runID, SessionID: sessionID, RequestID: requestID, UserID: userID, toolCalls: make(map[string]int)}
}

func (r *Run) RecordTranscript(ctx context.Context, eventType, name, content string, metadata map[string]any) {
	if r == nil {
		return
	}
	store, ok := r.store.(transcriptStore)
	if !ok {
		return
	}
	r.mu.Lock()
	r.sequence++
	sequence := r.sequence
	r.mu.Unlock()
	raw, err := json.Marshal(metadata)
	if err != nil {
		raw = []byte("{}")
	}
	_ = store.AppendTranscriptEvent(ctx, &model.TranscriptEvent{
		UserID: r.UserID, SessionID: r.SessionID, RequestID: r.RequestID,
		EventKey: fmt.Sprintf("%s:%s:%d", eventType, name, sequence), EventType: eventType,
		Content: content, Payload: string(raw),
	})
}

type contextKey struct{}

func WithRun(ctx context.Context, run *Run) context.Context {
	return context.WithValue(ctx, contextKey{}, run)
}
func FromContext(ctx context.Context) *Run {
	run, _ := ctx.Value(contextKey{}).(*Run)
	return run
}

func (r *Run) Record(ctx context.Context, eventType, name, status string, duration time.Duration, metadata map[string]any) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.steps++
	if eventType == "tool_result" {
		r.toolCalls[name]++
		if status == "error" || status == "denied" {
			r.errors++
		}
	}
	if eventType == "retry" {
		r.retries++
	}
	if eventType == "fallback" {
		r.fallbacks++
	}
	r.mu.Unlock()
	if r.store == nil {
		return
	}
	raw, err := json.Marshal(metadata)
	if err != nil {
		raw = []byte(`{"metadata_error":"marshal failed"}`)
	}
	_ = r.store.AppendTraceEvent(ctx, &model.AgentTraceEvent{
		RunID: r.RunID, SessionID: r.SessionID, RequestID: r.RequestID,
		EventType: eventType, Name: name, Status: status, Metadata: string(raw), DurationMs: duration.Milliseconds(),
	})
}

func (r *Run) Stats() (duplicates, retries, fallbacks, toolErrors int) {
	if r == nil {
		return 0, 0, 0, 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, count := range r.toolCalls {
		if count > 1 {
			duplicates += count - 1
		}
	}
	return duplicates, r.retries, r.fallbacks, r.errors
}

func (r *Run) ToolCallCount(name string) int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.toolCalls[name]
}

func (r *Run) Steps() int {
	if r == nil {
		return 0
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.steps
}
