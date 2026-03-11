package fact

import (
	"context"
	my_prompt "liveclass/internal/rpc/agent/prompt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func newFactExtractTemplate(ctx context.Context) (prompt.ChatTemplate, error) {
	ctp := prompt.FromMessages(
		schema.FString,
		schema.SystemMessage(my_prompt.FactExtractSystemPrompt),
		schema.UserMessage("{message}"),
	)
	return ctp, nil
}
