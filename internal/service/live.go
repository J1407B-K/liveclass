package service

import (
	"context"
	"errors"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"liveclass/idl/kitex_gen/live"
	"liveclass/internal/code"
	"liveclass/internal/global"
	"liveclass/internal/model"
	"log"
	"net/http"
)

func CreateLive(c context.Context, ctx *app.RequestContext) {
	//鉴权获得userid
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"code": code.AuthError,
			"msg":  errors.New("无法获取userid"),
			"data": "nil",
		})
		return
	}

	userid := data.(*model.User).UserId

	var lesson model.Lesson

	err := ctx.BindJSON(&lesson)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
			Data: "nil",
		})
		return
	}

	log.Println(userid)
	resp, err := global.Clients.LiveClient.CreateLive(c, &live.CreateLiveReq{
		Livename:    lesson.Name,
		Userid:      userid,
		Description: lesson.Description,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"code": code.RPCError,
			"msg":  err,
			"data": "nil",
		})
		return
	}

	ctx.JSON(http.StatusOK, utils.H{
		"code": 0,
		"msg":  "ok",
		"data": resp,
	})
}

func CloseLive(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"code": code.AuthError,
			"msg":  errors.New("无法获取userid"),
			"data": "nil",
		})
		return
	}
	userid := data.(*model.User).UserId

	livename := ctx.PostForm("livename")

	resp, err := global.Clients.LiveClient.CloseLive(c, &live.CloseLiveReq{
		Livename: livename,
		Userid:   userid,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, model.Response{
			Code: code.RPCError,
			Msg:  err.Error(),
			Data: "nil",
		})
		return
	}
	ctx.JSON(http.StatusOK, model.Response{
		Code: 0,
		Msg:  "ok",
		Data: resp.Resp.Data,
	})
}
