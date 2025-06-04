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
	"strconv"
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
// TODO
func SelectLessonInfo(c context.Context, ctx *app.RequestContext) {
	lessonid := ctx.PostForm("lesson_id")
	teacher := ctx.PostForm("teacher")

	resp, err := global.Clients.LiveClient.SelectLessonInfo(c, &live.SelectLessonInfoReq{
		Lessonid: lessonid,
		Teacher:  teacher,
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

	lid := ctx.PostForm("lesson_id")
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
		Lessonid: lid,
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

func ChangeUserToLesson(c context.Context, ctx *app.RequestContext) {
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
	ln := ctx.PostForm("lesson_name")
	t := ctx.PostForm("teacher")
	o := ctx.PostForm("option")
	stuId := ctx.PostForm("student_id")

	resp, err := global.Clients.LiveClient.ChangeUserToLesson(c, &live.ChangeUserToLessonReq{
		Userid:     userid,
		Lessonname: ln,
		Teacher:    t,
		Option:     o,
		Studentid:  stuId,
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

func IsStudentInLesson(c context.Context, ctx *app.RequestContext) {
	sid := ctx.PostForm("student_id")
	lid := ctx.PostForm("lesson_id")

	resp, err := global.Clients.LiveClient.IsStudentInLesson(c, &live.IsStudentInLessonReq{
		Studentid: sid,
		Lessonid:  lid,
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

func RecordLesson(c context.Context, ctx *app.RequestContext) {
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
	streamurl := ctx.PostForm("stream_url")
	lessonId := ctx.PostForm("lesson_id")
	duration := ctx.PostForm("duration")

	d, err := strconv.Atoi(duration)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"resp": model.Response{
				Code: code.BadRequest,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}

	resp, err := global.Clients.LiveClient.RecordLesson(c, &live.RecordLessonReq{
		Userid:    userid,
		StreamURL: streamurl,
		LessonId:  lessonId,
		Duration:  int32(d),
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

func CreateSignIn(c context.Context, ctx *app.RequestContext) {
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
	lid := ctx.PostForm("lesson_id")
	resp, err := global.Clients.LiveClient.CreateSignIn(c, &live.CreateSignInReq{
		Userid:   userid,
		Lessonid: lid,
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

func SignIn(c context.Context, ctx *app.RequestContext) {
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
	lid := ctx.PostForm("lesson_id")

	resp, err := global.Clients.LiveClient.SignIn(c, &live.SignInReq{
		Userid:   userid,
		Lessonid: lid,
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

func SelectSignIn(c context.Context, ctx *app.RequestContext) {
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
	lid := ctx.PostForm("lesson_id")

	resp, err := global.Clients.LiveClient.SelectSignIn(c, &live.SelectSignInReq{
		Userid:   userid,
		Lessonid: lid,
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

func DelSignIn(c context.Context, ctx *app.RequestContext) {
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
	lid := ctx.PostForm("lesson_id")

	resp, err := global.Clients.LiveClient.DelSign(c, &live.DelSignInReq{
		Userid:   userid,
		Lessonid: lid,
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

func RollCallInRandom(c context.Context, ctx *app.RequestContext) {
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
	lid := ctx.PostForm("lesson_id")

	resp, err := global.Clients.LiveClient.RollCallInRandom(c, &live.RollCallInRandomReq{
		Userid:   userid,
		LessonId: lid,
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
