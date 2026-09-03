package session

import (
	"sort"
	"strings"
	"unicode"

	"github.com/cloudwego/eino/schema"

	"liveclass/internal/rpc/agent/model"
)

type Budget struct {
	ModelContext      int
	SystemReserve     int
	OutputReserve     int
	RAG               int
	Memory            int
	Conversation      int
	CompactionTrigger int
	RecentTail        int
	MaxToolResult     int
}

func DefaultBudget() Budget {
	return Budget{
		ModelContext: 32768, SystemReserve: 5000, OutputReserve: 4000,
		RAG: 7000, Memory: 3000, Conversation: 12000,
		CompactionTrigger: 10500, RecentTail: 4000, MaxToolResult: 1200,
	}
}

func (b Budget) Validate() bool {
	return b.ModelContext > 0 && b.OutputReserve > 0 && b.Conversation > 0 &&
		b.SystemReserve+b.OutputReserve+b.RAG+b.Memory+b.Conversation <= b.ModelContext &&
		b.RecentTail > 0 && b.RecentTail < b.Conversation
}

// EstimateTextTokens is deliberately deterministic and dependency-free. CJK
// runes count as one token and runs of ASCII letters/digits count as one token
// per four characters. Production metrics retain actual model usage separately.
func EstimateTextTokens(text string) int {
	var cjk, ascii, other int
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Han, r), unicode.Is(unicode.Hiragana, r), unicode.Is(unicode.Katakana, r), unicode.Is(unicode.Hangul, r):
			cjk++
		case r <= unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r)):
			ascii++
		case !unicode.IsSpace(r):
			other++
		}
	}
	estimate := cjk + (ascii+3)/4 + (other+1)/2
	if estimate == 0 && strings.TrimSpace(text) != "" {
		return 1
	}
	return estimate
}

type BuildInput struct {
	Profile          string
	Facts            string
	Docs             string
	TaskState        string
	CurrentRequestID string
	CurrentRequest   string
	Checkpoint       *model.SummaryCheckpoint
	Events           []model.TranscriptEvent
}

type Context struct {
	Profile         string
	Facts           string
	Docs            string
	History         []*schema.Message
	CurrentRequest  string
	EstimatedTokens int
	Truncated       bool
}

type Builder struct{ Budget Budget }

func NewBuilder(b Budget) *Builder {
	if !b.Validate() {
		b = DefaultBudget()
	}
	return &Builder{Budget: b}
}

func (b *Builder) Build(in BuildInput) Context {
	result := Context{}
	var cut bool
	result.Profile, cut = truncateToTokens(in.Profile, b.Budget.Memory/3)
	result.Truncated = cut
	remainingMemory := b.Budget.Memory - EstimateTextTokens(result.Profile)
	result.Facts, cut = truncateToTokens(in.Facts, remainingMemory)
	result.Truncated = result.Truncated || cut
	result.Docs, cut = truncateToTokens(in.Docs, b.Budget.RAG)
	result.Truncated = result.Truncated || cut

	conversationBudget := b.Budget.Conversation
	result.CurrentRequest, cut = truncateToTokens(in.CurrentRequest, conversationBudget)
	result.Truncated = result.Truncated || cut
	currentTokens := EstimateTextTokens(result.CurrentRequest)
	conversationBudget -= currentTokens
	if conversationBudget < 0 {
		conversationBudget = 0
	}
	if taskState, taskCut := truncateToTokens(in.TaskState, conversationBudget/3); strings.TrimSpace(taskState) != "" {
		result.History = append(result.History, schema.SystemMessage("当前任务计划状态：\n"+taskState))
		conversationBudget -= EstimateTextTokens(taskState)
		result.EstimatedTokens += EstimateTextTokens(taskState)
		result.Truncated = result.Truncated || taskCut
	}
	summaryTokens := 0
	if in.Checkpoint != nil && strings.TrimSpace(in.Checkpoint.Summary) != "" {
		summary, cut := truncateToTokens(in.Checkpoint.Summary, conversationBudget/3)
		if strings.TrimSpace(summary) != "" {
			result.History = append(result.History, schema.SystemMessage("历史会话摘要：\n"+summary))
			summaryTokens = EstimateTextTokens(summary)
			conversationBudget -= summaryTokens
		}
		result.Truncated = result.Truncated || cut
	}

	events := append([]model.TranscriptEvent(nil), in.Events...)
	sort.SliceStable(events, func(i, j int) bool { return events[i].ID < events[j].ID })
	selected := make([]*schema.Message, 0, len(events))
	used := 0
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if in.CurrentRequestID != "" && e.EventType == "user_message" && e.RequestID == in.CurrentRequestID {
			continue
		}
		msg := eventMessage(e, b.Budget.MaxToolResult)
		if msg == nil {
			continue
		}
		cost := EstimateTextTokens(msg.Content)
		if used+cost > conversationBudget {
			result.Truncated = true
			continue
		}
		selected = append(selected, msg)
		used += cost
	}
	for i := len(selected) - 1; i >= 0; i-- {
		result.History = append(result.History, selected[i])
	}
	result.EstimatedTokens += EstimateTextTokens(result.Profile) + EstimateTextTokens(result.Facts) +
		EstimateTextTokens(result.Docs) + currentTokens + summaryTokens + used
	return result
}

func eventMessage(e model.TranscriptEvent, maxToolTokens int) *schema.Message {
	content := e.Content
	switch e.EventType {
	case "user_message":
		return schema.UserMessage(content)
	case "assistant_message":
		return schema.AssistantMessage(content, nil)
	case "tool_result":
		content, _ = truncateToTokens(content, maxToolTokens)
		return schema.SystemMessage("工具结果（" + e.EventKey + "）：\n" + content)
	case "error", "runtime_event":
		return schema.SystemMessage(content)
	default:
		return nil
	}
}

func truncateToTokens(text string, budget int) (string, bool) {
	if budget <= 0 {
		return "", strings.TrimSpace(text) != ""
	}
	if EstimateTextTokens(text) <= budget {
		return text, false
	}
	marker := "[截断]"
	contentBudget := budget - EstimateTextTokens(marker)
	if contentBudget <= 0 {
		return marker, true
	}
	runes := []rune(text)
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if EstimateTextTokens(string(runes[:mid])) <= contentBudget {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return strings.TrimSpace(string(runes[:lo])) + marker, true
}
