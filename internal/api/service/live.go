package service

import (
	"context"
	"errors"
	"liveclass/idl/kitex_gen/live"
	"liveclass/internal/api/code"
	"liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

func CreateLive(c context.Context, ctx *app.RequestContext) {
	//鉴权获得userid
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	userid := data.(*model2.User).UserId

	lessonName := ctx.PostForm("lesson_name")
	desc := ctx.PostForm("description")

	resp, err := global.Clients.LiveClient.CreateLive(c, &live.CreateLiveReq{
		Livename:    lessonName,
		Userid:      userid,
		Description: desc,
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

func CloseLive(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}
	userid := data.(*model2.User).UserId

	livename := ctx.PostForm("livename")

	resp, err := global.Clients.LiveClient.CloseLive(c, &live.CloseLiveReq{
		Livename: livename,
		Userid:   userid,
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

// 查询在线人数
// TODO
func SelectLessonInfo(c context.Context, ctx *app.RequestContext) {
	lid := ctx.PostForm("lesson_id")
	teacher := ctx.PostForm("teacher")

	ilid, err := strconv.ParseInt(lid, 10, 64)

	resp, err := global.Clients.LiveClient.SelectLessonInfo(c, &live.SelectLessonInfoReq{
		Lessonid: ilid,
		Teacher:  teacher,
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

func ChangeUserInLive(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	userid := data.(*model2.User).UserId

	lid := ctx.PostForm("lesson_id")
	options := ctx.PostForm("options")
	if options != "add" && options != "del" {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"resp": model2.Response{
				Code: code.BadRequest,
				Msg:  errors.New("参数错误").Error(),
			},
		})
		return
	}
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{
				Code: code.BadRequest,
				Msg:  err.Error(),
			},
		})
	}
	resp, err := global.Clients.LiveClient.ChangeUserInLive(c, &live.ChangeUserInLiveReq{
		Lessonid: ilid,
		Userid:   userid,
		Options:  options,
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

func ChangeUserToLesson(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	userid := data.(*model2.User).UserId
	ln := ctx.PostForm("lesson_name")
	t := ctx.PostForm("teacher")
	o := ctx.PostForm("option")
	stuId := ctx.PostForm("student_id")
	istuId, err := strconv.ParseInt(stuId, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}

	resp, err := global.Clients.LiveClient.ChangeUserToLesson(c, &live.ChangeUserToLessonReq{
		Userid:     userid,
		Lessonname: ln,
		Teacher:    t,
		Option:     o,
		Studentid:  istuId,
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

func IsStudentInLesson(c context.Context, ctx *app.RequestContext) {
	sid := ctx.PostForm("student_id")
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}
	isid, err := strconv.ParseInt(sid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}

	resp, err := global.Clients.LiveClient.IsStudentInLesson(c, &live.IsStudentInLessonReq{
		Studentid: isid,
		Lessonid:  ilid,
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

func RecordLesson(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	userid := data.(*model2.User).UserId
	streamurl := ctx.PostForm("stream_url")
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}
	duration := ctx.PostForm("duration")

	d, err := strconv.Atoi(duration)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"resp": model2.Response{
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
		LessonId:  ilid,
		Duration:  int32(d),
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

func CreateSignIn(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	userid := data.(*model2.User).UserId
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}
	resp, err := global.Clients.LiveClient.CreateSignIn(c, &live.CreateSignInReq{
		Userid:   userid,
		Lessonid: ilid,
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

func SignIn(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	userid := data.(*model2.User).UserId
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}
	resp, err := global.Clients.LiveClient.SignIn(c, &live.SignInReq{
		Userid:   userid,
		Lessonid: ilid,
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

func SelectSignIn(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	userid := data.(*model2.User).UserId
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}

	resp, err := global.Clients.LiveClient.SelectSignIn(c, &live.SelectSignInReq{
		Userid:   userid,
		Lessonid: ilid,
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

func DelSignIn(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	userid := data.(*model2.User).UserId
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}

	resp, err := global.Clients.LiveClient.DelSign(c, &live.DelSignInReq{
		Userid:   userid,
		Lessonid: ilid,
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

func RollCallInRandom(c context.Context, ctx *app.RequestContext) {
	data, e := ctx.Get("userid")
	if !e {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  errors.New("无法获取userid").Error(),
				Data: "nil",
			},
		})
		return
	}

	userid := data.(*model2.User).UserId
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}

	resp, err := global.Clients.LiveClient.RollCallInRandom(c, &live.RollCallInRandomReq{
		Userid:   userid,
		LessonId: ilid,
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
