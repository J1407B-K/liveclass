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
	"net/http"
)

func CreateLive(c context.Context, ctx *app.RequestContext) {
	//鉴权获得userid
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})

		return
	}

	userid := data.(*model.User).UserId

	lessonName := ctx.PostForm("lesson_name")
	desc := ctx.PostForm("description")

	resp, err := global.Clients.LiveClient.CreateLive(c, &live.CreateLiveReq{
		Livename:    lessonName,
		Userid:      userid,
		Description: desc,
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

func CloseLive(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
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

// 查询在线人数
func SelectLessonInfo(c context.Context, ctx *app.RequestContext) {
	lessonname := ctx.PostForm("lesson_name")
	teacher := ctx.PostForm("teacher")

	resp, err := global.Clients.LiveClient.SelectLessonInfo(c, &live.SelectLessonInfoReq{
		Lessonname: lessonname,
		Teacher:    teacher,
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

func ChangeUserInLive(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	userid := data.(*model.User).UserId

	livename := ctx.PostForm("livename")
	options := ctx.PostForm("options")
	if options != "add" && options != "del" {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"resp": model.Response{
				Code: code.BadRequest,
				Msg:  errors.New("参数错误").Error(),
			},
		})
		return
	}
	resp, err := global.Clients.LiveClient.ChangeUserInLive(c, &live.ChangeUserInLiveReq{
		Livename: livename,
		Userid:   userid,
		Options:  options,
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

// 查询Mysql存储
func GetLessonInfo(c context.Context, ctx *app.RequestContext) {
	lessonname := ctx.PostForm("lesson_name")
	teacher := ctx.PostForm("teacher")

	resp, err := global.Clients.LiveClient.GetLessonInfo(c, &live.GetLessonInfoReq{
		Lessonname: lessonname,
		Teacher:    teacher,
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
