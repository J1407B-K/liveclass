package main

import (
	"context"
	"errors"
	"github.com/cloudwego/kitex/client"
	etcd "github.com/kitex-contrib/registry-etcd"
	"gorm.io/gorm"
	"liveclass/idl/kitex_gen/live"
	"liveclass/idl/kitex_gen/live/liveservice"
	quiz "liveclass/idl/kitex_gen/quiz"
	"liveclass/idl/kitex_gen/user"
	"liveclass/idl/kitex_gen/user/userservice"
	"liveclass/internal/rpc/quiz/dao"
	"liveclass/internal/utils/cut"
	"log"
	"strconv"
)

// QuizServiceImpl implements the last service interface defined in the IDL.
type QuizServiceImpl struct {
	DB      *gorm.DB
	liveCli liveservice.Client
	userCli userservice.Client
}

func NewLiveClient() (liveservice.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	return liveservice.NewClient("liveservice", client.WithResolver(r)) // 指定 Resolver
}

func NewUserClient() (userservice.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	return userservice.NewClient("userservice", client.WithResolver(r))
}

// CreateQuestion implements the QuizServiceImpl interface.
func (s *QuizServiceImpl) CreateQuestion(ctx context.Context, req *quiz.CreateQuestionReq) (resp *quiz.CreateQuestionResp, err error) {
	userresp, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}
	username, auth := cut.SplitInfo(userresp.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！你不是老师或不是当前课程老师")
	}

	liveresp, err := s.liveCli.GetLessonInfo(ctx, &live.GetLessonInfoReq{Lessonname: req.LessonName, Teacher: username})
	if err != nil {
		return nil, err
	}

	idStr := cut.SplitToLessonID(liveresp.Resp.Data)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, err
	}
	dao.SaveQuestion(s.DB, id, req)

	return
}

// TorFAnswer implements the QuizServiceImpl interface.
func (s *QuizServiceImpl) TorFAnswer(ctx context.Context, req *quiz.TorFAnswerReq) (resp *quiz.TorFAnswerResp, err error) {
	// TODO: Your code here...
	return
}

// ListAllUserQuiz implements the QuizServiceImpl interface.
func (s *QuizServiceImpl) ListAllUserQuiz(ctx context.Context, req *quiz.ListAllUserQuizReq) (resp *quiz.ListAllUserQuizResp, err error) {
	// TODO: Your code here...
	return
}
