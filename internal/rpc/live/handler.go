package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/cloudwego/kitex/client"
	"github.com/go-redis/redis/v8"
	etcd "github.com/kitex-contrib/registry-etcd"
	"gorm.io/gorm"
	"io"
	"liveclass/idl/kitex_gen/common"
	live "liveclass/idl/kitex_gen/live"
	"liveclass/idl/kitex_gen/user"
	"liveclass/idl/kitex_gen/user/userservice"
	"liveclass/internal/rpc/live/dao"
	"liveclass/internal/rpc/live/global"
	"liveclass/internal/rpc/live/model"
	"liveclass/internal/utils/cut"
	"log"
	"net/http"
	"strconv"
)

// LiveServiceImpl implements the last service interface defined in the IDL.
type LiveServiceImpl struct {
	DB             *gorm.DB
	RDB            *redis.Client
	userCli        userservice.Client
	GetLiveKeyAddr string
	countsha       string
	membersha      string
	delsha         string
	selectsha      string
}

func NewUserClient() (userservice.Client, error) {
	// 使用时请传入真实 etcd 的服务地址，本例中为 127.0.0.1:2379
	r, err := etcd.NewEtcdResolver([]string{"127.0.0.1:2379"})
	if err != nil {
		log.Fatal(err)
	}
	return userservice.NewClient("userservice", client.WithResolver(r)) // 指定 Resolver
}

// CreateLive implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) CreateLive(ctx context.Context, req *live.CreateLiveReq) (resp *live.CreateLiveResp, err error) {
	//请求userrpc拿到userinfo
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！非老师不能创建课程/直播")
	}

	var datajson model.Livegojson
	//拿到json
	data, err := http.Get(s.GetLiveKeyAddr + cut.CombineLesson(req.Livename, username))
	if err != nil {
		return nil, err
	}
	defer data.Body.Close()

	//读取
	body, err := io.ReadAll(data.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &datajson)
	if err != nil {
		return nil, err
	}

	if datajson.Status != 200 {
		return nil, errors.New("livego 生成key错误")
	}

	err = dao.SaveLesson(s.DB, req.Livename, req.Description, username, datajson.Data)
	if err != nil {
		return nil, err
	}

	addr := cut.CombineAddr(global.Config.RTMPPlayAddr, global.Config.FLVPlayAddr, global.Config.HLSPlayAddr, cut.CombineLesson(req.Livename, username))

	return &live.CreateLiveResp{
		Resp: &common.Resp{
			Data: datajson.Data + "$" + cut.ShowAddr(addr),
		},
	}, nil
}

// CloseLive implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) CloseLive(ctx context.Context, req *live.CloseLiveReq) (resp *live.CloseLiveResp, err error) {
	//请求userrpc拿到userinfo
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！你不是当前课程老师")
	}

	lessons, err := dao.SelectLessonByTeacher(s.DB, username)
	if err != nil {
		return nil, err
	}
	for _, lesson := range lessons {
		if lesson.Name == req.Livename {
			err := dao.DeleteLesson(s.DB, req.Livename, username)
			if err != nil {
				return nil, err
			}

			//直接删除两个redis kv
			_, err = s.RDB.EvalSha(ctx, s.delsha, []string{lesson.Name + ":" + username + ":count", lesson.Name + ":" + username + ":member"}).Result()
			if err != nil {
				return nil, err
			}

			return &live.CloseLiveResp{
				Resp: &common.Resp{
					Data: "success",
				},
			}, nil
		}
	}

	return nil, errors.New("权限不够！你不是当前课程老师")
}

// SelectLessonInfo implements the LiveServiceImpl interface.
// 获取人数等信息
func (s *LiveServiceImpl) SelectLessonInfo(ctx context.Context, req *live.SelectLessonInfoReq) (resp *live.SelectLessonInfoResp, err error) {
	r, err := s.RDB.EvalSha(ctx, s.selectsha, []string{req.Lessonname + ":" + req.Teacher + ":count", req.Lessonname + ":" + req.Teacher + ":member"}).Result()
	if err != nil {
		return nil, err
	}
	ar, ok := r.([]interface{})
	if !ok {
		return nil, errors.New("解析redis lessonInfo 失败")
	}

	// 解析在线人数
	countStr := ar[0].(string)

	// 解析成员列表
	var membersStr string
	for i := 1; i < len(ar); i++ {
		membersStr += ar[i].(string)
		if i%2 == 0 {
			membersStr += "  "
		} else {
			membersStr += "$"
		}
	}

	return &live.SelectLessonInfoResp{
		Resp: &common.Resp{
			Data: "count:" + countStr + "///" + "live member:" + membersStr,
		},
	}, nil
}

// ChangeUserInLive implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) ChangeUserInLive(ctx context.Context, req *live.ChangeUserInLiveReq) (resp *live.ChangeUserInLiveResp, err error) {
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)

	var c int
	if req.Options == "add" {
		c = 1
	} else {
		c = -1
	}

	//暂时只设计了+1/-1
	_, err = s.RDB.EvalSha(ctx, s.countsha, []string{req.Livename + ":" + username + ":count"}, c).Result()
	if err != nil {
		return nil, err
	}
	_, err = s.RDB.EvalSha(ctx, s.membersha, []string{req.Livename + ":" + username + ":member"}, req.Options, username, auth).Result()
	if err != nil {
		return nil, err
	}

	return &live.ChangeUserInLiveResp{
		Resp: &common.Resp{
			Data: "success",
		},
	}, nil
}

// GetLessonInfo implements the LiveServiceImpl interface.
// 这个是获取在mysql中lesson的信息
func (s *LiveServiceImpl) GetLessonInfo(ctx context.Context, req *live.GetLessonInfoReq) (resp *live.GetLessonInfoResp, err error) {
	lesson, err := dao.SelectLessonByNandT(s.DB, req.Lessonname, req.Teacher)
	if err != nil {
		return nil, err
	}
	info := strconv.Itoa(lesson.LessonId) + "$" + lesson.Name + "$" + lesson.Teacher + "$" + lesson.Description + "$" + lesson.Code

	return &live.GetLessonInfoResp{
		Resp: &common.Resp{
			Data: info,
		},
	}, nil
}
