package agent

import (
	"context"
	"liveclass/internal/rpc/agent/model"
	my_prompt "liveclass/internal/rpc/agent/prompt"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

func newChatTemplate(ctx context.Context) (prompt.ChatTemplate, error) {
	config := &model.TemplateConfig{
		FormatType: schema.FString,
		Templates: []schema.MessagesTemplate{
			schema.SystemMessage(my_prompt.SystemPrompt),
			schema.SystemMessage("以下是对当前用户的画像摘要，请酌情参考：\n{profile}"),
			schema.SystemMessage("以下是与当前用户相关的历史事实与偏好，请仅在相关时参考：\n{facts}"),
			schema.SystemMessage("当前用户ID：{user_id}"),
			schema.MessagesPlaceholder("history", true),
			schema.UserMessage("{content}"),
		},
	}

	return prompt.FromMessages(config.FormatType, config.Templates...), nil
}
