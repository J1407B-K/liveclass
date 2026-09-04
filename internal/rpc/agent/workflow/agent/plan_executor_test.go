package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"liveclass/internal/rpc/agent/config"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	agentmodel "liveclass/internal/rpc/agent/model"
	"liveclass/internal/rpc/agent/toolruntime"
)

func TestRuntimeRequiresPlanUsesRulesAndAdvisor(t *testing.T) {
	tests := []struct {
		name string
		in   *agentmodel.UserMessage
		want bool
	}{
		{name: "simple question", in: &agentmodel.UserMessage{Query: "什么是学习计划"}, want: false},
		{name: "explicit dependent task", in: &agentmodel.UserMessage{Query: "分析薄弱点，然后制定下周复习计划"}, want: true},
		{name: "advisor complex", in: &agentmodel.UserMessage{Query: "帮我完成这项任务", SkillAdvice: &agentmodel.SkillAdvice{RequiresPlan: true, Complexity: "complex", EstimatedSteps: 3}}, want: true},
		{name: "advisor simple", in: &agentmodel.UserMessage{Query: "解释概念", SkillAdvice: &agentmodel.SkillAdvice{RequiresPlan: false, Complexity: "simple", EstimatedSteps: 1}}, want: false},
		{name: "definition of plan", in: &agentmodel.UserMessage{Query: "什么是 Plan and Execute"}, want: false},
		{name: "single rewrite", in: &agentmodel.UserMessage{Query: "帮我润色这段话"}, want: false},
		{name: "single lookup", in: &agentmodel.UserMessage{Query: "查询我的用户信息"}, want: false},
		{name: "simple code question", in: &agentmodel.UserMessage{Query: "解释 Go channel"}, want: false},
		{name: "simple plan wording", in: &agentmodel.UserMessage{Query: "学习计划是什么意思"}, want: false},
		{name: "weekly plan", in: &agentmodel.UserMessage{Query: "制定本周数据库复习计划"}, want: true},
		{name: "ordered plan", in: &agentmodel.UserMessage{Query: "规划学习任务并安排执行顺序"}, want: true},
		{name: "staged plan", in: &agentmodel.UserMessage{Query: "为面试制定分阶段准备方案"}, want: true},
		{name: "dependency plan", in: &agentmodel.UserMessage{Query: "制定有依赖关系的课程任务"}, want: true},
		{name: "english multi step", in: &agentmodel.UserMessage{Query: "plan this multiple step study project"}, want: true},
		{name: "advisor estimate too small", in: &agentmodel.UserMessage{Query: "完成任务", SkillAdvice: &agentmodel.SkillAdvice{RequiresPlan: true, Complexity: "complex", EstimatedSteps: 1}}, want: false},
		{name: "advisor wrong complexity", in: &agentmodel.UserMessage{Query: "完成任务", SkillAdvice: &agentmodel.SkillAdvice{RequiresPlan: true, Complexity: "simple", EstimatedSteps: 4}}, want: false},
		{name: "advisor two steps", in: &agentmodel.UserMessage{Query: "完成任务", SkillAdvice: &agentmodel.SkillAdvice{RequiresPlan: true, Complexity: "complex", EstimatedSteps: 2}}, want: true},
		{name: "runtime overrides advisor", in: &agentmodel.UserMessage{Query: "制定下个月的复习计划", SkillAdvice: &agentmodel.SkillAdvice{RequiresPlan: false, Complexity: "simple", EstimatedSteps: 1}}, want: true},
		{name: "nil input", in: nil, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := runtimeRequiresPlan(tt.in); got != tt.want {
				t.Fatalf("runtimeRequiresPlan()=%v want %v", got, tt.want)
			}
		})
	}
}

func TestValidatePlanProposal(t *testing.T) {
	valid := planProposal{Goal: "prepare", Steps: []planStepProposal{
		{Key: "collect_context", Description: "collect"},
		{Key: "build_answer", Description: "build", DependsOn: []string{"collect_context"}},
	}}
	if err := validatePlanProposal(valid, 6, 2, nil); err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
	cyclic := planProposal{Goal: "bad", Steps: []planStepProposal{
		{Key: "first_step", Description: "first", DependsOn: []string{"second_step"}},
		{Key: "second_step", Description: "second", DependsOn: []string{"first_step"}},
	}}
	if err := validatePlanProposal(cyclic, 6, 2, nil); err == nil {
		t.Fatal("cyclic plan accepted")
	}
}

