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
	"liveclass/internal/api/observability"
	"liveclass/internal/api/utils/jwt"
	"liveclass/internal/api/utils/ratelimit"
	"log"
	"math/rand"
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
	resp, err := global2.Clients.Webrtc_liveClient.IsStudentInLesson(
		c,
		&webrtc_live.IsStudentInLessonReq{Lessonid: ilid, Studentid: uid},
	)
	if err != nil {
		ctx.JSON(http.StatusServiceUnavailable, utils.H{
			"code": code.RPCError,
			"msg":  "lesson permission check failed",
		})
		return
	}
	if resp == nil || resp.Resp == nil || resp.Resp.Msg != "exist" {
		ctx.JSON(http.StatusForbidden, utils.H{
			"code": code.AuthError,
			"msg":  "not a lesson member",
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
	pubsub := rdb.PSubscribe(ctx, "chat:broadcast:*")
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
			if err := global2.ChatRooms.BroadcastJSON(chatMsg.LessonID, chatMsg); err != nil {
				log.Printf("chat redis broadcast marshal failed: err=%v", err)
			}
		}
	}
}

type ChatReader interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
	Close() error
}

func consumeChat(ctx context.Context, reader ChatReader) error {
	observability.SubscriberConnected.Set(1)
	defer observability.SubscriberConnected.Set(0)
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

		if err = global2.ChatRooms.BroadcastJSON(msg.LessonID, msg); err != nil {
			log.Printf("chat consumer broadcast marshal failed: offset=%d err=%v", m.Offset, err)
		}

		if err = reader.CommitMessages(ctx, m); err != nil {
			log.Printf("chat consumer commit failed: offset=%d err=%v", m.Offset, err)
		}
	}
}

func RunChatConsumer(ctx context.Context, newReader func() ChatReader) error {
	const (
		initialBackoff = 100 * time.Millisecond
		maxBackoff     = 5 * time.Second
		stableWindow   = 30 * time.Second
	)

	backoff := initialBackoff
	for {
		reader := newReader()
		started := time.Now()
		err := consumeChat(ctx, reader)
		_ = reader.Close()
		if ctx.Err() != nil {
			return nil
		}
		log.Printf("chat kafka consumer fetch failed: err=%v", err)
		observability.SubscriberReconnectTotal.Inc()
		if time.Since(started) >= stableWindow {
			backoff = initialBackoff
		}
		jitter := time.Duration(rand.Int63n(int64(backoff/2) + 1))
		timer := time.NewTimer(backoff + jitter)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

func chatHandler(c context.Context, userId, lessonId int64) websocket.HertzHandler {
	return func(conn *websocket.Conn) {
		// The HTTP request context has a short deadline. A successfully upgraded
		// WebSocket is a long-lived session and must not inherit that deadline.
		client := global2.ChatRooms.NewClient(context.WithoutCancel(c), lessonId, conn)
		_ = client.Serve(func(clientCtx context.Context, messageType int, message []byte) error {
			if messageType != websocket.TextMessage {
				return nil
			}

			var msgJson model2.Message
			if err := json.Unmarshal(message, &msgJson); err != nil {
				log.Println("unmarshal chat message error:", err)
				return nil
			}

			redisStarted := time.Now()
			if err := waitForDependency(clientCtx, global2.Config.FaultInjection.RedisDelay); err != nil {
				return err
			}
			allowed, err := ratelimit.AllowRedis(
				clientCtx,
				global2.DBManager.RDB,
				fmt.Sprintf("rl:chat:send:%d", userId),
				20,
				40,
				1,
				time.Minute,
			)
			observability.ChatRedisRateLimitLatency.Observe(time.Since(redisStarted).Seconds())
			if err != nil {
				log.Println("chat limiter error:", err)
				return nil
			}
			if !allowed {
				client.Enqueue([]byte("发送过于频繁，请稍后再试"))
				return nil
			}

			observability.ChatMessagesTotal.Inc()
			rpcStarted := time.Now()
			_, err = global2.Clients.ChatClient.LiveChat(clientCtx, &chat.LiveChatReq{
				Lessonid: lessonId,
				Userid:   userId,
				Message:  msgJson.Content,
			})
			observability.ChatRPCLatency.Observe(time.Since(rpcStarted).Seconds())
			if err != nil {
				log.Println("chat rpc error:", err)
				return nil
			}
			return nil
		})
	}
}

func waitForDependency(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
