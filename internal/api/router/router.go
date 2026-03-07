package router

import (
	"context"
	service2 "liveclass/internal/api/service"
	"liveclass/internal/api/utils/cors"
	"liveclass/internal/api/utils/jwt"
	"log"

	"github.com/cloudwego/hertz/pkg/app/server"
	prometheus "github.com/hertz-contrib/monitor-prometheus"
	"github.com/hertz-contrib/obs-opentelemetry/provider"
	"github.com/hertz-contrib/obs-opentelemetry/tracing"
)

func InitRouter() {
	p := provider.NewOpenTelemetryProvider(
		provider.WithServiceName("liveclass-api"),
		provider.WithExportEndpoint("localhost:4317"),
		provider.WithInsecure(),
		provider.WithEnableMetrics(false),
	)
	defer p.Shutdown(context.Background())

	tracer, cfg := tracing.NewServerTracer()

	prom := prometheus.NewServerTracer(":10001", "/metrics")

	h := server.New(server.WithHostPorts(":8080"),
		server.WithMaxRequestBodySize(1024*1024*1024),
		server.WithTracer(prom),
		tracer)

	h.NoHijackConnPool = true
	h.Use(cors.CORS())
	h.Use(tracing.ServerMiddleware(cfg))

	authMiddlewire, err := jwt.NewJWTMiddle()
	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}

	v1 := h.Group("/")
	{
		v1.POST("/register", service2.Register)
		v1.POST("/login", authMiddlewire.LoginHandler)
		v1.POST("/refresh", service2.RefreshToken)
		v1.GET("/userinfo", service2.GetUserInfo)
	}

	v2 := h.Group("/")
	v2.Use(authMiddlewire.MiddlewareFunc())
	{
		//webrtc
		v2.POST("/broadcast", service2.Broadcast)
		v2.POST("/view", service2.View)
		v2.POST("/create_lesson_webrtc", service2.CreateLesson)
		v2.DELETE("/del_lesson_webrtc", service2.DelLesson)
		v2.PUT("/change_user_to_lesson_webrtc", service2.ChangeUserToLesson_WebRTC)
		v2.GET("/change_user_in_live_webrtc", service2.ChangeUserInLive_WebRTC)
		v2.GET("/select_lesson_webrtc", service2.SelectLessonInfo_WebRTC)
		v2.GET("/get_lesson_webrtc", service2.GetLessonInfo_WebRTC)
		v2.POST("/create_signin_webrtc", service2.CreateSignIn_WebRTC)
		v2.PUT("/signin_webrtc", service2.SignIn_WebRTC)
		v2.GET("/select_signin_webrtc", service2.SelectSignIn_WebRTC)
		v2.DELETE("/del_signin_webrtc", service2.DelSignIn_WebRTC)
		v2.GET("/roll_call_webrtc", service2.RollCallInRandom_WebRTC)
		v2.POST("/record_lesson_webrtc", service2.RecordLesson_WebRTC)
		v2.GET("/list_lesson_record", service2.ListAllLessonRecord)
		v2.GET("/get_lesson_record", service2.GetLessonRecord)
		v2.POST("/save_whiteboard", service2.SaveWhiteBoard)
		//之后可以用现有WebSocket把白板json广播出去，现在暂时用http吧，而且前端实现中其实是iframe嵌入的网页(简单实现，被调得破防了hhh)
		v2.GET("/get_whiteboard", service2.GetWhiteBoard)
		v2.PUT("/raise_hand", service2.RaiseHand)
		v2.GET("/get_raise_hand", service2.GetRaiseHand)
		v2.PUT("/approve_hand", service2.ApproveHand)
		v2.POST("/publish_mic", service2.PublishMic)
		v2.POST("/view_mic", service2.ViewMic)

		v2.POST("/create_question", service2.CreateQuestion)
		v2.DELETE("/del_question", service2.DelQuestion)
		v2.GET("/get_question", service2.GetAllLessonQuiz)

		v2.POST("/chat_agent", service2.ChatWithAgent)
		v2.GET("/list_user_conv", service2.ListAllUserConv)
		v2.DELETE("/del_user_conv", service2.DelAllUserConv)

		v2.GET("/get_his", service2.GetHistory)
		v2.DELETE("/del_his", service2.DelHistory)

	}
	v3 := h.Group("/ws")
	{
		v3.GET("/quiz", service2.QuizConnection)

		v3.GET("/live_chat", service2.ChatConnections)
	}

	h.Spin()
}
