package model

import (
	agent "liveclass/idl/kitex_gen/agent/agentservice"
	chat "liveclass/idl/kitex_gen/chat/chatservice"
	quiz "liveclass/idl/kitex_gen/quiz/quizservice"
	user "liveclass/idl/kitex_gen/user/userservice"
	webrtc_live "liveclass/idl/kitex_gen/webrtc_live/webrtclive"
)

type Clients struct {
	UserClient        user.Client
	QuizClient        quiz.Client
	AgentClient       agent.Client
	ChatClient        chat.Client
	Webrtc_liveClient webrtc_live.Client
}
