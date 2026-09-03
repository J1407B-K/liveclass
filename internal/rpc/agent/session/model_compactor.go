package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	modelcomponent "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"liveclass/internal/rpc/agent/agentmetrics"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/model"
)

type ModelCompactor struct {
	Model          modelcomponent.BaseChatModel
	RepairAttempts int
	Fallback       Compactor
}

func (c *ModelCompactor) Compact(ctx context.Context, previous *model.SummaryCheckpoint, events []model.TranscriptEvent) (Summary, error) {
	if c == nil || c.Model == nil {
		return c.fallback(ctx, previous, events, errors.New("nil compaction model"))
	}
	payload := struct {
		Previous string                  `json:"previous_summary,omitempty"`
		Events   []model.TranscriptEvent `json:"events"`
	}{Events: events}
	if previous != nil {
		payload.Previous = previous.Summary
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return Summary{}, err
	}

	messages := []*schema.Message{
		schema.SystemMessage("你是会话压缩器。只总结可观察事实，不推测，不输出思维过程。严格返回 JSON：{\"summary\":string,\"important_facts\":string[],\"decisions\":string[],\"unfinished_tasks\":string[]}。"),
		schema.UserMessage(string(raw)),
	}
	resp, err := c.generate(ctx, messages)
	if err != nil {
		return c.fallback(ctx, previous, events, err)
	}
	result, parseErr := parseSummary(resp.Content)
	attempts := c.RepairAttempts
	if attempts <= 0 {
		attempts = 2
	}
	for i := 0; parseErr != nil && i < attempts; i++ {
		agentmetrics.Repairs.WithLabelValues("session_compaction", "attempt").Inc()
		repairMessages := []*schema.Message{
			schema.SystemMessage("修复给定输出使其符合指定 JSON schema。只返回修复后的 JSON，不重新执行总结。"),
			schema.UserMessage(fmt.Sprintf("schema={summary:string,important_facts:string[],decisions:string[],unfinished_tasks:string[]}\nvalidation_error=%s\noriginal_output=%s", parseErr, resp.Content)),
		}
		resp, err = c.generate(ctx, repairMessages)
		if err != nil {
			break
		}
		result, parseErr = parseSummary(resp.Content)
	}
	if parseErr == nil {
		agentmetrics.Repairs.WithLabelValues("session_compaction", "success").Inc()
	}
	if parseErr != nil || err != nil {
		if err == nil {
			err = parseErr
		}
		return c.fallback(ctx, previous, events, err)
	}
	return result, nil
}

func (c *ModelCompactor) generate(ctx context.Context, messages []*schema.Message) (*schema.Message, error) {
	return dependency.Do(ctx, dependency.MainLLM, "compact_session", func(callCtx context.Context) (*schema.Message, error) {
		return c.Model.Generate(callCtx, messages)
	})
}

func (c *ModelCompactor) fallback(ctx context.Context, previous *model.SummaryCheckpoint, events []model.TranscriptEvent, cause error) (Summary, error) {
	dependency.Fallback(dependency.MainLLM, "compact_session")
	agentmetrics.Fallbacks.WithLabelValues("context_compaction").Inc()
	if c != nil && c.Fallback != nil {
		return c.Fallback.Compact(ctx, previous, events)
	}
	return Summary{}, cause
}

func parseSummary(raw string) (Summary, error) {
	raw = strings.TrimSpace(raw)
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return Summary{}, errors.New("summary output does not contain JSON object")
	}
	var result Summary
	if err := json.Unmarshal([]byte(raw[start:end+1]), &result); err != nil {
		return Summary{}, err
	}
	if strings.TrimSpace(result.Summary) == "" {
		return Summary{}, errors.New("summary is empty")
	}
	return result, nil
}