func TestNextReadyStepRespectsDependencies(t *testing.T) {
	steps := []agentmodel.TaskStep{
		{StepKey: "collect_context", Status: "done", DependsOn: "[]"},
		{StepKey: "build_answer", Status: "pending", DependsOn: `["collect_context"]`},
	}
	got, err := nextReadyStep(steps)
	if err != nil || got.StepKey != "build_answer" {
		t.Fatalf("nextReadyStep()=%#v err=%v", got, err)
	}
}

func TestParseStepExecutionResult(t *testing.T) {
	got := parseStepExecutionResult(`{"status":"done","output":"evidence","needs_replan":false}`)
	if got.Status != "done" || got.Output != "evidence" || got.NeedsReplan {
		t.Fatalf("unexpected step result: %#v", got)
	}
	fallback := parseStepExecutionResult("plain answer")
	if fallback.Status != "done" || fallback.Output != "plain answer" {
		t.Fatalf("unexpected fallback: %#v", fallback)
	}
}

type fakePlanStore struct {
	plan    *agentmodel.TaskPlan
	steps   []agentmodel.TaskStep
	results map[string]string
}

func (f *fakePlanStore) GetTaskPlanByRequest(context.Context, int64, string, string) (*agentmodel.TaskPlan, []agentmodel.TaskStep, error) {
	return f.plan, append([]agentmodel.TaskStep(nil), f.steps...), nil
}
func (f *fakePlanStore) CreateTaskPlan(_ context.Context, userID int64, sessionID, requestID, goal string, inputs []memory.NewTaskStep) (*agentmodel.TaskPlan, error) {
	f.plan = &agentmodel.TaskPlan{ID: 1, UserID: userID, SessionID: sessionID, RequestID: requestID, Goal: goal, Status: "pending"}
	f.results = make(map[string]string)
	for i, input := range inputs {
		f.steps = append(f.steps, agentmodel.TaskStep{ID: int64(i + 1), PlanID: 1, StepKey: input.Key, Description: input.Description, Status: "pending", DependsOn: input.DependsOn})
	}
	return f.plan, nil
}
func (f *fakePlanStore) GetTaskPlan(context.Context, int64, int64) (*agentmodel.TaskPlan, []agentmodel.TaskStep, error) {
	return f.plan, append([]agentmodel.TaskStep(nil), f.steps...), nil
}
func (f *fakePlanStore) UpdateTaskStep(_ context.Context, _ int64, _ int64, key, status string) error {
	for i := range f.steps {
		if f.steps[i].StepKey == key {
			f.steps[i].Status = status
			allDone := true
			for _, step := range f.steps {
				allDone = allDone && step.Status == "done"
			}
			if allDone {
				f.plan.Status = "done"
			} else {
				f.plan.Status = "running"
			}
			return nil
		}
	}
	return fmt.Errorf("step not found: %s", key)
}
func (f *fakePlanStore) SaveTaskStepResult(_ context.Context, _ int64, _, _ string, _ int64, key, result string) error {
	f.results[key] = result
	return nil
}
func (f *fakePlanStore) LoadTaskStepResult(_ context.Context, _, _ string, _ int64, key string) (string, error) {
	return f.results[key], nil
}
func (f *fakePlanStore) ReviseTaskPlan(context.Context, int64, int64, []memory.NewTaskStep) error {
	return nil
}

type fakeStepRunner struct{ calls int }

func (f *fakeStepRunner) Invoke(_ context.Context, _ *agentmodel.UserMessage, _ ...compose.Option) (*schema.Message, error) {
	f.calls++
	payload, _ := json.Marshal(stepExecutionResult{Status: "done", Output: fmt.Sprintf("result-%d", f.calls)})
	return schema.AssistantMessage(string(payload), nil), nil
}

