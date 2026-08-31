package tool

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type FormatRequest struct {
	Text   string `json:"text" jsonschema_description:"需要格式化的文本内容"`
	Format string `json:"format" jsonschema_description:"输出格式：json / markdown / plain"`
}

func NewFormatTool(scriptsDir string) (tool.BaseTool, error) {
	scriptPath := filepath.Join(scriptsDir, "format_output.py")

	call := func(ctx context.Context, req *FormatRequest) (string, error) {
		if req.Format == "" {
			req.Format = "plain"
		}
		out, err := exec.CommandContext(ctx, "python3", scriptPath,
			"--text", req.Text,
			"--format", req.Format,
		).Output()
		if err != nil {
			return "", fmt.Errorf("format_output script: %w", err)
		}
		return strings.TrimSpace(string(out)), nil
	}

	return utils.InferTool("format_output", "将文本格式化为指定格式（json/markdown/plain），用于确定性输出处理", call)
}
