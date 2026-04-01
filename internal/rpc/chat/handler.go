package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	chat "liveclass/idl/kitex_gen/chat"
	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/rpc/chat/dao"
	"liveclass/internal/rpc/chat/global"
	"liveclass/internal/rpc/chat/kafka"
	"liveclass/internal/rpc/chat/model"
	"log"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	"go.mongodb.org/mongo-driver/mongo"
	"go.opentelemetry.io/otel"
)

// ChatServiceImpl implements the last service interface defined in the IDL.
type ChatServiceImpl struct {
	mongoClient *mongo.Client
	webrtcCli   webrtclive.Client
}

func NewWebRTCLiveClient() (webrtclive.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{global.Config.EtcdAddr})
	if err != nil {
		return nil, err
	}
	return webrtclive.NewClient("webrtc_liveservice", client.WithResolver(r),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler))
}

func (s *ChatServiceImpl) LiveChat(ctx context.Context, req *chat.LiveChatReq) (*chat.LiveChatResp, error) {
	if err := s.requireStudentInLesson(ctx, req.Lessonid, req.Userid); err != nil {
		log.Printf("[LiveChat] Permission denied: user=%d, lesson=%d, err=%v", req.Userid, req.Lessonid, err)
		return nil, err
	}

	cleanedMsg, timestamp := kafka.FilterMessage(req.Message)

	msg := model.Message{
		LessonID:  req.Lessonid,
		Sender:    req.Userid,
		Content:   cleanedMsg,
		Timestamp: timestamp,
	}

	tracer := otel.Tracer("chatservice")
	var kafkaErr, mongoErr error
	done := make(chan struct{}, 2)

	go func() {
		spanCtx, span := tracer.Start(ctx, "kafka.produce")
		defer span.End()
		kafkaErr = kafka.ProduceFilteredMessage(req.Userid, req.Lessonid, msg)
		if kafkaErr != nil {
			log.Printf("[LiveChat] Kafka write failed: user=%d, lesson=%d, err=%v", req.Userid, req.Lessonid, kafkaErr)
		}
		_ = spanCtx
		done <- struct{}{}
	}()

	go func() {
		spanCtx, span := tracer.Start(ctx, "mongo.insert")
		defer span.End()
		coll := dao.ChooseCollection(req.Lessonid, s.mongoClient)
		mongoErr = dao.InsertMongo(spanCtx, coll, msg)
		if mongoErr != nil {
			log.Printf("[LiveChat] MongoDB write failed: user=%d, lesson=%d, err=%v", req.Userid, req.Lessonid, mongoErr)
		}
		done <- struct{}{}
	}()

	<-done
	<-done

	if mongoErr != nil {
		return nil, mongoErr
	}

	go func() {
		msgBytes, err := json.Marshal(msg)
		if err != nil {
			log.Printf("[LiveChat] Redis marshal failed: %v", err)
			return
		}
		if err := global.RedisClient.Publish(context.Background(), "chat:broadcast", msgBytes).Err(); err != nil {
			log.Printf("[LiveChat] Redis publish failed: %v", err)
		}
	}()

	return &chat.LiveChatResp{
		Resp: &common.Resp{
			Code: 0,
			Msg:  "success",
		},
	}, nil
}

// GetHistory implements the ChatServiceImpl interface.
func (s *ChatServiceImpl) GetHistory(ctx context.Context, req *chat.GetHistoryReq) (*chat.GetHistoryResp, error) {
	tracer := otel.Tracer("chatservice")
	ctx, span := tracer.Start(ctx, "GetHistory")
	defer span.End()

	if err := s.requireStudentInLesson(ctx, req.LessonId, req.Userid); err != nil {
		log.Printf("[GetHistory] Permission denied: user=%d, lesson=%d, err=%v", req.Userid, req.LessonId, err)
		return nil, errors.New("你无权查看聊天记录")
	}

	coll := dao.ChooseCollection(req.LessonId, s.mongoClient)

	h, err := dao.SelectMongo(ctx, coll)
	if err != nil {
		log.Printf("[GetHistory] Query failed: user=%d, lesson=%d, err=%v", req.Userid, req.LessonId, err)
		return nil, err
	}

	log.Printf("[GetHistory] Success: user=%d, lesson=%d", req.Userid, req.LessonId)
	return &chat.GetHistoryResp{
		Resp: &common.Resp{
			Code: 0,
			Msg:  "success",
			Data: &common.Data{ChatInfo: &common.Chat{Message: h}},
		},
	}, nil
}

