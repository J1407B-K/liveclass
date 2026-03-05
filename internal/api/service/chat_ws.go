package service

import (
	"context"
	"encoding/json"
	"liveclass/idl/kitex_gen/chat"
	"liveclass/idl/kitex_gen/live"
	"liveclass/internal/api/code"
	global2 "liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"
	"log"
	"net/http"
	"strconv"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/hertz-contrib/websocket"
)

func ChatConnections(c context.Context, ctx *app.RequestContext) {
	token := ctx.Query("token")
	lid := ctx.Query("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"code": code.BadRequest,
			"msg":  err.Error(),
		})
	}
	claim, err := parse(token)
	uid, err := strconv.ParseInt(claim.UserId, 10, 64)

	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"code": code.BadRequest,
			"msg":  err.Error(),
		})
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
	err = global2.Upgrader.Upgrade(ctx, chatHandler(c, uid, ilid))
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

func chatHandler(c context.Context, userId, lessonId int64) websocket.HertzHandler {
	return func(conn *websocket.Conn) {
		global2.Mux.Lock()
		global2.WsConnsChat[conn] = lessonId
		global2.Mux.Unlock()

		for {
			resp, err := global2.Clients.LiveClient.IsStudentInLesson(c, &live.IsStudentInLessonReq{Lessonid: lessonId, Studentid: userId})
			if err != nil {
				global2.Mux.Lock()
				delete(global2.WsConnsChat, conn)
				global2.Mux.Unlock()
				break
			}

			if resp.Resp.Msg == "not_in" {
				conn.WriteMessage(websocket.TextMessage, []byte("不是该课程学生！"))
				global2.Mux.Lock()
				delete(global2.WsConnsChat, conn)
				global2.Mux.Unlock()
				break
			}

			messageType, message, err := conn.ReadMessage()
			if err != nil {
				global2.Mux.Lock()
				delete(global2.WsConnsChat, conn)
				global2.Mux.Unlock()
				break
			}
			// 只处理文本消息
			if messageType != websocket.TextMessage {
				log.Println("非文本消息，忽略")
				continue
			}

			var msgJson model2.Message
			if err := json.Unmarshal(message, &msgJson); err != nil {
				log.Println("Unmarshal message error:", err)
				global2.Mux.Lock()
				delete(global2.WsConnsChat, conn)
				global2.Mux.Unlock()
				break
			}

			_, err = global2.Clients.ChatClient.LiveChat(c, &chat.LiveChatReq{Lessonid: msgJson.LessonID, Userid: userId, Message: msgJson.Content})
			if err != nil {
				global2.Mux.Lock()
				delete(global2.WsConnsChat, conn)
				global2.Mux.Unlock()
				break
			}
		}
	}
}
