package service

import (
	"context"
	"liveclass/idl/kitex_gen/quiz"
	"liveclass/internal/api/code"
	global2 "liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"
	"liveclass/internal/api/utils/cut"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/websocket"
)

func QuizConnection(c context.Context, ctx *app.RequestContext) {
	lessonid := ctx.Query("lesson_id")
	token := ctx.Query("token")
	claim, err := parse(token)

	ilid, err := strconv.ParseInt(lessonid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}
	uid, err := strconv.ParseInt(claim.UserId, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model2.Response{
				Code: code.AuthError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}
	err = global2.Upgrader.Upgrade(ctx, ansHandler(c, uid, ilid))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model2.Response{
				Code: code.UpgraderError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}
}

func ansHandler(c context.Context, userId, lessonid int64) websocket.HertzHandler {
	return func(conn *websocket.Conn) {
		global2.Mux.Lock()
		global2.WsConnsQuiz[conn] = userId
		global2.WsConnsQuizLesson[conn] = lessonid
		global2.Mux.Unlock()

		for {
			var ans model2.Answer
			if err := conn.ReadJSON(&ans); err != nil {
				global2.Mux.Lock()
				delete(global2.WsConnsQuiz, conn)
				delete(global2.WsConnsQuizLesson, conn)
				global2.Mux.Unlock()
				break
			}

			resp, err := global2.Clients.QuizClient.TorFAnswer(c, &quiz.TorFAnswerReq{
				QuestionId: ans.QuestionId,
				Userid:     userId,
				UserAnswer: ans.Answer,
			})
			if err != nil {
				global2.Mux.Lock()
				delete(global2.WsConnsQuiz, conn)
				delete(global2.WsConnsQuizLesson, conn)
				global2.Mux.Unlock()
				break
			}
			if resp.Resp.Msg != "" {
				teacherid, options := cut.SplitAnsResp(resp.Resp.Msg)
				itid, err := strconv.ParseInt(teacherid, 10, 64)
				if err != nil {
					break
				}
				err = broadcastToTeacher(itid, options)
				if err != nil {
					global2.Mux.Lock()
					delete(global2.WsConnsQuiz, conn)
					delete(global2.WsConnsQuizLesson, conn)
					global2.Mux.Unlock()
					break
				}
			}
		}
	}
}
