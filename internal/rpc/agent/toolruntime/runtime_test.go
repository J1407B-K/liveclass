package toolruntime

import (
	"context"
	"testing"
	"time"

	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
	"github.com/getkin/kin-openapi/openapi3"
)

type ownRequest struct {
	UserID int64 `json:"user_id"`
}

type invalidOutputTool struct{}

func (invalidOutputTool) Info(context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "invalid", Desc: "invalid"}, nil
}
func (invalidOutputTool) InvokableRun(context.Context, string, ...einotool.Option) (string, error) {
	return `{"wrong":true}`, nil
}

func TestRegistryValidatesOutput(t *testing.T) {
	registry := NewRegistry(nil)
	output := &openapi3.Schema{
		Type:       "object",
		Properties: map[string]*openapi3.SchemaRef{"ok": {Value: &openapi3.Schema{Type: "boolean"}}},
		Required:   []string{"ok"},
	}
	if err := registry.Register(context.Background(), invalidOutputTool{}, ToolSpec{Name: "invalid", Permission: PermissionAuthenticated, OutputSchema: output}); err != nil {
		t.Fatal(err)
	}
	_, err := registry.Tools()[0].(*wrappedTool).InvokableRun(WithPrincipal(context.Background(), Principal{UserID: 1}), `{}`)
	if err == nil {
		t.Fatal("expected output validation error")
	}
}

type ownResponse struct {
	OK bool `json:"ok"`
}

func TestRegistryEnforcesOwnUserPermission(t *testing.T) {
	base, err := utils.InferTool("own", "own data", func(_ context.Context, req *ownRequest) (*ownResponse, error) { return &ownResponse{OK: true}, nil })
	if err != nil {
		t.Fatal(err)
	}
	r := NewRegistry(nil)
	if err := r.Register(context.Background(), base, ToolSpec{Name: "own", Permission: PermissionOwnUser, Timeout: time.Second}); err != nil {
		t.Fatal(err)
	}
	wrapped := r.Tools()[0].(*wrappedTool)
	ctx := WithPrincipal(context.Background(), Principal{UserID: 7})
	if _, err := wrapped.InvokableRun(ctx, `{"user_id":8}`); err == nil {
		t.Fatal("expected cross-user call to be denied")
	}
	if _, err := wrapped.InvokableRun(ctx, `{"user_id":7}`); err != nil {
		t.Fatalf("own-user call failed: %v", err)
	}
}

func TestRegistryValidatesInputBeforeExecution(t *testing.T) {
	called := false
	base, _ := utils.InferTool("validated", "validated", func(_ context.Context, req *ownRequest) (*ownResponse, error) {
		called = true
		return &ownResponse{OK: true}, nil
	})
	r := NewRegistry(nil)
	_ = r.Register(context.Background(), base, ToolSpec{Name: "validated", Permission: PermissionAuthenticated})
	wrapped := r.Tools()[0].(*wrappedTool)
	_, err := wrapped.InvokableRun(WithPrincipal(context.Background(), Principal{UserID: 1}), `{}`)
	if err == nil || called {
		t.Fatalf("invalid input reached tool: err=%v called=%v", err, called)
	}
}

func TestRegistryBlocksTaskPlanForOrdinaryQA(t *testing.T) {
	base, _ := utils.InferTool("plan", "plan", func(_ context.Context, req *ownRequest) (*ownResponse, error) {
		return &ownResponse{OK: true}, nil
	})
	r := NewRegistry(nil)
	_ = r.Register(context.Background(), base, ToolSpec{Name: "plan", Permission: PermissionAuthenticated, Metadata: map[string]string{"complex_task": "required"}})
	wrapped := r.Tools()[0].(*wrappedTool)
	if _, err := wrapped.InvokableRun(WithPrincipal(context.Background(), Principal{UserID: 1}), `{"user_id":1}`); err == nil {
		t.Fatal("ordinary QA principal must not create a plan")
	}
	if _, err := wrapped.InvokableRun(WithPrincipal(context.Background(), Principal{UserID: 1, AllowPlanning: true}), `{"user_id":1}`); err != nil {
		t.Fatalf("complex task plan was denied: %v", err)
	}
}
