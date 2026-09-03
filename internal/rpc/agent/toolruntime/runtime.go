package toolruntime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/getkin/kin-openapi/openapi3"

	"liveclass/internal/rpc/agent/agentmetrics"
	"liveclass/internal/rpc/agent/dependency"
	agenttrace "liveclass/internal/rpc/agent/trace"
)

type Permission string

const (
	PermissionPublic        Permission = "public"
	PermissionAuthenticated Permission = "authenticated"
	PermissionOwnUser       Permission = "own_user"
	PermissionLessonMember  Permission = "lesson_member"
	PermissionAdmin         Permission = "admin"
	PermissionApproval      Permission = "approval"
)

type RiskLevel string

const (
	RiskReadOnly RiskLevel = "read_only"
	RiskLow      RiskLevel = "low"
	RiskHigh     RiskLevel = "high"
)

type RetryPolicy struct {
	Attempts int
	Backoff  time.Duration
}

type ToolSpec struct {
	Name         string
	Description  string
	InputSchema  *openapi3.Schema
	OutputSchema *openapi3.Schema
	Permission   Permission
	Timeout      time.Duration
	Retry        RetryPolicy
	RiskLevel    RiskLevel
	Metadata     map[string]string
}

type Principal struct {
	UserID        int64
	Role          string
	LessonID      int64
	SessionID     string
	RequestID     string
	ApprovedTools map[string]bool
	AllowedTools  map[string]bool
	AllowPlanning bool
}

type principalKey struct{}

func WithPrincipal(ctx context.Context, p Principal) context.Context {
	return context.WithValue(ctx, principalKey{}, p)
}
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

type Repairer interface {
	Repair(context.Context, ToolSpec, string, error) (string, error)
}

type Registry struct {
	specs    map[string]ToolSpec
	tools    []einotool.BaseTool
	repairer Repairer
}

func NewRegistry(repairer Repairer) *Registry {
	return &Registry{specs: make(map[string]ToolSpec), repairer: repairer}
}

func (r *Registry) Register(ctx context.Context, base einotool.BaseTool, spec ToolSpec) error {
	if base == nil {
		return errors.New("nil tool")
	}
	info, err := base.Info(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(spec.Name) == "" {
		spec.Name = info.Name
	}
	if spec.Name != info.Name {
		return fmt.Errorf("tool spec name %q does not match tool %q", spec.Name, info.Name)
	}
	if _, exists := r.specs[spec.Name]; exists {
		return fmt.Errorf("duplicate tool %q", spec.Name)
	}
	if spec.Description == "" {
		spec.Description = info.Desc
	}
	if spec.InputSchema == nil && info.ParamsOneOf != nil {
		spec.InputSchema, err = info.ParamsOneOf.ToOpenAPIV3()
		if err != nil {
			return err
		}
	}
	if spec.Permission == "" {
		spec.Permission = PermissionAuthenticated
	}
	if spec.Timeout <= 0 {
		spec.Timeout = 3 * time.Second
	}
	if spec.Retry.Attempts <= 0 {
		spec.Retry.Attempts = 1
	}
	if spec.RiskLevel == "" {
		spec.RiskLevel = RiskReadOnly
	}
	invokable, ok := base.(einotool.InvokableTool)
	if !ok {
		return fmt.Errorf("tool %q is not invokable", spec.Name)
	}
	r.specs[spec.Name] = spec
	r.tools = append(r.tools, &wrappedTool{base: invokable, spec: spec, repairer: r.repairer})
	return nil
}

func (r *Registry) Tools() []einotool.BaseTool        { return append([]einotool.BaseTool(nil), r.tools...) }
func (r *Registry) Spec(name string) (ToolSpec, bool) { spec, ok := r.specs[name]; return spec, ok }

type wrappedTool struct {
	base     einotool.InvokableTool
	spec     ToolSpec
	repairer Repairer
}

func (w *wrappedTool) Info(ctx context.Context) (*schema.ToolInfo, error) { return w.base.Info(ctx) }

func (w *wrappedTool) InvokableRun(ctx context.Context, arguments string, opts ...einotool.Option) (string, error) {
	started := time.Now()
	if run := agenttrace.FromContext(ctx); run != nil {
		run.Record(ctx, "tool_call", w.spec.Name, "started", 0, map[string]any{"risk_level": w.spec.RiskLevel})
		run.RecordTranscript(ctx, "tool_call", w.spec.Name, bounded(arguments, 8192), map[string]any{"tool": w.spec.Name})
	}
	var input any
	if err := json.Unmarshal([]byte(arguments), &input); err != nil {
		return "", fmt.Errorf("validate %s input JSON: %w", w.spec.Name, err)
	}
	if w.spec.InputSchema != nil {
		if err := w.spec.InputSchema.VisitJSON(input); err != nil {
			return "", fmt.Errorf("validate %s input schema: %w", w.spec.Name, err)
		}
	}
	if err := authorize(ctx, w.spec, input); err != nil {
		record(ctx, w.spec.Name, "denied", started, err)
		return "", err
	}

	attempts := w.spec.Retry.Attempts
	var output string
	var err error
	for attempt := 1; attempt <= attempts; attempt++ {
		callCtx, cancel := context.WithTimeout(ctx, w.spec.Timeout)
		output, err = w.base.InvokableRun(callCtx, arguments, opts...)
		cancel()
		if err == nil || !dependency.IsTransient(err) || attempt == attempts {
			break
		}
		if run := agenttrace.FromContext(ctx); run != nil {
			run.Record(ctx, "retry", w.spec.Name, "scheduled", 0, map[string]any{"attempt": attempt + 1, "error": err.Error()})
		}
		timer := time.NewTimer(w.spec.Retry.Backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			err = ctx.Err()
		case <-timer.C:
		}
	}
	if err == nil && w.spec.OutputSchema != nil {
		err = validateOutput(w.spec.OutputSchema, output)
		if err != nil && w.repairer != nil {
			agentmetrics.Repairs.WithLabelValues("tool_output", "attempt").Inc()
			output, err = w.repairer.Repair(ctx, w.spec, output, err)
			if err == nil {
				err = validateOutput(w.spec.OutputSchema, output)
			}
			repairStatus := "success"
			if err != nil {
				repairStatus = "error"
			}
			agentmetrics.Repairs.WithLabelValues("tool_output", repairStatus).Inc()
		}
	}
	status := "ok"
	if err != nil {
		status = "error"
	}
	if tracker := ProgressTrackerFromContext(ctx); tracker != nil {
		tracker.ObserveTool(w.spec.Name, err == nil)
	}
	record(ctx, w.spec.Name, status, started, err)
	if run := agenttrace.FromContext(ctx); run != nil {
		run.RecordTranscript(ctx, "tool_result", w.spec.Name, bounded(output, 16384), map[string]any{"tool": w.spec.Name, "status": status})
	}
	return output, err
}

func bounded(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max] + "\n[truncated]"
}

