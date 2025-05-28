package service

import (
	"context"
	"errors"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"io"
	"liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/internal/code"
	"liveclass/internal/global"
	"liveclass/internal/model"
	"net/http"
)

func Broadcast(c context.Context, ctx *app.RequestContext) {
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
	lid := ctx.PostForm("lesson_id")
	offer := ctx.PostForm("b64offer")

	resp, err := global.Clients.Webrtc_liveClient.Broadcast(c, &webrtc_live.BroadcastReq{
		Userid:   userid,
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
	lid := ctx.PostForm("lesson_id")
	offer := ctx.PostForm("b64offer")

	resp, err := global.Clients.Webrtc_liveClient.View(c, &webrtc_live.ViewReq{
		Userid:   userid,
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

func CreateLesson(c context.Context, ctx *app.RequestContext) {
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

	ln := ctx.PostForm("lesson_name")
	desc := ctx.PostForm("desc")
	resp, err := global.Clients.Webrtc_liveClient.CreateLesson(c, &webrtc_live.CreateLessonReq{
		Userid:      userid,
		LessonName:  ln,
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

func DelLesson(c context.Context, ctx *app.RequestContext) {
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

	resp, err := global.Clients.Webrtc_liveClient.DelLesson(c, &webrtc_live.DelLessonReq{
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

func ChangeUserToLesson_WebRTC(c context.Context, ctx *app.RequestContext) {
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
	o := ctx.PostForm("options")
	resp, err := global.Clients.Webrtc_liveClient.ChangeUserToLesson(c, &webrtc_live.ChangeUserToLessonReq{
		Userid:   userid,
		Lessonid: lid,
		Options:  o,
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

func ChangeUserInLive_WebRTC(c context.Context, ctx *app.RequestContext) {
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
	o := ctx.PostForm("options")
	resp, err := global.Clients.Webrtc_liveClient.ChangeUserInLive(c, &webrtc_live.ChangeUserInLiveReq{
		Userid:   userid,
		Lessonid: lid,
		Options:  o,
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

func SelectLessonInfo_WebRTC(c context.Context, ctx *app.RequestContext) {
	lessonid := ctx.PostForm("lesson_id")

	resp, err := global.Clients.Webrtc_liveClient.SelectLessonInfo(c, &webrtc_live.SelectLessonInfoReq{
		Lessonid: lessonid,
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

func GetLessonInfo_WebRTC(c context.Context, ctx *app.RequestContext) {
	lessonname := ctx.PostForm("lesson_name")
	teacher := ctx.PostForm("teacher")

	resp, err := global.Clients.Webrtc_liveClient.GetLessonInfo(c, &webrtc_live.GetLessonInfoReq{
		LessonName: lessonname,
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

func CreateSignIn_WebRTC(c context.Context, ctx *app.RequestContext) {
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
	resp, err := global.Clients.Webrtc_liveClient.CreateSignIn(c, &webrtc_live.CreateSignInReq{
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

func SignIn_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	resp, err := global.Clients.Webrtc_liveClient.SignIn(c, &webrtc_live.SignInReq{
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

func SelectSignIn_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	resp, err := global.Clients.Webrtc_liveClient.SelectSignIn(c, &webrtc_live.SelectSignInReq{
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

func DelSignIn_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	resp, err := global.Clients.Webrtc_liveClient.DelSign(c, &webrtc_live.DelSignInReq{
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

func RollCallInRandom_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	resp, err := global.Clients.Webrtc_liveClient.RollCallInRandom(c, &webrtc_live.RollCallInRandomReq{
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

func RecordLesson_WebRTC(c context.Context, ctx *app.RequestContext) {
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
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"resp": model.Response{
				Code: code.InternalError,
				Msg:  "缺少录制文件",
				Data: "nil",
			},
		})
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model.Response{
				Code: code.InternalError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}
	defer f.Close()

	dataBytes, err := io.ReadAll(f)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model.Response{
				Code: code.InternalError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}
	if len(dataBytes) == 0 {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"resp": model.Response{
				Code: code.InternalError,
				Msg:  "没有任何数据需要写入",
				Data: "nil",
			},
		})
		return
	}

	resp, err := global.Clients.Webrtc_liveClient.RecordLesson(c, &webrtc_live.RecordLessonReq{
		Userid:   userid,
		Lessonid: lid,
		Data:     dataBytes,
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

func SaveWhiteBoard(c context.Context, ctx *app.RequestContext) {
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
	file := ctx.PostForm("file")

	resp, err := global.Clients.Webrtc_liveClient.SaveWhiteBoardJson(c, &webrtc_live.SaveWhiteBoardJsonReq{
		Userid:   userid,
		Lessonid: lid,
		File:     file,
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

func GetWhiteBoard(c context.Context, ctx *app.RequestContext) {
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

	resp, err := global.Clients.Webrtc_liveClient.GetWhiteBoardJson(c, &webrtc_live.GetWhiteBoardJsonReq{
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

func PublishMic(c context.Context, ctx *app.RequestContext) {
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
	offer := ctx.PostForm("b64offer")

	resp, err := global.Clients.Webrtc_liveClient.PublishMic(c, &webrtc_live.PublishMicReq{
		Userid:   userid,
		Lessonid: lid,
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
