package main

import (
	"context"
	"errors"
	"github.com/cloudwego/kitex/client"
	etcd "github.com/kitex-contrib/registry-etcd"
	"go.mongodb.org/mongo-driver/mongo"
	chat "liveclass/idl/kitex_gen/chat"
	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/live"
	"liveclass/idl/kitex_gen/live/liveservice"
	"liveclass/internal/rpc/chat/dao"
	"liveclass/internal/rpc/chat/kafka"
	"liveclass/internal/rpc/chat/model"
	"log"
	"strings"
	"time"
)

// ChatServiceImpl implements the last service interface defined in the IDL.
type ChatServiceImpl struct {
	mongoClient *mongo.Client
	liveCli     liveservice.Client
}

func NewLiveClient() (liveservice.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	return liveservice.NewClient("liveservice", client.WithResolver(r)) // 指定 Resolver
}

// LiveChat implements the ChatServiceImpl interface.
func (s *ChatServiceImpl) LiveChat(ctx context.Context, req *chat.LiveChatReq) (resp *chat.LiveChatResp, err error) {
	r, err := s.liveCli.GetLessonInfoById(ctx, &live.GetLessonInfoByIdReq{Lessonid: req.Lessonid})
	if err != nil {
		return nil, err
	}

	uids := strings.Split(r.Resp.Data, "/")
	uids = uids[:len(uids)-1]

	for _, id := range uids {
		if id != req.Userid {
			continue
		}

		coll := dao.ChooseCollection(req.Lessonid, s.mongoClient)

		timestamp := time.Now()
		var msg = model.Message{
			LessonID:  req.Lessonid,
			Sender:    req.Userid,
			Content:   req.Message,
			Timestamp: timestamp,
		}

		err = dao.InsertMongo(ctx, coll, msg)
		if err != nil {
			return nil, err
		}

		err = kafka.ProduceMessage(req.Userid, req.Lessonid, timestamp, []byte(req.Message))
		if err != nil {
			return nil, err
		}

		return &chat.LiveChatResp{Resp: &common.Resp{Data: req.Lessonid}}, nil
	}

	return nil, errors.New("你不是当前课程学生！")
}

// GetHistory implements the ChatServiceImpl interface.
func (s *ChatServiceImpl) GetHistory(ctx context.Context, req *chat.GetHistoryReq) (resp *chat.GetHistoryResp, err error) {
	coll := dao.ChooseCollection(req.LessonId, s.mongoClient)

	h, err := dao.SelectMongo(ctx, coll)
	if err != nil {
		return nil, err
	}
	return &chat.GetHistoryResp{Resp: &common.Resp{Data: h}}, nil
}

// DelHistory implements the ChatServiceImpl interface.
func (s *ChatServiceImpl) DelHistory(ctx context.Context, req *chat.DelHistoryReq) (resp *chat.DelHistoryResp, err error) {
	coll := dao.ChooseCollection(req.LessonId, s.mongoClient)

	err = dao.DropCollection(ctx, coll)
	if err != nil {
		return nil, err
	}

	return &chat.DelHistoryResp{Resp: &common.Resp{Data: "success"}}, nil
}
