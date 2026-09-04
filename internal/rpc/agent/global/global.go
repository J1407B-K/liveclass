package global

import (
	"context"
	"liveclass/internal/rpc/agent/config"
	"liveclass/internal/rpc/agent/model"
	"os"
	"path/filepath"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	mallservice "liveclass/idl/kitex_gen/mall/mallservice"
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
	MallClient          mallservice.Client

	SkriptsDir = func() string {
		if d := os.Getenv("SCRIPTS_DIR"); d != "" {
			return d
		}
		return filepath.Join(os.Getenv("PWD"), "internal/rpc/agent/skills/scripts")
	}()
)

type TextMultiModalEmbedder interface {
	EmbedText(context.Context, string) ([]float64, error)
	EmbedTextBatch(context.Context, []string) ([][]float64, error)
}

var MultiModalEmbedder TextMultiModalEmbedder
