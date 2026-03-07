package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"liveclass/idl/kitex_gen/common"
	quiz "liveclass/idl/kitex_gen/quiz"
	"liveclass/idl/kitex_gen/user"
	"liveclass/idl/kitex_gen/user/userservice"
	"liveclass/idl/kitex_gen/webrtc_live"
	"liveclass/idl/kitex_gen/webrtc_live/webrtclive"
	"liveclass/internal/rpc/quiz/dao"
	"liveclass/internal/rpc/quiz/model"
	"log"
	"math/rand"
	"strconv"
	"time"

	"github.com/cloudwego/kitex/client"
	"github.com/cloudwego/kitex/pkg/transmeta"
	"github.com/go-redis/redis/v8"
	"github.com/kitex-contrib/obs-opentelemetry/tracing"
	etcd "github.com/kitex-contrib/registry-etcd"
	"golang.org/x/sync/singleflight"
)

// QuizServiceImpl implements the last service interface defined in the IDL.
type QuizServiceImpl struct {
	DBManager *dao.DBManager
	webrtcCli webrtclive.Client
	userCli   userservice.Client

	sfQuiz singleflight.Group
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
func (s *QuizServiceImpl) CreateQuestion(ctx context.Context, req *quiz.CreateQuestionReq) (*quiz.CreateQuestionResp, error) {
	_, _, err := s.requireTeacherOfLesson(ctx, req.LessonId, req.Userid)
	if err != nil {
		return nil, err
	}

	q, err := s.DBManager.CreateQuestion(req)
	if err != nil {
		return nil, err
	}

	if s.DBManager.RDB != nil {
		_ = s.DBManager.RDB.Del(ctx, fmt.Sprintf("quiz:question:%d", q.ID)).Err()
	}

	return &quiz.CreateQuestionResp{
		Resp: &common.Resp{
			Code: 0,
			Msg:  "success",
			Data: &common.Data{
				QuizInfo: &common.Quiz{QuizID: q.ID},
			},
		},
	}, nil
}

// TorFAnswer implements the QuizServiceImpl interface.
func (s *QuizServiceImpl) TorFAnswer(ctx context.Context, req *quiz.TorFAnswerReq) (*quiz.TorFAnswerResp, error) {
	q, err := s.getQuestionCached(ctx, req.QuestionId)
	if err != nil {
		return nil, err
	}

	if err = s.requireStudentInLesson(ctx, q.LessonID, req.Userid); err != nil {
		return nil, err
	}

	err = s.DBManager.CheckCloseTime(req.QuestionId)
	if err != nil {
		if err.Error() == "close" {
			return &quiz.TorFAnswerResp{
				Resp: &common.Resp{Msg: "问题已经关闭"},
			}, nil
		}
		return nil, err
	}

	exists, err := s.DBManager.HasAnswered(req.QuestionId, req.Userid)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("你已经回答过此问题")
	}

	if err = s.DBManager.CreateUserAnswer(q, req.Userid, req.UserAnswer); err != nil {
		return nil, err
	}

	stats, err := s.DBManager.CountAnswersByQuestion(req.QuestionId)
	if err != nil {
		return nil, err
	}

	return &quiz.TorFAnswerResp{
		Resp: &common.Resp{
			Code: 0,
			Msg:  "success",
			Data: &common.Data{QuizInfo: &common.Quiz{QuizID: q.ID, TeacherID: q.TeacherID, Stats: toQuizStats(stats), Answer: q.Answer}},
		},
	}, nil
}

// DelQuestion implements the QuizServiceImpl interface.
func (s *QuizServiceImpl) DelQuestion(ctx context.Context, req *quiz.DelQuestionReq) (*quiz.DelQuestionResp, error) {
	q, err := s.getQuestionCached(ctx, req.QuestionId)
	if err != nil {
		return nil, err
	}

	_, _, err = s.requireTeacherOfLesson(ctx, q.LessonID, req.Userid)
	if err != nil {
		return nil, err
	}

	if err = s.DBManager.DelQuestionAndAnswer(req.QuestionId); err != nil {
		return nil, err
	}

	if s.DBManager.RDB != nil {
		_ = s.DBManager.RDB.Del(ctx, fmt.Sprintf("quiz:question:%d", req.QuestionId)).Err()
	}

	return &quiz.DelQuestionResp{
		Resp: &common.Resp{
			Code: 0,
			Msg:  "success",
		},
	}, nil
}

