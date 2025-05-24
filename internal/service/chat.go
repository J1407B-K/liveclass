package service

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"liveclass/idl/kitex_gen/chat"
	"liveclass/internal/code"
	"liveclass/internal/global"
	"liveclass/internal/model"
	"net/http"
)

func GetHistory(c context.Context, ctx *app.RequestContext) {
	lid := ctx.PostForm("lesson_id")
	resp, err := global.Clients.ChatClient.GetHistory(c, &chat.GetHistoryReq{LessonId: lid})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model.Response{
				Code: code.RPCError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
	}
	ctx.JSON(http.StatusOK, utils.H{
		"resp": model.Response{
			Code: 0,
			Msg:  "ok",
			Data: resp.Resp.Data,
		},
	})
}

func DelHistory(c context.Context, ctx *app.RequestContext) {
	lid := ctx.PostForm("lesson_id")

	resp, err := global.Clients.ChatClient.DelHistory(c, &chat.DelHistoryReq{LessonId: lid})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model.Response{
				Code: code.RPCError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
	}

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model.Response{
			Code: 0,
			Msg:  "ok",
			Data: resp.Resp.Data,
		},
	})
}