// DelHistory implements the ChatServiceImpl interface.
func (s *ChatServiceImpl) DelHistory(ctx context.Context, req *chat.DelHistoryReq) (*chat.DelHistoryResp, error) {
	tracer := otel.Tracer("chatservice")
	ctx, span := tracer.Start(ctx, "DelHistory")
	defer span.End()

	if err := s.requireTeacherOfLesson(ctx, req.LessonId, req.Userid); err != nil {
		log.Printf("[DelHistory] Permission denied: user=%d, lesson=%d, err=%v", req.Userid, req.LessonId, err)
		return nil, err
	}

	coll := dao.ChooseCollection(req.LessonId, s.mongoClient)

	if err := dao.DropCollection(ctx, coll); err != nil {
		log.Printf("[DelHistory] Drop collection failed: user=%d, lesson=%d, err=%v", req.Userid, req.LessonId, err)
		return nil, err
	}

	log.Printf("[DelHistory] Success: user=%d, lesson=%d", req.Userid, req.LessonId)
	return &chat.DelHistoryResp{
		Resp: &common.Resp{
			Code: 0,
			Msg:  "success",
		},
	}, nil
}

func (s *ChatServiceImpl) getLessonInfo(ctx context.Context, lessonID int64) (*common.Lesson, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	info, err := s.webrtcCli.GetLessonInfoById(ctx, &webrtc_live.GetLessonInfoByIdReq{Lessonid: lessonID})
	if err != nil {
		return nil, err
	}
	if info == nil || info.Resp == nil || info.Resp.Data == nil || info.Resp.Data.LessonInfo == nil {
		return nil, errors.New("webrtc_live service: empty lessoninfo")
	}
	return info.Resp.Data.LessonInfo, nil
}

func (s *ChatServiceImpl) requireStudentInLesson(ctx context.Context, lessonID int64, userID int64) error {
	cacheKey := fmt.Sprintf("perm:stu:%d:%d", lessonID, userID)

	if global.RedisClient != nil {
		if val, _ := global.RedisClient.Get(ctx, cacheKey).Result(); val == "1" {
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	resp, err := s.webrtcCli.IsStudentInLesson(ctx, &webrtc_live.IsStudentInLessonReq{
		Lessonid:  lessonID,
		Studentid: userID,
	})
	if err != nil {
		return err
	}
	if resp == nil || resp.Resp == nil {
		return errors.New("webrtc_live service: empty student check")
	}
	if resp.Resp.Msg != "exist" {
		return errors.New("你不是该课程学生")
	}

	if global.RedisClient != nil {
		_ = global.RedisClient.Set(ctx, cacheKey, "1", 5*time.Minute).Err()
	}
	return nil
}

func (s *ChatServiceImpl) requireTeacherOfLesson(ctx context.Context, lessonID int64, userID int64) error {
	cacheKey := fmt.Sprintf("perm:teacher:%d:%d", lessonID, userID)

	if global.RedisClient != nil {
		if val, _ := global.RedisClient.Get(ctx, cacheKey).Result(); val == "1" {
			return nil
		}
	}

	lesson, err := s.getLessonInfo(ctx, lessonID)
	if err != nil {
		return err
	}
	if lesson.TeacherID != userID {
		return errors.New("权限不够：你不是该课程老师")
	}

	if global.RedisClient != nil {
		_ = global.RedisClient.Set(ctx, cacheKey, "1", 5*time.Minute).Err()
	}
	return nil
}
