package agent

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/schema"

	"liveclass/internal/rpc/agent/memory"
	"liveclass/internal/rpc/agent/tool"
)

func TestOutputSchemasAreMaterialized(t *testing.T) {
	object := outputSchema[tool.UserInfoResponse]()
	if object == nil || object.Type != "object" || object.Properties["user_id"] == nil {
		t.Fatalf("unexpected object schema: %#v", object)
	}
	array := outputSchema[[]tool.SearchResult]()
	if array == nil || array.Type != "array" || array.Items == nil {
		t.Fatalf("unexpected array schema: %#v", array)
	}
}

func TestToolInputSchemasAreObjects(t *testing.T) {
	tools := []struct {
		name string
		new  func() (interface {
			Info(context.Context) (*schema.ToolInfo, error)
		}, error)
	}{
		{name: "search", new: func() (interface {
			Info(context.Context) (*schema.ToolInfo, error)
		}, error) {
			return tool.NewSearchTool()
		}},
		{name: "format", new: func() (interface {
			Info(context.Context) (*schema.ToolInfo, error)
		}, error) {
			return tool.NewFormatTool(".")
		}},
		{name: "plan", new: func() (interface {
			Info(context.Context) (*schema.ToolInfo, error)
		}, error) {
			return tool.NewCreateTaskPlanTool(&memory.DBManager{})
		}},
	}
	for _, tc := range tools {
		base, err := tc.new()
		if err != nil {
			t.Fatal(err)
		}
		info, err := base.Info(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		openapiSchema, err := info.ParamsOneOf.ToOpenAPIV3()
		if err != nil {
			t.Fatal(err)
		}
		if openapiSchema == nil || openapiSchema.Type != "object" {
			t.Fatalf("%s tool input must be an object: %#v", tc.name, openapiSchema)
		}
	}
}
