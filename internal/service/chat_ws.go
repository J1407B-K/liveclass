package service

import (
	"context"
	"encoding/json"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/websocket"
	"liveclass/idl/kitex_gen/chat"
	"liveclass/idl/kitex_gen/live"
	"liveclass/internal/code"
	"liveclass/internal/global"
	"liveclass/internal/model"
	"log"
	"net/http"
)

func ChatConnections(c context.Context, ctx *app.RequestContext) {
	token := ctx.Query("token")
	lessonId := ctx.Query("lesson_id")

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
	err = global.Upgrader.Upgrade(ctx, chatHandler(c, claim.UserId, lessonId))
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

func chatHandler(c context.Context, userId, lessonId string) websocket.HertzHandler {
	return func(conn *websocket.Conn) {
		global.Mux.Lock()
		global.WsConnsChat[conn] = lessonId
		global.Mux.Unlock()

		for {
			resp, err := global.Clients.LiveClient.IsStudentInLesson(c, &live.IsStudentInLessonReq{Lessonid: lessonId, Studentid: userId})
			if err != nil {
				global.Mux.Lock()
				delete(global.WsConnsChat, conn)
				global.Mux.Unlock()
				break
			}

			if resp.Resp.Data == "not_in" {
				conn.WriteMessage(websocket.TextMessage, []byte("不是该课程学生！"))
				global.Mux.Lock()
				delete(global.WsConnsChat, conn)
				global.Mux.Unlock()
				break
			}

			messageType, message, err := conn.ReadMessage()
			if err != nil {
				global.Mux.Lock()
				delete(global.WsConnsChat, conn)
				global.Mux.Unlock()
				break
			}
			// 只处理文本消息
			if messageType != websocket.TextMessage {
				log.Println("非文本消息，忽略")
				continue
			}

			var msgJson model.Message
			if err := json.Unmarshal(message, &msgJson); err != nil {
				log.Println("Unmarshal message error:", err)
				global.Mux.Lock()
				delete(global.WsConnsChat, conn)
				global.Mux.Unlock()
				break
			}

			_, err = global.Clients.ChatClient.LiveChat(c, &chat.LiveChatReq{Lessonid: msgJson.LessonID, Userid: userId, Message: msgJson.Content})
			if err != nil {
				global.Mux.Lock()
				delete(global.WsConnsChat, conn)
				global.Mux.Unlock()
				break
			}
		}
	}
}
