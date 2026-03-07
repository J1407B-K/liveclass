package main

import (
	"context"
	"errors"
	chat "liveclass/idl/kitex_gen/chat"
	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/rpc/chat/dao"
	"liveclass/internal/rpc/chat/kafka"
	"liveclass/internal/rpc/chat/model"
	"log"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	"go.mongodb.org/mongo-driver/mongo"
)

// ChatServiceImpl implements the last service interface defined in the IDL.
type ChatServiceImpl struct {
	mongoClient *mongo.Client
	webrtcCli   webrtclive.Client
}

func NewWebRTCLiveClient() (webrtclive.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	return webrtclive.NewClient("webrtc_liveservice", client.WithResolver(r),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler))
}

func (s *ChatServiceImpl) LiveChat(ctx context.Context, req *chat.LiveChatReq) (*chat.LiveChatResp, error) {
	if err := s.requireStudentInLesson(ctx, req.Lessonid, req.Userid); err != nil {
		return nil, err
	}

	coll := dao.ChooseCollection(req.Lessonid, s.mongoClient)

	msg := model.Message{
		LessonID:  req.Lessonid,
		Sender:    req.Userid,
		Content:   req.Message,
		Timestamp: time.Now(),
	}

	if err := dao.InsertMongo(ctx, coll, msg); err != nil {
		return nil, err
	}

	if err := kafka.ProduceMessage(req.Userid, req.Lessonid, msg.Timestamp, []byte(req.Message)); err != nil {
		return nil, err
	}

	return &chat.LiveChatResp{
		Resp: &common.Resp{
			Code: 0,
			Msg:  "success",
		},
	}, nil
}

// GetHistory implements the ChatServiceImpl interface.
func (s *ChatServiceImpl) GetHistory(ctx context.Context, req *chat.GetHistoryReq) (*chat.GetHistoryResp, error) {
	if err := s.requireStudentInLesson(ctx, req.LessonId, req.Userid); err != nil {
		return nil, errors.New("你无权查看聊天记录")
	}

	coll := dao.ChooseCollection(req.LessonId, s.mongoClient)

	h, err := dao.SelectMongo(ctx, coll)
	if err != nil {
		return nil, err
	}

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
	if err := s.requireTeacherOfLesson(ctx, req.LessonId, req.Userid); err != nil {
		return nil, err
	}

	coll := dao.ChooseCollection(req.LessonId, s.mongoClient)

	if err := dao.DropCollection(ctx, coll); err != nil {
		return nil, err
	}

	return &chat.DelHistoryResp{
		Resp: &common.Resp{
			Code: 0,
			Msg:  "success",
		},
	}, nil
}

func (s *ChatServiceImpl) getLessonInfo(ctx context.Context, lessonID int64) (*common.Lesson, error) {
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
	return nil
}

func (s *ChatServiceImpl) requireTeacherOfLesson(ctx context.Context, lessonID int64, userID int64) error {
	lesson, err := s.getLessonInfo(ctx, lessonID)
	if err != nil {
		return err
	}
	if lesson.TeacherID != userID {
		return errors.New("权限不够：你不是该课程老师")
	}
	return nil
}
