package service

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/websocket"
	"liveclass/idl/kitex_gen/quiz"
	"liveclass/internal/code"
	"liveclass/internal/global"
	"liveclass/internal/model"
	"liveclass/internal/utils/cut"
	"net/http"
)

func QuizConnection(c context.Context, ctx *app.RequestContext) {
	lessonid := ctx.Query("lesson_id")
	token := ctx.Query("token")
	claim, err := parse(token)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model.Response{
				Code: code.AuthError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}
	err = global.Upgrader.Upgrade(ctx, ansHandler(c, claim.UserId, lessonid))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"resp": model.Response{
				Code: code.UpgraderError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
		return
	}
}

func ansHandler(c context.Context, userId, lessonid string) websocket.HertzHandler {
	return func(conn *websocket.Conn) {
		global.Mux.Lock()
		global.WsConnsQuiz[conn] = userId
		global.WsConnsQuizLesson[conn] = lessonid
		global.Mux.Unlock()

		for {
			var ans model.Answer
			if err := conn.ReadJSON(&ans); err != nil {
				global.Mux.Lock()
				delete(global.WsConnsQuiz, conn)
				delete(global.WsConnsQuizLesson, conn)
				global.Mux.Unlock()
				break
			}

			resp, err := global.Clients.QuizClient.TorFAnswer(c, &quiz.TorFAnswerReq{
				QuestionId: ans.QuestionId,
				Userid:     userId,
				UserAnswer: ans.Answer,
			})
			if err != nil {
				global.Mux.Lock()
				delete(global.WsConnsQuiz, conn)
				delete(global.WsConnsQuizLesson, conn)
				global.Mux.Unlock()
				break
			}
			if resp.Resp.Data != "" {
				teacherid, options := cut.SplitAnsResp(resp.Resp.Data)
				err = broadcastToTeacher(teacherid, options)
				if err != nil {
					global.Mux.Lock()
					delete(global.WsConnsQuiz, conn)
					delete(global.WsConnsQuizLesson, conn)
					global.Mux.Unlock()
					break
				}
			}
		}
	}
}
