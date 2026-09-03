package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strings"
)

type Case struct {
	ID             string         `json:"id"`
	Question       string         `json:"question"`
	ExpectedSkill  string         `json:"expected_skill"`
	ExpectedTools  []string       `json:"expected_tools"`
	ForbiddenTools []string       `json:"forbidden_tools"`
	GoldDocs       []string       `json:"gold_docs"`
	ExpectedResult ExpectedResult `json:"expected_result"`
	Rubric         string         `json:"rubric"`
	Metadata       map[string]any `json:"metadata"`
}
type ExpectedResult struct {
	Contains []string `json:"contains"`
}

type Prediction struct {
	CaseID        string   `json:"case_id"`
	Skill         string   `json:"skill"`
	Tools         []string `json:"tools"`
	RetrievedDocs []string `json:"retrieved_docs"`
	CitedDocs     []string `json:"cited_docs"`
	Answer        string   `json:"answer"`
	LatencyMs     float64  `json:"latency_ms"`
	Tokens        int      `json:"tokens"`
	CostUSD       float64  `json:"cost_usd"`
	LLMCalls      int      `json:"llm_calls"`
	Steps         int      `json:"steps"`
	Retries       int      `json:"retries"`
	Fallbacks     int      `json:"fallbacks"`
	ToolErrors    int      `json:"tool_errors"`
}

type Report struct {
	Variant                 string   `json:"variant"`
	Cases                   int      `json:"cases"`
	TaskSuccess             float64  `json:"task_success"`
	AnswerCorrectness       float64  `json:"answer_correctness"`
	SkillRoutingAccuracy    float64  `json:"skill_routing_accuracy"`
	ToolAccuracy            float64  `json:"tool_accuracy"`
	ConstraintViolationRate float64  `json:"constraint_violation_rate"`
	HitAt1                  float64  `json:"hit_at_1"`
	RecallAt3               float64  `json:"recall_at_3"`
	RecallAt5               float64  `json:"recall_at_5"`
	MRR                     float64  `json:"mrr"`
	Faithfulness            float64  `json:"faithfulness"`
	AvgLatencyMs            float64  `json:"avg_latency_ms"`
	AvgTokens               float64  `json:"avg_tokens"`
	AvgCostUSD              float64  `json:"avg_cost_usd"`
	AvgLLMCalls             float64  `json:"avg_llm_calls"`
	AvgToolCalls            float64  `json:"avg_tool_calls"`
	AvgSteps                float64  `json:"avg_steps"`
	DuplicateToolCalls      int      `json:"duplicate_tool_calls"`
	Retries                 int      `json:"retries"`
	Fallbacks               int      `json:"fallbacks"`
	FallbackRate            float64  `json:"fallback_rate"`
	ToolErrors              int      `json:"tool_errors"`
	ToolErrorRate           float64  `json:"tool_error_rate"`
	RetryRate               float64  `json:"retry_rate"`
	MissingPredictions      []string `json:"missing_predictions,omitempty"`
}

