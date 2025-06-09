package main

import (
	"context"
	"errors"
	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	"gorm.io/gorm"
	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/live"
	"liveclass/idl/kitex_gen/live/liveservice"
	quiz "liveclass/idl/kitex_gen/quiz"
	"liveclass/idl/kitex_gen/user"
	"liveclass/idl/kitex_gen/user/userservice"
	"liveclass/internal/rpc/quiz/dao"
	"liveclass/internal/utils/cut"
	"log"
	"strconv"
	"time"
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
	return liveservice.NewClient("liveservice", client.WithResolver(r),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler))
}

func NewUserClient() (userservice.Client, error) {
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	return userservice.NewClient("userservice", client.WithResolver(r),
		client.WithSuite(tracing.NewClientSuite()),
		client.WithMetaHandler(transmeta.ClientTTHeaderHandler))
}

// CreateQuestion implements the QuizServiceImpl interface.
func (s *QuizServiceImpl) CreateQuestion(ctx context.Context, req *quiz.CreateQuestionReq) (resp *quiz.CreateQuestionResp, err error) {
	userresp, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//分割userinfo
	username, auth := cut.SplitInfo(userresp.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！你不是老师")
	}

	liveresp, err := s.liveCli.GetLessonInfoById(ctx, &live.GetLessonInfoByIdReq{Lessonid: req.LessonId})
	if err != nil {
		return nil, err
	}

	//分割lessoninfo
	slice := cut.SplitToLessonID(liveresp.Resp.Data)
	if username != slice[2] {
		return nil, errors.New("权限不够！你不是当前课程老师！")
	}

	lid, err := strconv.Atoi(req.LessonId)
	if err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(req.Userid)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	err = dao.SaveQuestion(s.DB, lid, uid, now, req)
	if err != nil {
		return nil, err
	}

	qid, err := dao.GetQuestionId(s.DB, req.Content)
	if err != nil {
		return nil, err
	}

	err = dao.CreateAnswer(s.DB, qid, int(req.OptionsNum), req.Answer, now, req.Duration)
	if err != nil {
		return nil, err
	}

	return &quiz.CreateQuestionResp{
		Resp: &common.Resp{
			Data: strconv.Itoa(qid),
		},
	}, nil
}

// TorFAnswer implements the QuizServiceImpl interface.
func (s *QuizServiceImpl) TorFAnswer(ctx context.Context, req *quiz.TorFAnswerReq) (resp *quiz.TorFAnswerResp, err error) {
	qid, err := strconv.Atoi(req.QuestionId)
	if err != nil {
		return nil, err
	}

	uid, err := strconv.Atoi(req.Userid)
	if err != nil {
		return nil, err
	}

	err = dao.CheckCloseTime(s.DB, qid)
	if err != nil {
		if err.Error() == "close" {
			return &quiz.TorFAnswerResp{Resp: &common.Resp{Data: req.Userid + "$" + "问题已经关闭"}}, nil
		}
		return nil, err
	}

	_, aid, err := dao.SelectAnswer(s.DB, qid)
	if err != nil {
		return nil, err
	}

	for _, id := range aid {
		suid, _ := strconv.Atoi(id)
		if suid == uid {
			return nil, errors.New("你已经回答过此问题！！！")
		}
	}

	q, err := dao.GetQuestion(s.DB, qid)
	if err != nil {
		return nil, err
	}

	err = dao.AddAnswerWithDefaultTx(s.DB, uid, qid, req.UserAnswer)
	if err != nil {
		return nil, err
	}

	so, _, err := dao.SelectAnswer(s.DB, qid)
	if err != nil {
		return nil, err
	}

	return &quiz.TorFAnswerResp{
		Resp: &common.Resp{
			Data: strconv.Itoa(q.TeacherId) + "$" + "问题ID:" + strconv.Itoa(q.ID) + "   " + cut.ShowOptions(so) + "|right answer:" + q.Answer,
		},
	}, nil
}

// DelQuestion implements the QuizServiceImpl interface.
func (s *QuizServiceImpl) DelQuestion(ctx context.Context, req *quiz.DelQuestionReq) (resp *quiz.DelQuestionResp, err error) {
	userresp, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//分割userinfo
	username, auth := cut.SplitInfo(userresp.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！你不是老师")
	}

	qid, err := strconv.Atoi(req.QuestionId)
	if err != nil {
		return nil, err
	}

	q, err := dao.GetQuestion(s.DB, qid)
	if err != nil {
		return nil, err
	}

	liveresp, err := s.liveCli.GetLessonInfoById(ctx, &live.GetLessonInfoByIdReq{Lessonid: strconv.Itoa(q.LessonId)})
	if err != nil {
		return nil, err
	}

	//分割lessoninfo
	slice := cut.SplitToLessonID(liveresp.Resp.Data)
	if username != slice[2] {
		return nil, errors.New("权限不够！你不是当前课程老师！")
	}

	err = dao.DelQuestionAndAnswer(s.DB, qid)
	if err != nil {
		return nil, err
	}

	return &quiz.DelQuestionResp{
		Resp: &common.Resp{
			Data: "success",
		},
	}, nil
}

// GetAllLessonQuiz implements the QuizServiceImpl interface.
func (s *QuizServiceImpl) GetAllLessonQuiz(ctx context.Context, req *quiz.GetAllLessonQuizReq) (resp *quiz.GetAllLessonQuizResp, err error) {
	lid, err := strconv.Atoi(req.LessonId)
	if err != nil {
		return nil, err
	}
	res, err := s.liveCli.IsStudentInLesson(ctx, &live.IsStudentInLessonReq{
		Studentid: req.Userid,
		Lessonid:  req.LessonId,
	})
	if err != nil {
		return nil, err
	}
	if res.Resp.Data == "not_in" {
		return &quiz.GetAllLessonQuizResp{Resp: &common.Resp{Data: "学生不在当前课程中"}}, nil
	}
	qs, err := dao.GetQustionByLesson(s.DB, lid)
	if err != nil {
		return nil, err
	}

	var str string
	var ops string
	for _, q := range qs {
		id := q.ID
		content := q.Content
		ans := q.Answer
		for _, o := range q.Options {
			ops += o + "|"
		}
		str += "问题ID:" + strconv.Itoa(id) + "$" + "问题内容:" + content + "$" + "问题选项:" + ops + "$" + "正确答案:" + ans + "\n"
	}
	return &quiz.GetAllLessonQuizResp{Resp: &common.Resp{Data: str}}, nil
}
