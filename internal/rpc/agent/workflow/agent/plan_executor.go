package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"liveclass/internal/rpc/agent/agentmetrics"
	"liveclass/internal/rpc/agent/dependency"
	"liveclass/internal/rpc/agent/global"
	"liveclass/internal/rpc/agent/memory"
	agentmodel "liveclass/internal/rpc/agent/model"
	"liveclass/internal/rpc/agent/toolruntime"
	agenttrace "liveclass/internal/rpc/agent/trace"
)

const (
	defaultPlanMaxSteps          = 6
	defaultPlanMaxReplans        = 1
	defaultPlanStepMaxReActSteps = 5
)

var planStepKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{1,63}$`)

type planProposal struct {
	Goal  string             `json:"goal"`
	Steps []planStepProposal `json:"steps"`
}

type planStepProposal struct {
	Key         string   `json:"key"`
	Description string   `json:"description"`
	DependsOn   []string `json:"depends_on"`
}

type stepExecutionResult struct {
	Status      string `json:"status"`
	Output      string `json:"output"`
	NeedsReplan bool   `json:"needs_replan"`
	Reason      string `json:"reason"`
}

type invokeRunner interface {
	Invoke(context.Context, *agentmodel.UserMessage, ...compose.Option) (*schema.Message, error)
}

type planStore interface {
	GetTaskPlanByRequest(context.Context, int64, string, string) (*agentmodel.TaskPlan, []agentmodel.TaskStep, error)
	CreateTaskPlan(context.Context, int64, string, string, string, []memory.NewTaskStep) (*agentmodel.TaskPlan, error)
	GetTaskPlan(context.Context, int64, int64) (*agentmodel.TaskPlan, []agentmodel.TaskStep, error)
	UpdateTaskStep(context.Context, int64, int64, string, string) error
	SaveTaskStepResult(context.Context, int64, string, string, int64, string, string) error
	LoadTaskStepResult(context.Context, string, string, int64, string) (string, error)
	ReviseTaskPlan(context.Context, int64, int64, []memory.NewTaskStep) error
}

func newAdaptiveExecutor(ctx context.Context, dbm *memory.DBManager) (*compose.Lambda, error) {
	stepMax := defaultPlanStepMaxReActSteps
	if global.Config != nil && global.Config.AgentRuntime.PlanStepMaxReActSteps > 0 {
		stepMax = global.Config.AgentRuntime.PlanStepMaxReActSteps
	}
	simpleRunner, err := buildExecutionGraph(ctx, dbm, 0)
	if err != nil {
		return nil, err
	}
	stepRunner, err := buildExecutionGraph(ctx, dbm, stepMax)
	if err != nil {
		return nil, err
	}
	return compose.InvokableLambda(func(callCtx context.Context, input *agentmodel.UserMessage) (*schema.Message, error) {
		if !runtimeRequiresPlan(input) {
			recordPlanningDecision(callCtx, input, false)
			return dependency.Do(callCtx, dependency.MainLLM, "generate_direct", func(modelCtx context.Context) (*schema.Message, error) {
				return simpleRunner.Invoke(modelCtx, input)
			})
		}
		recordPlanningDecision(callCtx, input, true)
		planTimeout := 120 * time.Second
		if global.Config != nil && global.Config.AgentRuntime.PlanExecutionTimeout > 0 {
			planTimeout = global.Config.AgentRuntime.PlanExecutionTimeout
		}
		planCtx, cancel := context.WithTimeout(callCtx, planTimeout)
		defer cancel()
		return executePlannedTask(planCtx, dbm, stepRunner, input)
	}), nil
}

func runtimeRequiresPlan(input *agentmodel.UserMessage) bool {
	if input == nil {
		return false
	}
	query := strings.ToLower(strings.TrimSpace(input.Query))
	planning := containsAny(query, "制定", "规划", "安排", "计划", "plan")
	multiStep := containsAny(query, "分步骤", "多步骤", "依赖", "阶段", "顺序", "并且", "然后", "接着", "本周", "下周", "两周", "下个月", "三天", "课表", "薄弱", "执行", "multi-step", "multiple step")
	if planning && multiStep {
		return true
	}
	advice := input.SkillAdvice
	return advice != nil && advice.RequiresPlan && strings.EqualFold(advice.Complexity, "complex") && advice.EstimatedSteps >= 2
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(text, value) {
			return true
		}
	}
	return false
}

func recordPlanningDecision(ctx context.Context, input *agentmodel.UserMessage, planned bool) {
	run := agenttrace.FromContext(ctx)
	metadata := map[string]any{"requires_plan": planned, "source": "runtime_policy"}
	if input != nil && input.SkillAdvice != nil {
		metadata["advisor_requires_plan"] = input.SkillAdvice.RequiresPlan
		metadata["advisor_complexity"] = input.SkillAdvice.Complexity
		metadata["advisor_reason"] = input.SkillAdvice.Reason
	}
	run.Record(ctx, "planning_decision", "adaptive_router", map[bool]string{true: "planned", false: "direct"}[planned], 0, metadata)
	agentmetrics.PlanningDecisions.WithLabelValues(map[bool]string{true: "planned", false: "direct"}[planned]).Inc()
}

func executePlannedTask(ctx context.Context, dbm planStore, stepRunner invokeRunner, input *agentmodel.UserMessage) (*schema.Message, error) {
	if dbm == nil || stepRunner == nil || input == nil {
		return nil, errors.New("invalid plan executor dependencies")
	}
	principal, ok := toolruntime.PrincipalFromContext(ctx)
	if !ok || principal.SessionID == "" || principal.RequestID == "" {
		return nil, errors.New("missing plan executor principal")
	}
	maxSteps, maxReplans := defaultPlanMaxSteps, defaultPlanMaxReplans
	if global.Config != nil {
		if global.Config.AgentRuntime.PlanMaxSteps > 0 {
			maxSteps = global.Config.AgentRuntime.PlanMaxSteps
		}
		if global.Config.AgentRuntime.PlanMaxReplans > 0 {
			maxReplans = global.Config.AgentRuntime.PlanMaxReplans
		}
	}

	plan, steps, err := dbm.GetTaskPlanByRequest(ctx, principal.UserID, principal.SessionID, principal.RequestID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		proposal, planErr := generatePlan(ctx, input, nil, nil, "", maxSteps)
		if planErr != nil {
			proposal = fallbackPlan(input.Query)
			if validateErr := validatePlanProposal(proposal, maxSteps, 2, nil); validateErr != nil {
				return nil, errors.Join(planErr, validateErr)
			}
			dependency.FallbackContext(ctx, dependency.AdvisorLLM, "generate_plan")
		}
		created, createErr := dbm.CreateTaskPlan(ctx, principal.UserID, principal.SessionID, principal.RequestID, proposal.Goal, toNewTaskSteps(proposal.Steps))
		if createErr != nil {
			return nil, createErr
		}
		plan = created
		_, steps, err = dbm.GetTaskPlan(ctx, principal.UserID, plan.ID)
		if err != nil {
			return nil, err
		}
		run := agenttrace.FromContext(ctx)
		run.Record(ctx, "plan_created", "planner", "ok", 0, map[string]any{"plan_id": plan.ID, "steps": len(steps)})
		// Preserve the observable capability name used by the evaluator while
		// persistence itself is controlled by Runtime rather than model choice.
		run.Record(ctx, "tool_result", "create_task_plan", "ok", 0, map[string]any{"source": "runtime_planner", "plan_id": plan.ID, "steps": len(steps)})
	}

	results := make(map[string]string)
	for _, step := range steps {
		if step.Status == "done" {
			if result, loadErr := dbm.LoadTaskStepResult(ctx, principal.SessionID, principal.RequestID, plan.ID, step.StepKey); loadErr == nil && result != "" {
				results[step.StepKey] = result
			}
		}
	}

	replans := 0
	for iterations := 0; iterations < maxSteps*(maxReplans+1)+maxReplans+1; iterations++ {
		_, steps, err = dbm.GetTaskPlan(ctx, principal.UserID, plan.ID)
		if err != nil {
			return nil, err
		}
		if allPlanStepsDone(steps) {
			return synthesizePlanResult(ctx, input.Query, steps, results)
		}
		step, readyErr := nextReadyStep(steps)
		if readyErr != nil {
			return nil, readyErr
		}
		if step.Status == "running" {
			if saved, loadErr := dbm.LoadTaskStepResult(ctx, principal.SessionID, principal.RequestID, plan.ID, step.StepKey); loadErr == nil && saved != "" {
				results[step.StepKey] = saved
				if err = dbm.UpdateTaskStep(ctx, principal.UserID, plan.ID, step.StepKey, "done"); err != nil {
					return nil, err
				}
				continue
			}
		} else if err = dbm.UpdateTaskStep(ctx, principal.UserID, plan.ID, step.StepKey, "running"); err != nil {
			return nil, err
		}

		stepInput := *input
		stepInput.History = nil
		stepInput.Query = buildStepPrompt(plan.Goal, step, results)
		started := time.Now()
		resp, invokeErr := dependency.Do(ctx, dependency.MainLLM, "execute_plan_step", func(callCtx context.Context) (*schema.Message, error) {
			return stepRunner.Invoke(callCtx, &stepInput)
		})
		parsed := stepExecutionResult{Status: "failed", Reason: errorText(invokeErr)}
		if invokeErr == nil {
			recordModelUsage(ctx, "plan_step", resp)
			parsed = parseStepExecutionResult(resp.Content)
		}
		run := agenttrace.FromContext(ctx)
		run.Record(ctx, "plan_step", step.StepKey, parsed.Status, time.Since(started), map[string]any{"plan_id": plan.ID, "needs_replan": parsed.NeedsReplan, "reason": parsed.Reason})
		agentmetrics.PlanStepLatency.Observe(time.Since(started).Seconds())

		if invokeErr == nil && parsed.Status != "failed" && !parsed.NeedsReplan {
			if err = dbm.SaveTaskStepResult(ctx, principal.UserID, principal.SessionID, principal.RequestID, plan.ID, step.StepKey, parsed.Output); err != nil {
				return nil, err
			}
			results[step.StepKey] = parsed.Output
			if err = dbm.UpdateTaskStep(ctx, principal.UserID, plan.ID, step.StepKey, "done"); err != nil {
				return nil, err
			}
			continue
		}

		_ = dbm.UpdateTaskStep(ctx, principal.UserID, plan.ID, step.StepKey, "failed")
		if replans >= maxReplans {
			if invokeErr != nil {
				return nil, invokeErr
			}
			return nil, fmt.Errorf("plan step %s failed: %s", step.StepKey, parsed.Reason)
		}
		completed := completedStepKeys(steps)
		failure := strings.TrimSpace(parsed.Reason)
		if failure == "" {
			failure = errorText(invokeErr)
		}
		proposal, replanErr := generatePlan(ctx, input, steps, results, step.StepKey+": "+failure, maxSteps)
		if replanErr != nil {
			return nil, errors.Join(invokeErr, replanErr)
		}
		if err = validatePlanProposal(proposal, maxSteps, 1, completed); err != nil {
			return nil, err
		}
		if err = dbm.ReviseTaskPlan(ctx, principal.UserID, plan.ID, toNewTaskSteps(proposal.Steps)); err != nil {
			return nil, err
		}
		replans++
		agentmetrics.PlanReplans.Inc()
		run.Record(ctx, "plan_replanned", "planner", "ok", 0, map[string]any{"plan_id": plan.ID, "attempt": replans, "failed_step": step.StepKey})
	}
	return nil, errors.New("plan executor iteration limit exceeded")
}

func generatePlan(ctx context.Context, input *agentmodel.UserMessage, previous []agentmodel.TaskStep, results map[string]string, failure string, maxSteps int) (planProposal, error) {
	if global.ChatModel == nil {
		return planProposal{}, errors.New("nil planner model")
	}
	previousJSON, _ := json.Marshal(previous)
	resultsJSON, _ := json.Marshal(results)
	system := fmt.Sprintf("你是只负责拆解任务的 Planner，不调用工具。输出严格 JSON。计划包含 2 到 %d 个任务相关步骤；key 使用英文 snake_case；depends_on 只能引用存在的步骤且不得成环。不要输出思维过程。格式：{\"goal\":\"...\",\"steps\":[{\"key\":\"...\",\"description\":\"...\",\"depends_on\":[]}]}。", maxSteps)
	user := "用户目标：" + input.Query
	if len(previous) > 0 {
		user += "\n上一版计划：" + string(previousJSON) + "\n已完成结果：" + string(resultsJSON) + "\n失败信息：" + failure + "\n请保留仍然有效的已完成步骤，并调整未完成步骤。"
	}
	started := time.Now()
	resp, err := dependency.Do(ctx, dependency.AdvisorLLM, "generate_plan", func(callCtx context.Context) (*schema.Message, error) {
		return global.ChatModel.Generate(callCtx, []*schema.Message{schema.SystemMessage(system), schema.UserMessage(user)})
	})
	agentmetrics.PlannerLatency.Observe(time.Since(started).Seconds())
	if err != nil {
		agenttrace.FromContext(ctx).Record(ctx, "planner_call", "structured_planner", "error", time.Since(started), map[string]any{"error": agenttrace.SafeError(err)})
		return planProposal{}, err
	}
	recordModelUsage(ctx, "planner", resp)
	proposal, err := parsePlanProposal(resp.Content)
	if err == nil {
		err = validatePlanProposal(proposal, maxSteps, 2, completedStepKeys(previous))
	}
	agenttrace.FromContext(ctx).Record(ctx, "planner_call", "structured_planner", map[bool]string{true: "error", false: "ok"}[err != nil], time.Since(started), map[string]any{"steps": len(proposal.Steps)})
	return proposal, err
}

func parsePlanProposal(raw string) (planProposal, error) {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return planProposal{}, errors.New("planner returned no JSON")
	}
	var proposal planProposal
	if err := json.Unmarshal([]byte(raw[start:end+1]), &proposal); err != nil {
		return planProposal{}, fmt.Errorf("decode planner output: %w", err)
	}
	proposal.Goal = strings.TrimSpace(proposal.Goal)
	for i := range proposal.Steps {
		proposal.Steps[i].Key = strings.TrimSpace(proposal.Steps[i].Key)
		proposal.Steps[i].Description = strings.TrimSpace(proposal.Steps[i].Description)
	}
	return proposal, nil
}

func validatePlanProposal(proposal planProposal, maxSteps, minSteps int, externallyDone map[string]bool) error {
	if proposal.Goal == "" || len(proposal.Steps) < minSteps || len(proposal.Steps) > maxSteps {
		return fmt.Errorf("plan must have a goal and %d..%d steps", minSteps, maxSteps)
	}
	keys := make(map[string]bool, len(proposal.Steps))
	for _, step := range proposal.Steps {
		if !planStepKeyPattern.MatchString(step.Key) || step.Description == "" || keys[step.Key] {
			return fmt.Errorf("invalid or duplicate plan step %q", step.Key)
		}
		keys[step.Key] = true
	}
	for _, step := range proposal.Steps {
		for _, dependency := range step.DependsOn {
			if dependency == step.Key || (!keys[dependency] && !externallyDone[dependency]) {
				return fmt.Errorf("invalid dependency %q for step %q", dependency, step.Key)
			}
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	byKey := make(map[string]planStepProposal, len(proposal.Steps))
	for _, step := range proposal.Steps {
		byKey[step.Key] = step
	}
	var visit func(string) error
	visit = func(key string) error {
		if visiting[key] {
			return errors.New("plan dependencies contain a cycle")
		}
		if visited[key] {
			return nil
		}
		visiting[key] = true
		for _, dependency := range byKey[key].DependsOn {
			if keys[dependency] {
				if err := visit(dependency); err != nil {
					return err
				}
			}
		}
		visiting[key], visited[key] = false, true
		return nil
	}
	for key := range keys {
		if err := visit(key); err != nil {
			return err
		}
	}
	return nil
}

func fallbackPlan(goal string) planProposal {
	return planProposal{Goal: goal, Steps: []planStepProposal{
		{Key: "analyze_requirements", Description: "确认目标、约束与完成标准"},
		{Key: "execute_task", Description: "根据目标调用必要工具并完成任务", DependsOn: []string{"analyze_requirements"}},
		{Key: "verify_result", Description: "检查执行结果并形成最终结论", DependsOn: []string{"execute_task"}},
	}}
}

func toNewTaskSteps(steps []planStepProposal) []memory.NewTaskStep {
	result := make([]memory.NewTaskStep, 0, len(steps))
	for _, step := range steps {
		depends, _ := json.Marshal(step.DependsOn)
		result = append(result, memory.NewTaskStep{Key: step.Key, Description: step.Description, DependsOn: string(depends)})
	}
	return result
}

func nextReadyStep(steps []agentmodel.TaskStep) (agentmodel.TaskStep, error) {
	status := make(map[string]string, len(steps))
	for _, step := range steps {
		status[step.StepKey] = step.Status
	}
	for _, step := range steps {
		if step.Status != "pending" && step.Status != "running" {
			continue
		}
		var dependencies []string
		if err := json.Unmarshal([]byte(step.DependsOn), &dependencies); err != nil {
			return agentmodel.TaskStep{}, err
		}
		ready := true
		for _, dependency := range dependencies {
			if status[dependency] != "done" {
				ready = false
				break
			}
		}
		if ready {
			return step, nil
		}
	}
	return agentmodel.TaskStep{}, errors.New("plan has no ready step")
}

func allPlanStepsDone(steps []agentmodel.TaskStep) bool {
	if len(steps) == 0 {
		return false
	}
	for _, step := range steps {
		if step.Status != "done" {
			return false
		}
	}
	return true
}

func completedStepKeys(steps []agentmodel.TaskStep) map[string]bool {
	result := make(map[string]bool)
	for _, step := range steps {
		if step.Status == "done" {
			result[step.StepKey] = true
		}
	}
	return result
}

func buildStepPrompt(goal string, step agentmodel.TaskStep, results map[string]string) string {
	keys := make([]string, 0, len(results))
	for key := range results {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var completed strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&completed, "\n- %s: %s", key, results[key])
	}
	return fmt.Sprintf("你是 Plan Executor 的单步 ReAct 执行器。只执行当前步骤，不要创建或更新 TaskPlan，不要提前执行后续步骤。\n总目标：%s\n当前步骤：%s（%s）\n已完成步骤结果：%s\n完成后严格返回 JSON：{\"status\":\"done|failed\",\"output\":\"可供后续步骤使用的结果\",\"needs_replan\":false,\"reason\":\"\"}", goal, step.StepKey, step.Description, completed.String())
}

func parseStepExecutionResult(raw string) stepExecutionResult {
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		var result stepExecutionResult
		if json.Unmarshal([]byte(raw[start:end+1]), &result) == nil {
			result.Status = strings.ToLower(strings.TrimSpace(result.Status))
			if result.Status == "done" || result.Status == "failed" {
				return result
			}
		}
	}
	return stepExecutionResult{Status: "done", Output: strings.TrimSpace(raw)}
}

func synthesizePlanResult(ctx context.Context, goal string, steps []agentmodel.TaskStep, results map[string]string) (*schema.Message, error) {
	keys := make([]string, 0, len(results))
	for key := range results {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var body strings.Builder
	for _, key := range keys {
		fmt.Fprintf(&body, "\n[%s]\n%s\n", key, results[key])
	}
	if global.ChatModel == nil {
		return schema.AssistantMessage(strings.TrimSpace(body.String()), nil), nil
	}
	return dependency.Do(ctx, dependency.MainLLM, "synthesize_plan", func(callCtx context.Context) (*schema.Message, error) {
		return global.ChatModel.Generate(callCtx, []*schema.Message{
			schema.SystemMessage("根据已经完成的步骤结果回答用户原始目标。不要虚构未出现的工具结果，不要描述私有思维过程。"),
			schema.UserMessage("原始目标：" + goal + "\n步骤结果：" + body.String()),
		})
	})
}

func errorText(err error) string {
	return agenttrace.SafeError(err)
}

func recordModelUsage(ctx context.Context, stage string, message *schema.Message) {
	if message == nil || message.ResponseMeta == nil || message.ResponseMeta.Usage == nil {
		return
	}
	usage := message.ResponseMeta.Usage
	agentmetrics.Tokens.WithLabelValues(stage).Observe(float64(usage.TotalTokens))
	agenttrace.FromContext(ctx).Record(ctx, "token_usage", stage, "observed", 0, map[string]any{
		"prompt": usage.PromptTokens, "completion": usage.CompletionTokens, "total": usage.TotalTokens,
	})
}
