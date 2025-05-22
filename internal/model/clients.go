package model

import (
	agent "liveclass/idl/kitex_gen/agent/agentservice"
	live "liveclass/idl/kitex_gen/live/liveservice"
	quiz "liveclass/idl/kitex_gen/quiz/quizservice"
	user "liveclass/idl/kitex_gen/user/userservice"
)

type Clients struct {
	UserClient  user.Client
	LiveClient  live.Client
	QuizClient  quiz.Client
	AgentClient agent.Client
}
