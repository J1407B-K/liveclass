package service

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/internal/code"
	"liveclass/internal/global"
	"liveclass/internal/model"
	"net/http"
)

func Broadcast(c context.Context, ctx *app.RequestContext) {

	lid := ctx.PostForm("lesson_id")
	offer := ctx.PostForm("b64offer")

	resp, err := global.Clients.Webrtc_liveClient.Broadcast(c, &webrtc_live.BroadcastReq{
		LessonId: lid,
		B64offer: offer,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model.Response{
				Code: code.RPCError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model.Response{
			Code: 0,
			Msg:  "ok",
			Data: resp.Resp.Data,
		},
	})
}

func View(c context.Context, ctx *app.RequestContext) {
	lid := ctx.PostForm("lesson_id")
	offer := ctx.PostForm("b64offer")

	resp, err := global.Clients.Webrtc_liveClient.View(c, &webrtc_live.ViewReq{
		LessonId: lid,
		B64offer: offer,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model.Response{
				Code: code.RPCError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model.Response{
			Code: 0,
			Msg:  "ok",
			Data: resp.Resp.Data,
		},
	})
}
