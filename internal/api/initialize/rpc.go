package initialize

import (
	"liveclass/idl/kitex_gen/agent/agentservice"
	"liveclass/idl/kitex_gen/chat/chatservice"
	"liveclass/idl/kitex_gen/mall/mallservice"
	"liveclass/idl/kitex_gen/quiz/quizservice"
	"liveclass/idl/kitex_gen/user/userservice"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/api/global"
	"liveclass/internal/api/utils"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
)

func InitNewClient() error {
	// Userservice 客户端，附带 OpenTelemetry 链路追踪
	uc, err := userservice.NewClient(
		"userservice",
		client.WithResolver(*global.Resolver),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
		client.WithRPCTimeout(800*time.Millisecond),
	)
	if err != nil {
		panic(err)
	}
	global.Clients.UserClient = uc

	// Quizservice 客户端，附带 OpenTelemetry 链路追踪
	qc, err := quizservice.NewClient(
		"quizservice",
		client.WithResolver(*global.Resolver),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
		client.WithRPCTimeout(800*time.Millisecond),
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
		// Complex Agent requests have their own 120s Plan Executor deadline.
		// Keep transport timeout higher so Runtime, not the API client, owns it.
		client.WithRPCTimeout(180*time.Second),
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
		client.WithRPCTimeout(2*time.Second),
	)
	if err != nil {
		panic(err)
	}
	global.Clients.ChatClient = cc

	mc, err := mallservice.NewClient(
		"mallservice",
		client.WithResolver(*global.Resolver),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
		client.WithRPCTimeout(20*time.Second),
	)
	if err != nil {
		panic(err)
	}
	global.Clients.MallClient = mc

	// WebrtcLiveService 客户端，附带 OpenTelemetry 链路追踪和一致性哈希负载均衡
	wc, err := webrtc_live.NewClient(
		"webrtc_liveservice",
		client.WithResolver(*global.Resolver),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler),
		client.WithLoadBalancer(utils.NewConsistentHashLoadBalancer()),
		client.WithRPCTimeout(800*time.Millisecond),
	)
	if err != nil {
		panic(err)
	}
	global.Clients.Webrtc_liveClient = wc

	return nil
}
