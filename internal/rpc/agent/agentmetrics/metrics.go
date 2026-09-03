package agentmetrics

import "github.com/prometheus/client_golang/prometheus"

var (
	Requests               = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "agent_requests_total", Help: "Agent requests by terminal status."}, []string{"status"})
	Success                = prometheus.NewCounter(prometheus.CounterOpts{Name: "agent_success_total", Help: "Successful agent requests."})
	Steps                  = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "agent_steps", Help: "Observable agent steps per request.", Buckets: []float64{1, 2, 3, 5, 8, 13, 21, 34}})
	Latency                = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "agent_latency_seconds", Help: "End-to-end agent request latency.", Buckets: prometheus.DefBuckets})
	Tokens                 = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "agent_tokens", Help: "Estimated or reported tokens by kind.", Buckets: prometheus.ExponentialBuckets(128, 2, 10)}, []string{"kind"})
	ToolCalls              = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "agent_tool_calls_total", Help: "Tool executions by tool and status."}, []string{"tool", "status"})
	DuplicateToolCalls     = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "agent_duplicate_tool_calls_total", Help: "Duplicate tool calls within one run."}, []string{"tool"})
	Fallbacks              = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "agent_fallback_total", Help: "Agent fallbacks by stage."}, []string{"stage"})
	Repairs                = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "agent_repair_total", Help: "Structured output repair attempts."}, []string{"stage", "status"})
	ContextTokens          = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "context_tokens", Help: "Estimated context tokens by source.", Buckets: prometheus.ExponentialBuckets(64, 2, 10)}, []string{"source"})
	Compactions            = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "context_compaction_total", Help: "Context compactions by status."}, []string{"status"})
	MemoryRetrievalLatency = prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "memory_retrieval_latency_seconds", Help: "Memory retrieval latency by type.", Buckets: prometheus.DefBuckets}, []string{"type"})
	SkillRoutes            = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "skill_route_total", Help: "Skill routes by bounded skill and status."}, []string{"skill", "status"})
	TaskPlans              = prometheus.NewCounter(prometheus.CounterOpts{Name: "agent_task_plans_total", Help: "Complex task plans created."})
	TaskStepUpdates        = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "agent_task_step_updates_total", Help: "Task step updates by bounded status."}, []string{"status"})
	PlanUpdateInterval     = prometheus.NewHistogram(prometheus.HistogramOpts{Name: "agent_plan_update_interval_seconds", Help: "Seconds between observable task plan updates.", Buckets: prometheus.ExponentialBuckets(1, 2, 12)})
)

func Collectors() []prometheus.Collector {
	return []prometheus.Collector{Requests, Success, Steps, Latency, Tokens, ToolCalls, DuplicateToolCalls, Fallbacks, Repairs, ContextTokens, Compactions, MemoryRetrievalLatency, SkillRoutes, TaskPlans, TaskStepUpdates, PlanUpdateInterval}
}
