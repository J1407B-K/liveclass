package service

import (
	"context"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	jwtt "github.com/golang-jwt/jwt/v4"
	"github.com/hertz-contrib/websocket"
	"liveclass/idl/kitex_gen/quiz"
	"liveclass/internal/code"
	"liveclass/internal/global"
	"liveclass/internal/model"
	"net/http"
)

var jwtSecret = []byte("by_kq")

type Claims struct {
	UserId string `json:"userId"`
	jwtt.RegisteredClaims
}

func broadcast(lessonId string, message interface{}) error {
	global.Mux.Lock()
	defer global.Mux.Unlock()
	for conn, l := range global.WsConns {
		if l == lessonId {
			err := conn.WriteJSON(message)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func ListenConnection(c context.Context, ctx *app.RequestContext) {
	l := ctx.Query("lesson_id")
	token := ctx.Query("token")
	cliams, err := parse(token)

	if err != nil {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"resp": model.Response{
				Code: code.AuthError,
				Msg:  err.Error(),
				Data: "nil",
			},
		})
	}
	err = global.Upgrader.Upgrade(ctx, addHandler(c, cliams.UserId, l))
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

func addHandler(c context.Context, userId, lessonId string) websocket.HertzHandler {
	return func(conn *websocket.Conn) {
		global.Mux.Lock()
		global.WsConns[conn] = lessonId
		global.Mux.Unlock()

		for {
			var ans model.Answer
			if err := conn.ReadJSON(&ans); err != nil {
				global.Mux.Lock()
				delete(global.WsConns, conn)
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
				delete(global.WsConns, conn)
				global.Mux.Unlock()
				break
			}

			err = broadcast(lessonId, resp.Resp.Data)
			if err != nil {
				global.Mux.Lock()
				delete(global.WsConns, conn)
				global.Mux.Unlock()
				break
			}
		}
	}
}

func parse(tokenstr string) (*Claims, error) {
	token, err := jwtt.ParseWithClaims(tokenstr, &Claims{}, func(token *jwtt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, err
}
