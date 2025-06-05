package initialize

import (
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	"liveclass/idl/kitex_gen/agent/agentservice"
	"liveclass/idl/kitex_gen/chat/chatservice"
	"liveclass/idl/kitex_gen/live/liveservice"
	"liveclass/idl/kitex_gen/quiz/quizservice"
	"liveclass/idl/kitex_gen/user/userservice"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/global"
)

func InitNewClient() error {
	// Userservice 客户端，附带 OpenTelemetry 链路追踪
	uc, err := userservice.NewClient(
		"userservice",
		client.WithResolver(*global.Resolver),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
	)
	if err != nil {
		panic(err)
	}
	global.Clients.UserClient = uc

	// Liveservice 客户端，附带 OpenTelemetry 链路追踪
	lc, err := liveservice.NewClient(
		"liveservice",
		client.WithResolver(*global.Resolver),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
	)
	if err != nil {
		panic(err)
	}
	global.Clients.LiveClient = lc

	// Quizservice 客户端，附带 OpenTelemetry 链路追踪
	qc, err := quizservice.NewClient(
		"quizservice",
		client.WithResolver(*global.Resolver),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
	)
	if err != nil {
		panic(err)
	}
	global.Clients.QuizClient = qc

	// Agentservice 客户端，附带 OpenTelemetry 链路追踪
	ac, err := agentservice.NewClient(
		"agentservice",
		client.WithResolver(*global.Resolver),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
	)
	if err != nil {
		panic(err)
	}
	global.Clients.AgentClient = ac

	// Chatservice 客户端，附带 OpenTelemetry 链路追踪
	cc, err := chatservice.NewClient(
		"chatservice",
		client.WithResolver(*global.Resolver),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
	)
	if err != nil {
		panic(err)
	}
	global.Clients.ChatClient = cc

	// WebrtcLiveService 客户端，附带 OpenTelemetry 链路追踪
	wc, err := webrtc_live.NewClient(
		"webrtc_liveservice",
		client.WithResolver(*global.Resolver),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
	)
	if err != nil {
		panic(err)
	}
	global.Clients.Webrtc_liveClient = wc

	return nil
}
