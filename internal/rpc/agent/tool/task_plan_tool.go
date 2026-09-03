package tool

import (
	"context"
	"encoding/json"
	"errors"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/toolruntime"
)

type PlanStepInput struct {
	Key         string   `json:"key"`
	Description string   `json:"description"`
	DependsOn   []string `json:"depends_on,omitempty"`
}
type CreateTaskPlanRequest struct {
	Goal  string          `json:"goal"`
	Steps []PlanStepInput `json:"steps"`
}
type CreateTaskPlanResponse struct {
	PlanID int64  `json:"plan_id"`
	Status string `json:"status"`
}

func NewCreateTaskPlanTool(db *memory.DBManager) (einotool.InvokableTool, error) {
	if db == nil {
		return nil, errors.New("nil task plan store")
	}
	return utils.InferTool("create_task_plan", "仅为包含至少两个相互依赖步骤的复杂任务创建执行计划；普通问答禁止调用", func(ctx context.Context, req *CreateTaskPlanRequest) (*CreateTaskPlanResponse, error) {
		principal, ok := toolruntime.PrincipalFromContext(ctx)
		if !ok {
			return nil, errors.New("missing runtime principal")
		}
		steps := make([]memory.NewTaskStep, 0, len(req.Steps))
		for _, step := range req.Steps {
			raw, _ := json.Marshal(step.DependsOn)
			steps = append(steps, memory.NewTaskStep{Key: step.Key, Description: step.Description, DependsOn: string(raw)})
		}
		plan, err := db.CreateTaskPlan(ctx, principal.UserID, principal.SessionID, principal.RequestID, req.Goal, steps)
		if err != nil {
			return nil, err
		}
		return &CreateTaskPlanResponse{PlanID: plan.ID, Status: plan.Status}, nil
	})
}

type UpdateTaskStepRequest struct {
	PlanID  int64  `json:"plan_id"`
	StepKey string `json:"step_key"`
	Status  string `json:"status"`
}
type UpdateTaskStepResponse struct {
	Updated bool `json:"updated"`
}

func NewUpdateTaskStepTool(db *memory.DBManager) (einotool.InvokableTool, error) {
	return newUpdateTaskStepTool(db, "update_task_step")
}

// NewUpdateTaskPlanCompatTool keeps old model/tool traces executable while the
// canonical name remains update_task_step. Both names enforce the same runtime
// permission and state-transition checks.
func NewUpdateTaskPlanCompatTool(db *memory.DBManager) (einotool.InvokableTool, error) {
	return newUpdateTaskStepTool(db, "update_task_plan")
}

func newUpdateTaskStepTool(db *memory.DBManager, name string) (einotool.InvokableTool, error) {
	if db == nil {
		return nil, errors.New("nil task plan store")
	}
	return utils.InferTool(name, "更新复杂任务计划中一个步骤的可观察状态；优先使用 update_task_step", func(ctx context.Context, req *UpdateTaskStepRequest) (*UpdateTaskStepResponse, error) {
		principal, ok := toolruntime.PrincipalFromContext(ctx)
		if !ok {
			return nil, errors.New("missing runtime principal")
		}
		if err := db.UpdateTaskStep(ctx, principal.UserID, req.PlanID, req.StepKey, req.Status); err != nil {
			return nil, err
		}
		return &UpdateTaskStepResponse{Updated: true}, nil
	})
}