// GetAllLessonQuiz implements the QuizServiceImpl interface.
func (s *QuizServiceImpl) GetAllLessonQuiz(ctx context.Context, req *quiz.GetAllLessonQuizReq) (*quiz.GetAllLessonQuizResp, error) {
	if err := s.requireStudentInLesson(ctx, req.LessonId, req.Userid); err != nil {
		return &quiz.GetAllLessonQuizResp{
			Resp: &common.Resp{Msg: "学生不在当前课程中"},
		}, nil
	}

	qs, err := s.DBManager.GetQuestionByLesson(req.LessonId)
	if err != nil {
		return nil, err
	}

	var str string
	for _, q := range qs {
		ops := ""
		for _, o := range q.Options {
			ops += o + "|"
		}
		str += "问题ID:" + strconv.FormatInt(q.ID, 10) +
			"$问题内容:" + q.Content +
			"$问题选项:" + ops +
			"$正确答案:" + q.Answer + "\n"
	}

	return &quiz.GetAllLessonQuizResp{
		Resp: &common.Resp{Msg: str},
	}, nil
}

func (s *QuizServiceImpl) getUserInfo(ctx context.Context, userID int64) (*common.User, error) {
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: userID})
	if err != nil {
		return nil, err
	}
	if info == nil || info.Resp == nil || info.Resp.Data == nil {
		return nil, errors.New("user service: empty userinfo")
	}
	return info.Resp.Data.UserInfo, nil
}

func (s *QuizServiceImpl) getLessonInfo(ctx context.Context, lessonID int64) (*common.Lesson, error) {
	info, err := s.webrtcCli.GetLessonInfoById(ctx, &webrtc_live.GetLessonInfoByIdReq{Lessonid: lessonID})
	if err != nil {
		return nil, err
	}
	if info == nil || info.Resp == nil || info.Resp.Data == nil || info.Resp.Data.LessonInfo == nil {
		return nil, errors.New("webrtc_live service: empty lessoninfo")
	}
	return info.Resp.Data.LessonInfo, nil
}

func (s *QuizServiceImpl) requireTeacherOfLesson(ctx context.Context, lessonID int64, userID int64) (*common.Lesson, *common.User, error) {
	u, err := s.getUserInfo(ctx, userID)
	if err != nil {
		return nil, nil, err
	}

	lesson, err := s.getLessonInfo(ctx, lessonID)
	if err != nil {
		return nil, nil, err
	}

	if lesson.TeacherID != u.UserID {
		return nil, nil, errors.New("权限不够：你不是该课程老师")
	}
	return lesson, u, nil
}

func (s *QuizServiceImpl) requireStudentInLesson(ctx context.Context, lessonID int64, userID int64) error {
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

func toQuizStats(stats []model.AnswerStat) []*common.QuizStat {
	res := make([]*common.QuizStat, 0, len(stats))
	for _, s := range stats {
		res = append(res, &common.QuizStat{
			Option: s.Answer,
			Count:  s.Count,
		})
	}
	return res
}

func (s *QuizServiceImpl) getQuestionCached(ctx context.Context, qid int64) (*model.Question, error) {
	cacheKey := fmt.Sprintf("quiz:question:%d", qid)

	if s.DBManager.RDB != nil {
		b, err := s.DBManager.RDB.Get(ctx, cacheKey).Bytes()
		if err == nil && len(b) > 0 {
			var q model.Question
			if jsonErr := json.Unmarshal(b, &q); jsonErr == nil {
				return &q, nil
			}
			_ = s.DBManager.RDB.Del(ctx, cacheKey).Err()
		}
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
	}

	v, err, _ := s.sfQuiz.Do(cacheKey, func() (interface{}, error) {
		q, err := s.DBManager.GetQuestion(qid)
		if err != nil {
			return nil, err
		}

		if s.DBManager.RDB != nil {
			if b, jerr := json.Marshal(q); jerr == nil {
				ttl := 5*time.Minute + time.Duration(rand.Intn(30))*time.Second
				_ = s.DBManager.RDB.Set(ctx, cacheKey, b, ttl).Err()
			}
		}
		return q, nil
	})
	if err != nil {
		return nil, err
	}

	q, ok := v.(*model.Question)
	if !ok || q == nil {
		return nil, errors.New("invalid question type")
	}
	return q, nil
}
