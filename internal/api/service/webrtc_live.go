package service

import (
	"context"
	"errors"
	"io"
	"liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/internal/api/code"
	"liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

func Broadcast(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	offer := ctx.PostForm("b64offer")

	resp, err := global.Clients.Webrtc_liveClient.Broadcast(c, &webrtc_live.BroadcastReq{
		Userid:   uid,
		LessonId: ilid,
		B64offer: offer,
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

func View(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	offer := ctx.PostForm("b64offer")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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

	resp, err := global.Clients.Webrtc_liveClient.View(c, &webrtc_live.ViewReq{
		Userid:   uid,
		LessonId: ilid,
		B64offer: offer,
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

func CreateLesson(c context.Context, ctx *app.RequestContext) {
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

	ln := ctx.PostForm("lesson_name")
	desc := ctx.PostForm("desc")
	resp, err := global.Clients.Webrtc_liveClient.CreateLesson(c, &webrtc_live.CreateLessonReq{
		Userid:      uid,
		LessonName:  ln,
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

func DelLesson(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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

	resp, err := global.Clients.Webrtc_liveClient.DelLesson(c, &webrtc_live.DelLessonReq{
		Userid:   uid,
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

func ChangeUserToLesson_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	stuid := ctx.PostForm("userid")
	o := ctx.PostForm("options")

	istuid, err := strconv.ParseInt(stuid, 10, 64)
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
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.ChangeUserToLesson(c, &webrtc_live.ChangeUserToLessonReq{
		Userid:   uid,
		Lessonid: ilid,
		Stuid:    istuid,
		Options:  o,
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

func ChangeUserInLive_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	o := ctx.PostForm("options")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.ChangeUserInLive(c, &webrtc_live.ChangeUserInLiveReq{
		Userid:   uid,
		Lessonid: ilid,
		Options:  o,
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

func SelectLessonInfo_WebRTC(c context.Context, ctx *app.RequestContext) {
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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

	resp, err := global.Clients.Webrtc_liveClient.SelectLessonInfo(c, &webrtc_live.SelectLessonInfoReq{
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

func GetLessonInfo_WebRTC(c context.Context, ctx *app.RequestContext) {
	lessonname := ctx.PostForm("lesson_name")
	teacher := ctx.PostForm("teacher")

	resp, err := global.Clients.Webrtc_liveClient.GetLessonInfo(c, &webrtc_live.GetLessonInfoReq{
		LessonName: lessonname,
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

func CreateSignIn_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	duration := ctx.PostForm("duration")

	ilid, err := strconv.ParseInt(lid, 10, 64)
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

	iduration, err := strconv.Atoi(duration)
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
	resp, err := global.Clients.Webrtc_liveClient.CreateSignIn(c, &webrtc_live.CreateSignInReq{
		Userid:   uid,
		Lessonid: ilid,
		Duration: int64(iduration),
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

func SignIn_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.SignIn(c, &webrtc_live.SignInReq{
		Userid:   uid,
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

func SelectSignIn_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.SelectSignIn(c, &webrtc_live.SelectSignInReq{
		Userid:   uid,
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

func DelSignIn_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.DelSign(c, &webrtc_live.DelSignInReq{
		Userid:   uid,
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

func RollCallInRandom_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.RollCallInRandom(c, &webrtc_live.RollCallInRandomReq{
		Userid:   uid,
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

func RecordLesson_WebRTC(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	fileHeader, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"resp": model2.Response{
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
			"resp": model2.Response{
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
			"resp": model2.Response{
				Code: code.InternalError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}
	if len(dataBytes) == 0 {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"resp": model2.Response{
				Code: code.InternalError,
				Msg:  "没有任何数据需要写入",
				Data: "nil",
			},
		})
		return
	}

	resp, err := global.Clients.Webrtc_liveClient.RecordLesson(c, &webrtc_live.RecordLessonReq{
		Userid:   uid,
		Lessonid: ilid,
		Data:     dataBytes,
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

func SaveWhiteBoard(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	file := ctx.PostForm("file")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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

	resp, err := global.Clients.Webrtc_liveClient.SaveWhiteBoardJson(c, &webrtc_live.SaveWhiteBoardJsonReq{
		Userid:   uid,
		Lessonid: ilid,
		File:     file,
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

func GetWhiteBoard(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.GetWhiteBoardJson(c, &webrtc_live.GetWhiteBoardJsonReq{
		Userid:   uid,
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

func PublishMic(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	offer := ctx.PostForm("b64offer")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.PublishMic(c, &webrtc_live.PublishMicReq{
		Userid:   uid,
		Lessonid: ilid,
		B64offer: offer,
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

func RaiseHand(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.RaiseHand(c, &webrtc_live.RaiseHandReq{
		Userid:   uid,
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

func GetRaiseHand(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.Query("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.GetRaiseHand(c, &webrtc_live.GetRaiseHandReq{
		Userid:   uid,
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

func ApproveHand(c context.Context, ctx *app.RequestContext) {
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

	stuid := ctx.PostForm("stu_id")
	istuid, err := strconv.ParseInt(stuid, 10, 64)
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
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.ApproveHand(c, &webrtc_live.ApproveHandReq{
		Userid:   uid,
		Lessonid: ilid,
		Stuid:    istuid,
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

func ViewMic(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	b64off := ctx.PostForm("b64offer")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.ViewMic(c, &webrtc_live.ViewMicReq{
		Userid:   uid,
		Lessonid: ilid,
		B64offer: b64off,
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

func ListAllLessonRecord(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.ListAllLessonRecord(c, &webrtc_live.ListAllLessonRecordReq{
		Userid:   uid,
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

func GetLessonRecord(c context.Context, ctx *app.RequestContext) {
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

	lid := ctx.PostForm("lesson_id")
	key := ctx.PostForm("key")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	file, err := global.Clients.Webrtc_liveClient.GetLessonRecord(c, &webrtc_live.GetLessonRecordReq{
		Userid:   uid,
		Lessonid: ilid,
		Key:      key,
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
			Data: "success",
		},
		"file": file,
	})
}

func IsStuInLesson(c context.Context, ctx *app.RequestContext) {
	uid := ctx.GetInt64("userid")
	if uid == 0 {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
			},
		})
		return
	}
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
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
	resp, err := global.Clients.Webrtc_liveClient.IsStudentInLesson(c, &webrtc_live.IsStudentInLessonReq{
		Studentid: uid,
		Lessonid:  ilid,
	})
	if err != nil {
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