func TestPlanExecutorReturnsToRuntimeAfterEveryStep(t *testing.T) {
	policy := config.DependencyPolicyConfig{Timeout: time.Second, Attempts: 1}
	if err := dependency.Configure(config.ResilienceConfig{
		MainLLM: policy, AdvisorLLM: policy, ProfileLLM: policy, Embedding: policy,
		Qdrant: policy, Elasticsearch: policy, Reranker: policy, WebSearch: policy,
		PostgresRead: policy, PostgresWrite: policy, InternalRPC: policy,
	}); err != nil {
		t.Fatal(err)
	}
	previousConfig, previousModel := global.Config, global.ChatModel
	global.Config = &config.Config{AgentRuntime: config.AgentRuntimeConfig{PlanMaxSteps: 6, PlanMaxReplans: 1, PlanStepMaxReActSteps: 5}}
	global.ChatModel = nil // Force the deterministic three-step fallback plan.
	defer func() { global.Config, global.ChatModel = previousConfig, previousModel }()

	store := &fakePlanStore{}
	runner := &fakeStepRunner{}
	ctx := toolruntime.WithPrincipal(context.Background(), toolruntime.Principal{UserID: 7, SessionID: "session", RequestID: "request", AllowPlanning: true})
	response, err := executePlannedTask(ctx, store, runner, &agentmodel.UserMessage{ID: 7, Query: "制定并执行多步骤计划"})
	if err != nil {
		t.Fatal(err)
	}
	if runner.calls != 3 || store.plan.Status != "done" || len(store.results) != 3 {
		t.Fatalf("calls=%d plan=%s results=%d", runner.calls, store.plan.Status, len(store.results))
	}
	if response == nil || !strings.Contains(response.Content, "result-1") || !strings.Contains(response.Content, "result-2") || !strings.Contains(response.Content, "result-3") {
		t.Fatalf("unexpected synthesis: %#v", response)
	}
}

func TestPlanExecutorRecoversPersistedRunningStepWithoutRepeatingIt(t *testing.T) {
	policy := config.DependencyPolicyConfig{Timeout: time.Second, Attempts: 1}
	if err := dependency.Configure(config.ResilienceConfig{
		MainLLM: policy, AdvisorLLM: policy, ProfileLLM: policy, Embedding: policy,
		Qdrant: policy, Elasticsearch: policy, Reranker: policy, WebSearch: policy,
		PostgresRead: policy, PostgresWrite: policy, InternalRPC: policy,
	}); err != nil {
		t.Fatal(err)
	}
	previousConfig, previousModel := global.Config, global.ChatModel
	global.Config = &config.Config{AgentRuntime: config.AgentRuntimeConfig{PlanMaxSteps: 6, PlanMaxReplans: 1, PlanStepMaxReActSteps: 5}}
	global.ChatModel = nil
	defer func() { global.Config, global.ChatModel = previousConfig, previousModel }()

	store := &fakePlanStore{
		plan: &agentmodel.TaskPlan{ID: 1, UserID: 7, SessionID: "session", RequestID: "request", Goal: "goal", Status: "running"},
		steps: []agentmodel.TaskStep{
			{ID: 1, PlanID: 1, StepKey: "analyze", Description: "analyze", Status: "running", DependsOn: "[]"},
			{ID: 2, PlanID: 1, StepKey: "execute", Description: "execute", Status: "pending", DependsOn: `["analyze"]`},
			{ID: 3, PlanID: 1, StepKey: "verify", Description: "verify", Status: "pending", DependsOn: `["execute"]`},
		},
		results: map[string]string{"analyze": "persisted-result"},
	}
	runner := &fakeStepRunner{}
	ctx := toolruntime.WithPrincipal(context.Background(), toolruntime.Principal{UserID: 7, SessionID: "session", RequestID: "request", AllowPlanning: true})
	response, err := executePlannedTask(ctx, store, runner, &agentmodel.UserMessage{ID: 7, Query: "goal"})
	if err != nil {
		t.Fatal(err)
	}
	if runner.calls != 2 {
		t.Fatalf("persisted running step was repeated: calls=%d want=2", runner.calls)
	}
	if response == nil || !strings.Contains(response.Content, "persisted-result") {
		t.Fatalf("persisted result missing from synthesis: %#v", response)
	}
}
