package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"liveclass/internal/rpc/agent/agentmetrics"
	"liveclass/internal/rpc/agent/model"
)

type Store interface {
	AppendTranscriptEvent(context.Context, *model.TranscriptEvent) error
	ListTranscriptEvents(context.Context, string, int64) ([]model.TranscriptEvent, error)
	LatestSummaryCheckpoint(context.Context, string) (*model.SummaryCheckpoint, error)
	SaveSummaryCheckpoint(context.Context, *model.SummaryCheckpoint) error
}

type taskStateStore interface {
	ActiveTaskPlanContext(context.Context, string) (string, error)
}

type Summary struct {
	Summary         string   `json:"summary"`
	ImportantFacts  []string `json:"important_facts"`
	Decisions       []string `json:"decisions"`
	UnfinishedTasks []string `json:"unfinished_tasks"`
}

type Compactor interface {
	Compact(context.Context, *model.SummaryCheckpoint, []model.TranscriptEvent) (Summary, error)
}

type Manager struct {
	store     Store
	builder   *Builder
	compactor Compactor
}

func NewManager(store Store, builder *Builder, compactor Compactor) *Manager {
	if builder == nil {
		builder = NewBuilder(DefaultBudget())
	}
	return &Manager{store: store, builder: builder, compactor: compactor}
}

func (m *Manager) AppendMessage(ctx context.Context, userID int64, sessionID, requestID, eventKey, eventType, role, content string) error {
	if m == nil || m.store == nil {
		return errors.New("nil session manager")
	}
	return m.store.AppendTranscriptEvent(ctx, &model.TranscriptEvent{
		UserID: userID, SessionID: sessionID, RequestID: requestID,
		EventKey: eventKey, EventType: eventType, Role: role, Content: content,
		Payload: "{}", EstimatedTokens: EstimateTextTokens(content),
	})
}

// Recover rebuilds the current working set from persisted transcript and the
// latest checkpoint. It optionally compacts an old prefix while retaining a
// high-fidelity tail.
func (m *Manager) Recover(ctx context.Context, sessionID string, input BuildInput) (Context, error) {
	if m == nil || m.store == nil {
		return Context{}, errors.New("nil session manager")
	}
	cp, err := m.store.LatestSummaryCheckpoint(ctx, sessionID)
	if err != nil {
		return Context{}, err
	}
	after := int64(0)
	if cp != nil {
		after = cp.SourceEventEndID
	}
	events, err := m.store.ListTranscriptEvents(ctx, sessionID, after)
	if err != nil {
		return Context{}, err
	}

	if m.compactor != nil && eventTokens(events) > m.builder.Budget.CompactionTrigger {
		prefix, tail := splitForTail(events, m.builder.Budget.RecentTail)
		if len(prefix) > 0 {
			summary, compactErr := m.compactor.Compact(ctx, cp, prefix)
			if compactErr == nil && strings.TrimSpace(summary.Summary) != "" {
				newCP := checkpointFromSummary(prefix[0].UserID, sessionID, cp, prefix, summary)
				if err = m.store.SaveSummaryCheckpoint(ctx, newCP); err != nil {
					return Context{}, err
				}
				_ = m.store.AppendTranscriptEvent(ctx, &model.TranscriptEvent{
					UserID: newCP.UserID, SessionID: sessionID, RequestID: prefix[len(prefix)-1].RequestID,
					EventKey: fmt.Sprintf("summary:%d", newCP.SourceEventEndID), EventType: "summary_event",
					Role: "system", Content: newCP.Summary, Payload: "{}", EstimatedTokens: newCP.EstimatedTokens,
				})
				cp, events = newCP, tail
				agentmetrics.Compactions.WithLabelValues("success").Inc()
			} else {
				agentmetrics.Compactions.WithLabelValues("fallback_or_error").Inc()
			}
		}
	}
	if taskStore, ok := m.store.(taskStateStore); ok && strings.TrimSpace(input.TaskState) == "" {
		input.TaskState, err = taskStore.ActiveTaskPlanContext(ctx, sessionID)
		if err != nil {
			return Context{}, err
		}
	}

	input.Checkpoint = cp
	input.Events = events
	return m.builder.Build(input), nil
}

func eventTokens(events []model.TranscriptEvent) int {
	total := 0
	for _, e := range events {
		if e.EstimatedTokens > 0 {
			total += e.EstimatedTokens
		} else {
			total += EstimateTextTokens(e.Content)
		}
	}
	return total
}

func splitForTail(events []model.TranscriptEvent, tailBudget int) ([]model.TranscriptEvent, []model.TranscriptEvent) {
	used, split := 0, len(events)
	for i := len(events) - 1; i >= 0; i-- {
		cost := events[i].EstimatedTokens
		if cost <= 0 {
			cost = EstimateTextTokens(events[i].Content)
		}
		if used+cost > tailBudget {
			break
		}
		used += cost
		split = i
	}
	return events[:split], events[split:]
}

func checkpointFromSummary(userID int64, sessionID string, previous *model.SummaryCheckpoint, events []model.TranscriptEvent, summary Summary) *model.SummaryCheckpoint {
	start := events[0].ID
	if previous != nil && previous.SourceEventStartID > 0 {
		start = previous.SourceEventStartID
	}
	marshal := func(v []string) string {
		b, _ := json.Marshal(v)
		return string(b)
	}
	return &model.SummaryCheckpoint{
		UserID: userID, SessionID: sessionID, Summary: summary.Summary,
		ImportantFacts: marshal(summary.ImportantFacts), Decisions: marshal(summary.Decisions),
		UnfinishedTasks: marshal(summary.UnfinishedTasks), SourceEventStartID: start,
		SourceEventEndID: events[len(events)-1].ID, EstimatedTokens: EstimateTextTokens(summary.Summary),
	}
}

// DeterministicCompactor is a safe fallback for tests and model outages. It
// keeps bounded observable excerpts and never invents facts or decisions.
type DeterministicCompactor struct{ MaxTokens int }

func (c DeterministicCompactor) Compact(_ context.Context, previous *model.SummaryCheckpoint, events []model.TranscriptEvent) (Summary, error) {
	maxTokens := c.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 1200
	}
	var parts []string
	if previous != nil && previous.Summary != "" {
		parts = append(parts, "已有摘要："+previous.Summary)
	}
	for _, e := range events {
		if e.EventType != "user_message" && e.EventType != "assistant_message" && e.EventType != "tool_result" {
			continue
		}
		content, _ := truncateToTokens(e.Content, 160)
		parts = append(parts, fmt.Sprintf("%s: %s", e.EventType, content))
	}
	text, _ := truncateToTokens(strings.Join(parts, "\n"), maxTokens)
	return Summary{Summary: text}, nil
}
