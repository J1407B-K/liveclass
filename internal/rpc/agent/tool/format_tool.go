package tool

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type FormatRequest struct {
	Text   string `json:"text" jsonschema_description:"需要格式化的文本内容"`
	Format string `json:"format" jsonschema_description:"输出格式：json / markdown / plain"`
}

func NewFormatTool(_ string) (tool.BaseTool, error) {
	call := func(ctx context.Context, req *FormatRequest) (string, error) {
		_ = ctx
		if req.Format == "" {
			req.Format = "plain"
		}
		switch req.Format {
		case "plain":
			return req.Text, nil
		case "markdown":
			return "**输出结果**\n\n" + req.Text, nil
		case "json":
			out, err := json.Marshal(map[string]string{"content": req.Text})
			if err != nil {
				return "", fmt.Errorf("format output: %w", err)
			}
			return string(out), nil
		default:
			return "", fmt.Errorf("unsupported format %q", req.Format)
		}
	}

	return utils.InferTool("format_output", "将文本格式化为指定格式（json/markdown/plain），用于确定性输出处理", call)
}