func validateOutput(schema *openapi3.Schema, output string) error {
	var value any
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		return err
	}
	return schema.VisitJSON(value)
}

func authorize(ctx context.Context, spec ToolSpec, input any) error {
	if spec.Permission == PermissionPublic {
		return nil
	}
	p, ok := PrincipalFromContext(ctx)
	if !ok || p.UserID <= 0 {
		return errors.New("tool permission denied: unauthenticated")
	}
	if p.AllowedTools != nil && !p.AllowedTools[spec.Name] {
		return errors.New("tool permission denied: tool is not allowlisted")
	}
	if roles := strings.TrimSpace(spec.Metadata["allowed_roles"]); roles != "" {
		allowed := false
		for _, role := range strings.Split(roles, ",") {
			if strings.TrimSpace(role) == p.Role {
				allowed = true
				break
			}
		}
		if !allowed {
			return errors.New("tool permission denied: role is not allowlisted")
		}
	}
	if spec.RiskLevel == RiskHigh && !p.ApprovedTools[spec.Name] {
		return errors.New("tool permission denied: high-risk tool requires approval")
	}
	if spec.Metadata["complex_task"] == "required" && !p.AllowPlanning {
		return errors.New("tool permission denied: task is not complex enough for a plan")
	}
	switch spec.Permission {
	case PermissionAuthenticated:
		return nil
	case PermissionOwnUser:
		obj, _ := input.(map[string]any)
		requested, _ := obj["user_id"].(float64)
		if int64(requested) != p.UserID {
			return errors.New("tool permission denied: cross-user access")
		}
		return nil
	case PermissionLessonMember:
		if p.LessonID <= 0 {
			return errors.New("tool permission denied: no lesson scope")
		}
		return nil
	case PermissionAdmin:
		if p.Role != "admin" {
			return errors.New("tool permission denied: admin required")
		}
		return nil
	case PermissionApproval:
		if !p.ApprovedTools[spec.Name] {
			return errors.New("tool permission denied: approval required")
		}
		return nil
	default:
		return errors.New("tool permission denied: unknown policy")
	}
}

func record(ctx context.Context, name, status string, started time.Time, err error) {
	agentmetrics.ToolCalls.WithLabelValues(name, status).Inc()
	run := agenttrace.FromContext(ctx)
	if run == nil {
		return
	}
	meta := map[string]any{}
	if err != nil {
		meta["error"] = err.Error()
	}
	if run.ToolCallCount(name) > 0 {
		agentmetrics.DuplicateToolCalls.WithLabelValues(name).Inc()
		meta["duplicate"] = true
	}
	run.Record(ctx, "tool_result", name, status, time.Since(started), meta)
}
