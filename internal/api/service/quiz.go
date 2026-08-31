package service

import (
	"context"
	"encoding/json"
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
		Userid:     uid,
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

	if err = publishQuizQuestion(c, question, resp); err != nil {
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

func publishQuizQuestion(c context.Context, question model2.Question, resp *quiz.CreateQuestionResp) error {
	quizID := int64(0)
	if resp != nil && resp.Resp != nil && resp.Resp.Data != nil && resp.Resp.Data.QuizInfo != nil {
		quizID = resp.Resp.Data.QuizInfo.QuizID
	}

	msg := utils.H{
		"type":        "quiz_question",
		"lesson_id":   question.LessonId,
		"question_id": quizID,
		"content":     question.Content,
		"options":     question.Options,
	}

	if global.DBManager == nil || global.DBManager.RDB == nil {
		return broadcastQuizToLesson(question.LessonId, msg)
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return global.DBManager.RDB.Publish(c, "quiz:broadcast", payload).Err()
}

func DelQuestion(c context.Context, ctx *app.RequestContext) {
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
