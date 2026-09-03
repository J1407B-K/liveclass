package toolruntime

import (
	"context"
	"encoding/json"
	"fmt"

	modelcomponent "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"liveclass/internal/rpc/agent/dependency"
)

type ModelRepairer struct {
	Model modelcomponent.BaseChatModel
}

func (r *ModelRepairer) Repair(ctx context.Context, spec ToolSpec, original string, validationErr error) (string, error) {
	if r == nil || r.Model == nil {
		return "", fmt.Errorf("nil repair model")
	}
	schemaJSON, _ := json.Marshal(spec.OutputSchema)
	messages := []*schema.Message{
		schema.SystemMessage("修复工具输出以符合 JSON schema。不得重新执行工具，不得增加原输出没有的事实。只返回 JSON。"),
		schema.UserMessage(fmt.Sprintf("tool=%s\nschema=%s\nvalidation_error=%s\noriginal_output=%s", spec.Name, schemaJSON, validationErr, original)),
	}
	resp, err := dependency.Do(ctx, dependency.MainLLM, "repair_tool_output", func(callCtx context.Context) (*schema.Message, error) {
		return r.Model.Generate(callCtx, messages)
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}
