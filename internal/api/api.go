package api

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"liveclass/internal/service"
	"liveclass/internal/utils/cors"
	"liveclass/internal/utils/jwt"
	"log"
)

func InitRouter() {
	h := server.New(server.WithHostPorts(":8080"))
	h.NoHijackConnPool = true

	h.Use(cors.CORS())

	authMiddlewire, err := jwt.NewMiddle()
	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}

	v1 := h.Group("/")
	{
		v1.POST("/register", service.Register)
		v1.POST("/login", authMiddlewire.LoginHandler)
		v1.GET("/userinfo", service.GetUserInfo)
		v1.GET("/is_stu_in_lesson", service.IsStudentInLesson)
	}

	v2 := h.Group("/")
	v2.Use(authMiddlewire.MiddlewareFunc())
	{
		//livego
		v2.POST("/create_live", service.CreateLive)
		v2.DELETE("/close_live", service.CloseLive)
		//前端接口，进入or退出直播间直接调用
		v2.PUT("/change_user_in_live", service.ChangeUserInLive)
		v2.PUT("/change_user_to_lesson", service.ChangeUserToLesson)
		//这个是直播间在线人数信息
		v2.GET("/select_lesson", service.SelectLessonInfo)
		//MYSQL中课程信息
		v2.GET("/get_lesson", service.GetLessonInfo)
		v2.GET("/record_lesson", service.RecordLesson)
		v2.POST("/create_signin", service.CreateSignIn)
		v2.PUT("/signin", service.SignIn)
		v2.GET("/select_signin", service.SelectSignIn)
		v2.DELETE("/del_signin", service.DelSignIn)
		v2.GET("/roll_call", service.RollCallInRandom)

		//webrtc
		v2.POST("/broadcast", service.Broadcast)
		v2.POST("/view", service.View)
		v2.POST("/create_lesson_webrtc", service.CreateLesson)
		v2.DELETE("/del_lesson_webrtc", service.DelLesson)
		v2.PUT("/change_user_to_lesson_webrtc", service.ChangeUserToLesson_WebRTC)
		v2.GET("/change_user_in_live_webrtc", service.ChangeUserInLive_WebRTC)
		v2.GET("/select_lesson_webrtc", service.SelectLessonInfo)
		v2.GET("/get_lesson_webrtc", service.GetLessonInfo)
		v2.POST("/create_signin_webrtc", service.CreateSignIn_WebRTC)
		v2.PUT("/signin_webrtc", service.SignIn_WebRTC)
		v2.GET("/select_signin_webrtc", service.SelectSignIn_WebRTC)
		v2.DELETE("/del_signin_webrtc", service.DelSignIn_WebRTC)
		v2.GET("/roll_call_webrtc", service.RollCallInRandom_WebRTC)

		v2.POST("/create_question", service.CreateQuestion)
		v2.DELETE("/del_question", service.DelQuestion)

		v2.POST("/chat_agent", service.ChatWithAgent)
		v2.GET("/list_user_conv", service.ListAllUserConv)
		v2.DELETE("/del_user_conv", service.DelAllUserConv)

		v2.GET("/get_his", service.GetHistory)
		v2.DELETE("/del_his", service.DelHistory)

	}
	v3 := h.Group("/ws")
	{
		v3.GET("/quiz", service.QuizConnection)

		v3.GET("/live_chat", service.ChatConnections)
	}

	h.Spin()
}
