package service

import (
	"context"
	"errors"
	"fmt"
	"liveclass/idl/kitex_gen/agent"
	"liveclass/internal/api/code"
	"liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"
	"liveclass/internal/api/utils/ratelimit"
	"net/http"
	"strconv"
	"time"

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
	request_id := ctx.PostForm("request_id")
	lessonStr := ctx.PostForm("lesson_id")
	var lessonID int64
	if lessonStr != "" {
		id, err := strconv.ParseInt(lessonStr, 10, 64)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, utils.H{
				"resp": model2.Response{
					Code: code.BadRequest,
					Msg:  "lesson_id must be int",
				},
			})
			return
		}
		lessonID = id
	}

	allowed, err := ratelimit.AllowRedis(
		c,
		global.DBManager.RDB,
		fmt.Sprintf("rl:agent:chat:%d", uid),
		2,
		6,
		1,
		time.Minute,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{
				Code: code.InternalError,
				Msg:  "限流错误: " + err.Error(),
			},
		})
		return
	}
	if !allowed {
		ctx.JSON(http.StatusTooManyRequests, utils.H{
			"resp": model2.Response{
				Code: code.TooManyRequests,
				Msg:  "请求过于频繁，请稍后重试",
			},
		})
		return
	}

	resp, err := global.Clients.AgentClient.ChatWithAgent(c, &agent.ChatWithAgentReq{
		Userid:    uid,
		Message:   msg,
		RequestId: request_id,
		ConvId:    conv_id,
		LessonId:  lessonID,
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
