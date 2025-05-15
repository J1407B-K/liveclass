package api

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"liveclass/internal/service"
	"liveclass/internal/utils/jwt"
	"log"
)

func InitRouter() {
	h := server.New(server.WithHostPorts(":8080"))

	authMiddlewire, err := jwt.NewMiddle()
	if err != nil {
		log.Fatal("JWT Error:" + err.Error())
	}

	v1 := h.Group("/")
	{
		v1.POST("/register", service.Register)
		v1.POST("/login", authMiddlewire.LoginHandler)
		v1.GET("/userinfo", service.GetUserInfo)
	}

	v2 := h.Group("/")
	v2.Use(authMiddlewire.MiddlewareFunc())
	{
		v2.POST("/create_live", service.CreateLive)
		v2.POST("/close_live", service.CloseLive)
		v2.PUT("/change_user_in_live", service.ChangeUserInLive)
		v2.GET("/select_lesson", service.SelectLessonInfo)
		v2.GET("/get_lesson", service.GetLessonInfo)
	}

	h.Spin()
}