func ReadJSONL[T any](reader io.Reader) ([]T, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	var values []T
	line := 0
	for scanner.Scan() {
		line++
		text := strings.TrimSpace(scanner.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		var value T
		if err := json.Unmarshal([]byte(text), &value); err != nil {
			return nil, fmt.Errorf("JSONL line %d: %w", line, err)
		}
		values = append(values, value)
	}
	return values, scanner.Err()
}

func Evaluate(variant string, cases []Case, predictions []Prediction) Report {
	report := Report{Variant: variant, Cases: len(cases)}
	byID := make(map[string]Prediction, len(predictions))
	for _, p := range predictions {
		byID[p.CaseID] = p
	}
	var task, correctness, skill, tools, violations, hit1, recall3, recall5, mrr, faith, ragCases float64
	for _, c := range cases {
		p, ok := byID[c.ID]
		if !ok {
			report.MissingPredictions = append(report.MissingPredictions, c.ID)
			continue
		}
		skillOK := c.ExpectedSkill == "" || p.Skill == c.ExpectedSkill
		if skillOK {
			skill++
		}
		toolOK := containsAll(p.Tools, c.ExpectedTools)
		if toolOK {
			tools++
		}
		violation := intersects(p.Tools, c.ForbiddenTools)
		if violation {
			violations++
		}
		answerScore := containsScore(strings.ToLower(p.Answer), c.ExpectedResult.Contains)
		correctness += answerScore
		if skillOK && toolOK && !violation && answerScore == 1 {
			task++
		}
		if len(c.GoldDocs) > 0 {
			ragCases++
			if len(p.RetrievedDocs) > 0 && contains(c.GoldDocs, p.RetrievedDocs[0]) {
				hit1++
			}
			recall3 += recallAt(p.RetrievedDocs, c.GoldDocs, 3)
			recall5 += recallAt(p.RetrievedDocs, c.GoldDocs, 5)
			mrr += reciprocalRank(p.RetrievedDocs, c.GoldDocs)
			faith += groundedCitationScore(p.CitedDocs, p.RetrievedDocs, c.GoldDocs)
		}
		report.AvgLatencyMs += p.LatencyMs
		report.AvgTokens += float64(p.Tokens)
		report.AvgCostUSD += p.CostUSD
		report.AvgLLMCalls += float64(p.LLMCalls)
		report.AvgToolCalls += float64(len(p.Tools))
		report.AvgSteps += float64(p.Steps)
		report.DuplicateToolCalls += duplicates(p.Tools)
		report.Retries += p.Retries
		report.Fallbacks += p.Fallbacks
		report.ToolErrors += p.ToolErrors
	}
	n := float64(len(cases))
	if n > 0 {
		report.TaskSuccess = task / n
		report.AnswerCorrectness = correctness / n
		report.SkillRoutingAccuracy = skill / n
		report.ToolAccuracy = tools / n
		report.ConstraintViolationRate = violations / n
		report.AvgLatencyMs /= n
		report.AvgTokens /= n
		report.AvgCostUSD /= n
		report.AvgLLMCalls /= n
		report.AvgToolCalls /= n
		report.AvgSteps /= n
		report.FallbackRate = float64(report.Fallbacks) / n
		report.ToolErrorRate = float64(report.ToolErrors) / n
		report.RetryRate = float64(report.Retries) / n
	}
	if ragCases > 0 {
		report.HitAt1 = hit1 / ragCases
		report.RecallAt3 = recall3 / ragCases
		report.RecallAt5 = recall5 / ragCases
		report.MRR = mrr / ragCases
		report.Faithfulness = faith / ragCases
	}
	return report
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
func containsAll(values, required []string) bool {
	for _, r := range required {
		if !contains(values, r) {
			return false
		}
	}
	return true
}
func intersects(a, b []string) bool {
	for _, v := range a {
		if contains(b, v) {
			return true
		}
	}
	return false
}
func containsScore(answer string, required []string) float64 {
	if len(required) == 0 {
		return 1
	}
	hit := 0
	for _, term := range required {
		if strings.Contains(answer, strings.ToLower(term)) {
			hit++
		}
	}
	return float64(hit) / float64(len(required))
}
func recallAt(retrieved, gold []string, k int) float64 {
	if len(gold) == 0 {
		return 0
	}
	if len(retrieved) > k {
		retrieved = retrieved[:k]
	}
	hit := 0
	for _, g := range gold {
		if contains(retrieved, g) {
			hit++
		}
	}
	return float64(hit) / float64(len(gold))
}
func reciprocalRank(retrieved, gold []string) float64 {
	for i, d := range retrieved {
		if contains(gold, d) {
			return 1 / float64(i+1)
		}
	}
	return 0
}
func groundedCitationScore(cited, retrieved, gold []string) float64 {
	if len(cited) == 0 {
		return 0
	}
	hit := 0
	for _, d := range cited {
		if contains(retrieved, d) && contains(gold, d) {
			hit++
		}
	}
	return float64(hit) / float64(len(cited))
}
func duplicates(values []string) int {
	seen := map[string]bool{}
	n := 0
	for _, v := range values {
		if seen[v] {
			n++
		}
		seen[v] = true
	}
	return n
}
func Round(value float64) float64 { return math.Round(value*10000) / 10000 }
