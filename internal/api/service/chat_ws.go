package service

import (
	"context"
	"encoding/json"
	"fmt"
	"liveclass/idl/kitex_gen/chat"
	"liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/internal/api/code"
	global2 "liveclass/internal/api/global"
	model2 "liveclass/internal/api/model"
	"liveclass/internal/api/utils/jwt"
	"liveclass/internal/api/utils/ratelimit"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/common/utils"
	"github.com/go-redis/redis/v8"
	"github.com/hertz-contrib/websocket"
	"github.com/segmentio/kafka-go"
)

func ChatConnections(c context.Context, ctx *app.RequestContext) {
	token := ctx.Query("token")
	if token == "" {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"code": code.AuthError,
			"msg":  "missing token",
		})
		return
	}

	uid, err := jwt.ParseAccessToken(token)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, utils.H{
			"code": code.AuthError,
			"msg":  err.Error(),
		})
		return
	}

	lid := ctx.Query("lesson_id")
	ilid, err := strconv.ParseInt(lid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, utils.H{
			"code": code.BadRequest,
			"msg":  err.Error(),
		})
		return
	}

	if err = global2.Upgrader.Upgrade(ctx, chatHandler(c, uid, ilid)); err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"code": code.UpgraderError,
			"msg":  err.Error(),
		})
		return
	}
}

func RunChatRedisSubscriber(ctx context.Context, rdb *redis.Client) error {
	pubsub := rdb.Subscribe(ctx, "chat:broadcast")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-ch:
			var chatMsg model2.ShowMessage
			if err := json.Unmarshal([]byte(msg.Payload), &chatMsg); err != nil {
				log.Printf("chat redis unmarshal failed: err=%v", err)
				continue
			}
			broadcastChatToLesson(chatMsg.LessonID, chatMsg)
		}
	}
}

func RunChatConsumer(ctx context.Context, reader *kafka.Reader) error {
	defer reader.Close()

	for {
		m, err := reader.FetchMessage(ctx)
		if err != nil {
			return err
		}
		var msg model2.ShowMessage
		if err = json.Unmarshal(m.Value, &msg); err != nil {
			log.Printf("chat consumer unmarshal failed: offset=%d err=%v", m.Offset, err)
			_ = reader.CommitMessages(ctx, m)
			continue
		}

		broadcastChatToLesson(msg.LessonID, msg)

		if err = reader.CommitMessages(ctx, m); err != nil {
			log.Printf("chat consumer commit failed: offset=%d err=%v", m.Offset, err)
		}
	}
}

func chatHandler(c context.Context, userId, lessonId int64) websocket.HertzHandler {
	return func(conn *websocket.Conn) {
		resp, err := global2.Clients.Webrtc_liveClient.IsStudentInLesson(
			c,
			&webrtc_live.IsStudentInLessonReq{
				Lessonid:  lessonId,
				Studentid: userId,
			},
		)
		if err != nil {
			_ = conn.Close()
			return
		}
		if resp == nil || resp.Resp == nil || resp.Resp.Msg != "exist" {
			_ = conn.WriteMessage(websocket.TextMessage, []byte("不是该课程成员！"))
			_ = conn.Close()
			return
		}

		addChatConn(lessonId, conn)
		defer removeChatConn(lessonId, conn)
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				break
			}
			if messageType != websocket.TextMessage {
				continue
			}

			var msgJson model2.Message
			if err = json.Unmarshal(message, &msgJson); err != nil {
				log.Println("unmarshal chat message error:", err)
				continue
			}

			allowed, err := ratelimit.AllowRedis(
				c,
				global2.DBManager.RDB,
				fmt.Sprintf("rl:chat:send:%d", userId),
				20,
				40,
				1,
				time.Minute,
			)
			if err != nil {
				log.Println("chat limiter error:", err)
				continue
			}
			if !allowed {
				_ = conn.WriteMessage(websocket.TextMessage, []byte("发送过于频繁，请稍后再试"))
				continue
			}

			_, err = global2.Clients.ChatClient.LiveChat(c, &chat.LiveChatReq{
				Lessonid: lessonId,
				Userid:   userId,
				Message:  msgJson.Content,
			})
			if err != nil {
				log.Println("chat rpc error:", err)
				continue
			}
		}
	}
}

func addChatConn(lessonID int64, conn *websocket.Conn) {
	global2.Mux.Lock()
	defer global2.Mux.Unlock()

	if global2.ChatLessonConns[lessonID] == nil {
		global2.ChatLessonConns[lessonID] = make(map[*websocket.Conn]struct{})
	}
	global2.ChatLessonConns[lessonID][conn] = struct{}{}
}

func removeChatConn(lessonID int64, conn *websocket.Conn) {
	global2.Mux.Lock()
	defer global2.Mux.Unlock()

	if conns, ok := global2.ChatLessonConns[lessonID]; ok {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(global2.ChatLessonConns, lessonID)
		}
	}
}

func broadcastChatToLesson(lessonID int64, msg interface{}) {
	global2.Mux.RLock()
	conns := global2.ChatLessonConns[lessonID]
	targets := make([]*websocket.Conn, 0, len(conns))
	for conn := range conns {
		targets = append(targets, conn)
	}
	global2.Mux.RUnlock()

	for _, conn := range targets {
		if err := conn.WriteJSON(msg); err != nil {
			_ = conn.Close()
			removeChatConn(lessonID, conn)
		}
	}
}
