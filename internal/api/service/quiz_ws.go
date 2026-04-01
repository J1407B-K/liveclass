package service

import (
	"context"
	"encoding/json"
	"fmt"
	"liveclass/idl/kitex_gen/quiz"
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
)

func QuizConnection(c context.Context, ctx *app.RequestContext) {
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

	lessonid := ctx.Query("lesson_id")
	ilid, err := strconv.ParseInt(lessonid, 10, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, model2.Response{
			Code: code.BadRequest,
			Msg:  err.Error(),
		})
		return
	}

	if err = global2.Upgrader.Upgrade(ctx, ansHandler(c, uid, ilid)); err != nil {
		ctx.JSON(http.StatusInternalServerError, utils.H{
			"code": code.UpgraderError,
			"msg":  err.Error(),
		})
		return
	}
}

func ansHandler(c context.Context, userId, lessonid int64) websocket.HertzHandler {
	return func(conn *websocket.Conn) {
		global2.Mux.Lock()
		global2.WsConnsQuiz[conn] = &model2.QuizConnMeta{
			LessonID: lessonid,
			UserID:   userId,
		}
		global2.Mux.Unlock()

		defer func() {
			global2.Mux.Lock()
			delete(global2.WsConnsQuiz, conn)
			global2.Mux.Unlock()
			_ = conn.Close()
		}()

		for {
			var ans model2.Answer
			if err := conn.ReadJSON(&ans); err != nil {
				break
			}

			allowed, err := ratelimit.AllowRedis(
				c,
				global2.DBManager.RDB,
				fmt.Sprintf("rl:quiz:answer:%d", userId),
				10,
				20,
				1,
				time.Minute,
			)
			if err != nil {
				_ = conn.WriteJSON(map[string]any{
					"type": "quiz_error",
					"msg":  "限流错误，请稍后再试",
				})
				continue
			}
			if !allowed {
				_ = conn.WriteJSON(map[string]any{
					"type": "quiz_error",
					"msg":  "提交过于频繁，请稍后再试",
				})
				continue
			}

			resp, err := global2.Clients.QuizClient.TorFAnswer(c, &quiz.TorFAnswerReq{
				QuestionId: ans.QuestionId,
				Userid:     userId,
				UserAnswer: ans.Answer,
			})
			if err != nil {
				_ = conn.WriteJSON(map[string]any{
					"type": "quiz_error",
					"msg":  err.Error(),
				})
				continue
			}

			if resp == nil || resp.Resp == nil {
				_ = conn.WriteJSON(map[string]any{
					"type": "quiz_error",
					"msg":  "empty response",
				})
				continue
			}

			if resp.Resp.Code != 0 {
				_ = conn.WriteJSON(map[string]any{
					"type": "quiz_error",
					"msg":  resp.Resp.Msg,
				})
				continue
			}

			if resp.Resp.Data == nil || resp.Resp.Data.QuizInfo == nil {
				continue
			}

			if err = broadcastToTeacher(resp.Resp.Data.QuizInfo.TeacherID, map[string]any{
				"type":    "quiz_stats",
				"quiz_id": resp.Resp.Data.QuizInfo.QuizID,
				"stats":   resp.Resp.Data.QuizInfo.Stats,
			}); err != nil {
				break
			}
		}
	}
}

func broadcastToTeacher(userid int64, message interface{}) error {
	global2.Mux.RLock()
	targets := make([]*websocket.Conn, 0)
	for conn, meta := range global2.WsConnsQuiz {
		if meta.UserID == userid {
			targets = append(targets, conn)
		}
	}
	global2.Mux.RUnlock()

	for _, conn := range targets {
		if err := conn.WriteJSON(message); err != nil {
			_ = conn.Close()
		}
	}
	return nil
}

func broadcastQuizToLesson(lessonid int64, quiz interface{}) error {
	global2.Mux.RLock()
	targets := make([]*websocket.Conn, 0)
	for conn, meta := range global2.WsConnsQuiz {
		if meta.LessonID == lessonid {
			targets = append(targets, conn)
		}
	}
	global2.Mux.RUnlock()

	for _, conn := range targets {
		if err := conn.WriteJSON(quiz); err != nil {
			_ = conn.Close()
		}
	}
	return nil
}

func RunQuizRedisSubscriber(ctx context.Context, rdb *redis.Client) error {
	pubsub := rdb.Subscribe(ctx, "quiz:broadcast")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg := <-ch:
			var quizMsg map[string]interface{}
			if err := json.Unmarshal([]byte(msg.Payload), &quizMsg); err != nil {
				log.Printf("quiz redis unmarshal failed: err=%v", err)
				continue
			}

			msgType, _ := quizMsg["type"].(string)
			if msgType == "quiz_stats" {
				teacherID, _ := quizMsg["teacher_id"].(float64)
				if teacherID > 0 {
					_ = broadcastToTeacher(int64(teacherID), quizMsg)
				}
			}
		}
	}
}
