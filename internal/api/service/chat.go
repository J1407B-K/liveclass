package service

import (
	"context"
	"errors"
	"liveclass/idl/kitex_gen/chat"
	"liveclass/internal/api/code"
	"liveclass/internal/api/global"
	"liveclass/internal/api/model"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

func GetHistory(c context.Context, ctx *app.RequestContext) {
	uid := ctx.GetInt64("userid")
	if uid == 0 {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}
	lid := ctx.Query("lesson_id")
	if lid == "" {
		lid = ctx.PostForm("lesson_id")
	}
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.H{"code": http.StatusBadRequest, "message": err.Error()})
		return
	}
	request := &chat.GetHistoryReq{LessonId: ilid, Userid: uid}
	if cursor := ctx.Query("cursor"); cursor != "" {
		request.Cursor = &cursor
	}
	if rawLimit := ctx.Query("limit"); rawLimit != "" {
		parsed, parseErr := strconv.ParseInt(rawLimit, 10, 32)
		if parseErr != nil || parsed <= 0 {
			ctx.JSON(http.StatusBadRequest, utils.H{"code": http.StatusBadRequest, "message": "invalid limit"})
			return
		}
		limit := int32(parsed)
		request.Limit = &limit
	}
	resp, err := global.Clients.ChatClient.GetHistory(c, request)
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
			Data: utils.H{
				"messages":    resp.Resp.Data.ChatInfo.Message,
				"next_cursor": resp.GetNextCursor(),
			},
		},
	})
}

func DelHistory(c context.Context, ctx *app.RequestContext) {
	uid := ctx.GetInt64("userid")
	if uid == 0 {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.H{"code": http.StatusBadRequest, "message": err.Error()})
		return
	}
	resp, err := global.Clients.ChatClient.DelHistory(c, &chat.DelHistoryReq{LessonId: ilid, Userid: uid})
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
