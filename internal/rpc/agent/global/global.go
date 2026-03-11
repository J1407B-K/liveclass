package global

import (
	"context"
	"liveclass/internal/rpc/agent/config"
	"liveclass/internal/rpc/agent/model"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	userservice "liveclass/idl/kitex_gen/user/userservice"
	webrtclive "liveclass/idl/kitex_gen/webrtc_live/webrtclive"
)

var (
	Config = &config.Config{}

	AgentRunner         compose.Runnable[*model.UserMessage, *schema.Message]
	FactExtractorRunner compose.Runnable[*model.FactExtractInput, []*model.FactCandidate]
	ChatModel           *ark.ChatModel
	UserClient          userservice.Client
	LessonClient        webrtclive.Client
)

type TextMultiModalEmbedder interface {
	EmbedText(context.Context, string) ([]float64, error)
	EmbedTextBatch(context.Context, []string) ([][]float64, error)
}

var MultiModalEmbedder TextMultiModalEmbedder
