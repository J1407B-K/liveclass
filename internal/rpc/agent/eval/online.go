package eval

import (
	"hash/fnv"
	"time"
)

type OnlinePolicy struct {
	MaxTokens        int
	MaxSteps         int
	MaxRetries       int
	MaxLatency       time.Duration
	RandomSampleRate float64
	AllowedSkills    map[string]bool
}

type Observation struct {
	RequestID, Skill                                              string
	Tokens, Steps, DuplicateTools, Retries, Fallbacks, ToolErrors int
	Latency                                                       time.Duration
}

// ShouldJudge selects anomaly samples plus a deterministic small random sample.
// It is cheap and never invokes a model itself.
func ShouldJudge(policy OnlinePolicy, observation Observation) (bool, []string) {
	var reasons []string
	if policy.MaxTokens > 0 && observation.Tokens > policy.MaxTokens {
		reasons = append(reasons, "token_spike")
	}
	if policy.MaxSteps > 0 && observation.Steps > policy.MaxSteps {
		reasons = append(reasons, "step_spike")
	}
	if observation.DuplicateTools > 0 {
		reasons = append(reasons, "duplicate_tool")
	}
	if policy.MaxRetries >= 0 && observation.Retries > policy.MaxRetries {
		reasons = append(reasons, "retry_spike")
	}
	if observation.Fallbacks > 0 {
		reasons = append(reasons, "fallback")
	}
	if observation.ToolErrors > 0 {
		reasons = append(reasons, "tool_error")
	}
	if policy.MaxLatency > 0 && observation.Latency > policy.MaxLatency {
		reasons = append(reasons, "latency_anomaly")
	}
	if len(policy.AllowedSkills) > 0 && !policy.AllowedSkills[observation.Skill] {
		reasons = append(reasons, "unexpected_skill_route")
	}
	if len(reasons) > 0 {
		return true, reasons
	}
	if policy.RandomSampleRate <= 0 {
		return false, nil
	}
	if policy.RandomSampleRate >= 1 {
		return true, []string{"random_sample"}
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(observation.RequestID))
	if float64(h.Sum32()%10000)/10000 < policy.RandomSampleRate {
		return true, []string{"random_sample"}
	}
	return false, nil
}
