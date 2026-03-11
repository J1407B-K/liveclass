package service

import (
	"context"
	"errors"
	"liveclass/idl/kitex_gen/agent"
	"liveclass/internal/api/code"
	"liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

func ChatWithAgent(c context.Context, ctx *app.RequestContext) {
	uid := ctx.GetInt64("userid")
	if uid == 0 {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	msg := ctx.PostForm("message")

	conv_id := ctx.PostForm("conv_id")

	request_id := global.Node.Generate().String()

	resp, err := global.Clients.AgentClient.ChatWithAgent(c, &agent.ChatWithAgentReq{
		Userid:    uid,
		Message:   msg,
		RequestId: request_id,
		ConvId:    conv_id,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{
				Code: code.RPCError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model2.Response{
			Code: 0,
			Msg:  resp.Resp.Msg,
			Data: "nil",
		},
	})
}

func ListAllUserConv(c context.Context, ctx *app.RequestContext) {
	uid := ctx.GetInt64("userid")
	if uid == 0 {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	resp, err := global.Clients.AgentClient.ListAllUserConv(c, &agent.ListAllUserConvReq{
		Userid: uid,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{
				Code: code.RPCError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model2.Response{
			Code: 0,
			Msg:  "ok",
			Data: resp.Resp.Data,
		},
	})
}

func DelAllUserConv(c context.Context, ctx *app.RequestContext) {
	uid := ctx.GetInt64("userid")
	if uid == 0 {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	conv_id := ctx.PostForm("convid")

	resp, err := global.Clients.AgentClient.DelAllUserConv(c, &agent.DelAllUserConvReq{
		Userid: uid,
		ConvId: conv_id,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{
				Code: code.RPCError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}

	ctx.JSON(http.StatusOK, utils.H{
		"resp": model2.Response{
			Code: 0,
			Msg:  "ok",
			Data: resp.Resp.Data,
		},
	})
}
