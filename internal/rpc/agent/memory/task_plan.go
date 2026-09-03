package memory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"liveclass/internal/rpc/agent/agentmetrics"
	"liveclass/internal/rpc/agent/model"
)

func (m *DBManager) ActiveTaskPlanContext(ctx context.Context, sessionID string) (string, error) {
	if m == nil || m.DB == nil || strings.TrimSpace(sessionID) == "" {
		return "", nil
	}
	var plan model.TaskPlan
	err := m.DB.WithContext(ctx).Where("session_id = ? AND status IN ?", sessionID, []string{"pending", "running"}).Order("updated_at DESC").First(&plan).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var steps []model.TaskStep
	if err := m.DB.WithContext(ctx).Where("plan_id = ?", plan.ID).Order("id ASC").Find(&steps).Error; err != nil {
		return "", err
	}
	var out strings.Builder
	fmt.Fprintf(&out, "计划 ID：%d\n目标：%s（%s）\n该计划已由运行时持久化；不要重复创建。只有实际执行某一步后才能更新其状态，当前仅制定方案时不要调用更新工具。", plan.ID, plan.Goal, plan.Status)
	for _, step := range steps {
		fmt.Fprintf(&out, "\n- [%s] %s: %s", step.Status, step.StepKey, step.Description)
	}
	return out.String(), nil
}

type NewTaskStep struct {
	Key, Description string
	DependsOn        string
}

func (m *DBManager) CreateTaskPlan(ctx context.Context, userID int64, sessionID, requestID, goal string, steps []NewTaskStep) (*model.TaskPlan, error) {
	if m == nil || m.DB == nil {
		return nil, errors.New("nil db")
	}
	if userID <= 0 || sessionID == "" || requestID == "" || goal == "" || len(steps) < 2 {
		return nil, errors.New("task plan requires a goal and at least two steps")
	}
	var plan model.TaskPlan
	created := false
	err := postgresWriteError(ctx, "create_task_plan", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Transaction(func(tx *gorm.DB) error {
			plan = model.TaskPlan{UserID: userID, SessionID: sessionID, RequestID: requestID, Goal: goal, Status: "pending"}
			result := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "session_id"}, {Name: "request_id"}}, DoNothing: true}).Create(&plan)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return tx.Where("session_id = ? AND request_id = ?", sessionID, requestID).First(&plan).Error
			}
			if plan.ID == 0 {
				return errors.New("created task plan has no id")
			}
			created = true
			for _, input := range steps {
				if input.Key == "" || input.Description == "" {
					return errors.New("task step key and description are required")
				}
				step := model.TaskStep{PlanID: plan.ID, StepKey: input.Key, Description: input.Description, Status: "pending", DependsOn: input.DependsOn}
				if step.DependsOn == "" {
					step.DependsOn = "[]"
				}
				if err := tx.Create(&step).Error; err != nil {
					return err
				}
			}
			return nil
		})
	})
	if err == nil && created {
		agentmetrics.TaskPlans.Inc()
	}
	return &plan, err
}

func (m *DBManager) UpdateTaskStep(ctx context.Context, userID, planID int64, stepKey, next string) error {
	if m == nil || m.DB == nil {
		return errors.New("nil db")
	}
	if next != "running" && next != "done" && next != "failed" {
		return fmt.Errorf("invalid task status %q", next)
	}
	var previousUpdate time.Time
	err := postgresWriteError(ctx, "update_task_step", func(callCtx context.Context) error {
		return m.DB.WithContext(callCtx).Transaction(func(tx *gorm.DB) error {
			var plan model.TaskPlan
			if err := tx.Where("id = ? AND user_id = ?", planID, userID).First(&plan).Error; err != nil {
				return err
			}
			previousUpdate = plan.UpdatedAt
			var step model.TaskStep
			if err := tx.Where("plan_id = ? AND step_key = ?", planID, stepKey).First(&step).Error; err != nil {
				return err
			}
			if !validTaskTransition(step.Status, next) {
				return fmt.Errorf("invalid task transition %s -> %s", step.Status, next)
			}
			if next == "running" {
				var dependencies []string
				if err := json.Unmarshal([]byte(step.DependsOn), &dependencies); err != nil {
					return fmt.Errorf("invalid step dependencies: %w", err)
				}
				if len(dependencies) > 0 {
					var done int64
					if err := tx.Model(&model.TaskStep{}).Where("plan_id = ? AND step_key IN ? AND status = ?", planID, dependencies, "done").Count(&done).Error; err != nil {
						return err
					}
					if done != int64(len(dependencies)) {
						return errors.New("task dependencies are not complete")
					}
				}
			}
			if err := tx.Model(&step).Update("status", next).Error; err != nil {
				return err
			}
			var remaining, failed int64
			if err := tx.Model(&model.TaskStep{}).Where("plan_id = ? AND status NOT IN ?", planID, []string{"done", "failed"}).Count(&remaining).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.TaskStep{}).Where("plan_id = ? AND status = ?", planID, "failed").Count(&failed).Error; err != nil {
				return err
			}
			planStatus := "running"
			if remaining == 0 {
				if failed > 0 {
					planStatus = "failed"
				} else {
					planStatus = "done"
				}
			}
			return tx.Model(&plan).Update("status", planStatus).Error
		})
	})
	if err == nil {
		agentmetrics.TaskStepUpdates.WithLabelValues(next).Inc()
		if !previousUpdate.IsZero() {
			agentmetrics.PlanUpdateInterval.Observe(time.Since(previousUpdate).Seconds())
		}
	}
	return err
}

func validTaskTransition(current, next string) bool {
	if current == next {
		return true
	}
	switch current {
	case "pending":
		return next == "running" || next == "failed"
	case "running":
		return next == "done" || next == "failed"
	default:
		return false
	}
}

func (m *DBManager) GetTaskPlan(ctx context.Context, userID, planID int64) (*model.TaskPlan, []model.TaskStep, error) {
	if m == nil || m.DB == nil {
		return nil, nil, errors.New("nil db")
	}
	plan, err := postgresRead(ctx, "get_task_plan", func(callCtx context.Context) (model.TaskPlan, error) {
		var p model.TaskPlan
		err := m.DB.WithContext(callCtx).Where("id = ? AND user_id = ?", planID, userID).First(&p).Error
		return p, err
	})
	if err != nil {
		return nil, nil, err
	}
	steps, err := postgresRead(ctx, "get_task_steps", func(callCtx context.Context) ([]model.TaskStep, error) {
		var rows []model.TaskStep
		err := m.DB.WithContext(callCtx).Where("plan_id = ?", planID).Order("id asc").Find(&rows).Error
		return rows, err
	})
	return &plan, steps, err
}
