package agent

import (
	"context"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	_type "liveclass/internal/rpc/agent/eino_gen/agent/type"
	my_prompt "liveclass/internal/rpc/agent/prompt"
)

func newChatTemplate(ctx context.Context) (ctp prompt.ChatTemplate, err error) {
	config := &_type.TemplateConfig{
		FormatType: schema.FString,
		Templates: []schema.MessagesTemplate{
			schema.SystemMessage(my_prompt.SystemPrompt),
			schema.MessagesPlaceholder("history", true),
			schema.UserMessage("{content}"),
		},
	}

	ctp = prompt.FromMessages(config.FormatType, config.Templates...)
	return ctp, nil
}
