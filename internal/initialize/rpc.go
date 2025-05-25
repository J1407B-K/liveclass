package initialize

import (
	"github.com/cloudwego/kitex/client"
	"liveclass/idl/kitex_gen/agent/agentservice"
	"liveclass/idl/kitex_gen/chat/chatservice"
	"liveclass/idl/kitex_gen/live/liveservice"
	"liveclass/idl/kitex_gen/quiz/quizservice"
	"liveclass/idl/kitex_gen/user/userservice"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/global"
)

func InitNewClient() error {
	uc, err := userservice.NewClient("userservice", client.WithResolver(*global.Resolver))
	if err != nil {
		panic(err)
	}
	global.Clients.UserClient = uc

	lc, err := liveservice.NewClient("liveservice", client.WithResolver(*global.Resolver))
	if err != nil {
		panic(err)
	}
	global.Clients.LiveClient = lc

	qc, err := quizservice.NewClient("quizservice", client.WithResolver(*global.Resolver))
	if err != nil {
		panic(err)
	}
	global.Clients.QuizClient = qc

	ac, err := agentservice.NewClient("agentservice", client.WithResolver(*global.Resolver))
	if err != nil {
		panic(err)
	}
	global.Clients.AgentClient = ac

	cc, err := chatservice.NewClient("chatservice", client.WithResolver(*global.Resolver))
	if err != nil {
		panic(err)
	}
	global.Clients.ChatClient = cc

	wc, err := webrtc_live.NewClient("webrtc_liveservice", client.WithResolver(*global.Resolver))
	if err != nil {
		panic(err)
	}
	global.Clients.Webrtc_liveClient = wc

	return nil
}
