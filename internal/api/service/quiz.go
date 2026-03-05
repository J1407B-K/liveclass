package service

import (
	"context"
	"errors"
	"liveclass/idl/kitex_gen/quiz"
	"liveclass/internal/api/code"
	"liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
)

func CreateQuestion(c context.Context, ctx *app.RequestContext) {
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

	var question model2.Question
	err := ctx.BindJSON(&question)
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

	resp, err := global.Clients.QuizClient.CreateQuestion(c, &quiz.CreateQuestionReq{
		LessonId:   question.LessonId,
		Userid:     userid,
		Content:    question.Content,
		OptionsNum: int32(question.OptionNums),
		Options:    question.Options,
		Answer:     question.Answer,
		Duration:   question.Duration,
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

	err = broadcastToLesson(question.LessonId, utils.H{
		"Content": question.Content,
		"Options": question.Options,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{
				Code: code.BroadCastError,
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

func DelQuestion(c context.Context, ctx *app.RequestContext) {
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

	uid := data.(*model2.User).UserId
	qid := ctx.PostForm("question_id")
	iqid, err := strconv.ParseInt(qid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}

	resp, err := global.Clients.QuizClient.DelQuestion(c, &quiz.DelQuestionReq{
		Userid:     uid,
		QuestionId: iqid,
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

func GetAllLessonQuiz(c context.Context, ctx *app.RequestContext) {
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

	uid := data.(*model2.User).UserId
	lid := ctx.PostForm("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}

	resp, err := global.Clients.QuizClient.GetAllLessonQuiz(c, &quiz.GetAllLessonQuizReq{
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
