package api

import (
	"github.com/cloudwego/hertz/pkg/app/server"
	"liveclass/internal/handler"
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
		v1.POST("/register", handler.Register)
		v1.POST("/login", authMiddlewire.LoginHandler)
	}
	v2 := h.Group("/")
	v2.Use(authMiddlewire.MiddlewareFunc())
	{

	}

	h.Spin()
}
