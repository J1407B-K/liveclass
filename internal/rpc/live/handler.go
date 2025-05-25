package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudwego/kitex/client"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	etcd "github.com/kitex-contrib/registry-etcd"
	"github.com/tencentyun/cos-go-sdk-v5"
	"gorm.io/gorm"
	"io"
	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/live"
	"liveclass/idl/kitex_gen/user"
	"liveclass/idl/kitex_gen/user/userservice"
	my_cos "liveclass/internal/rpc/live/cos"
	"liveclass/internal/rpc/live/dao"
	"liveclass/internal/rpc/live/global"
	"liveclass/internal/rpc/live/model"
	"liveclass/internal/utils/cut"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	cosClient      *cos.Client
}

func NewUserClient() (userservice.Client, error) {
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

	err = dao.SaveLesson(s.DB, req.Livename, req.Description, username, datajson.Data, req.Userid)
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
// 获取在mysql中lesson的信息(用name和teacher)
func (s *LiveServiceImpl) GetLessonInfo(ctx context.Context, req *live.GetLessonInfoReq) (resp *live.GetLessonInfoResp, err error) {
	lesson, err := dao.SelectLessonByNandT(s.DB, req.Lessonname, req.Teacher)
	if err != nil {
		return nil, err
	}

	var stuidStr string
	for _, uid := range lesson.StudentID {
		stuidStr += uid + "/"
	}

	info := strconv.Itoa(lesson.LessonId) + "$" + lesson.Name + "$" + lesson.Teacher + "$" + lesson.Description + "$" + lesson.Code + "$" + stuidStr

	return &live.GetLessonInfoResp{
		Resp: &common.Resp{
			Data: info,
		},
	}, nil
}

// GetLessonInfoById implements the LiveServiceImpl interface.
// 用id获取
func (s *LiveServiceImpl) GetLessonInfoById(ctx context.Context, req *live.GetLessonInfoByIdReq) (resp *live.GetLessonInfoByIdResp, err error) {
	id, err := strconv.Atoi(req.Lessonid)
	if err != nil {
		return nil, err
	}
	lesson, err := dao.SelectLessonById(s.DB, id)
	if err != nil {
		return nil, err
	}

	var stuidStr string
	for _, uid := range lesson.StudentID {
		stuidStr += uid + "/"
	}

	info := strconv.Itoa(lesson.LessonId) + "$" + lesson.Name + "$" + lesson.Teacher + "$" + lesson.Description + "$" + lesson.Code + "/" + stuidStr

	return &live.GetLessonInfoByIdResp{
		Resp: &common.Resp{
			Data: info,
		},
	}, nil
}

// ChangeUserToLesson implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) ChangeUserToLesson(ctx context.Context, req *live.ChangeUserToLessonReq) (resp *live.ChangeUserToLessonResp, err error) {
	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！！！你不是老师")
	} else if username != req.Teacher {
		return nil, errors.New("权限不够！！！你不是当前课程老师")
	}

	if req.Option != "add" && req.Option != "del" {
		return nil, errors.New("invalid options")
	}

	err = dao.ChangeUserToLesson(s.DB, req.Studentid, req.Lessonname, req.Teacher, req.Option)
	if err != nil {
		return nil, err
	}
	return &live.ChangeUserToLessonResp{
		Resp: &common.Resp{
			Data: "success",
		},
	}, nil
}

// IsStudentInLesson implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) IsStudentInLesson(ctx context.Context, req *live.IsStudentInLessonReq) (resp *live.IsStudentInLessonResp, err error) {
	info, err := dao.CheckStudentInLesson(s.DB, req.Studentid, req.Lessonid)
	if err != nil {
		return nil, err
	}
	if info == "in" {
		return &live.IsStudentInLessonResp{
			Resp: &common.Resp{
				Data: "in",
			},
		}, nil
	} else if info == "not_in" {
		return &live.IsStudentInLessonResp{
			Resp: &common.Resp{
				Data: "not_in",
			},
		}, nil
	}

	return &live.IsStudentInLessonResp{
		Resp: &common.Resp{
			Data: errors.New("unknown error happened").Error(),
		},
	}, nil
}

// RecordLesson implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) RecordLesson(ctx context.Context, req *live.RecordLessonReq) (resp *live.RecordLessonResp, err error) {
	lid, err := strconv.Atoi(req.LessonId)
	if err != nil {
		return nil, err
	}

	linfo, err := dao.SelectLessonById(s.DB, lid)
	if err != nil {
		return nil, err
	}

	for _, uid := range linfo.StudentID {
		if req.Userid == uid {
			err = os.Mkdir(global.Config.TmpBaseDir, 0755)
			if err != nil && !os.IsExist(err) {
				return nil, err
			}

			filename := fmt.Sprintf("%s-record-%s.mp4", req.LessonId, uuid.NewString())
			localfile := filepath.Join(global.Config.TmpBaseDir, filename)

			args := []string{"-y", "-i", req.StreamURL, "-c", "copy", localfile}
			if req.Duration > 0 {
				args = append(args, "-t", fmt.Sprintf("%d", req.Duration))
			}

			log.Println("开始录制")
			go func() {
				cmd := exec.CommandContext(ctx, "ffmpeg", args...)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				if err := cmd.Run(); err != nil {
					log.Printf("ffmpeg拉流失败: %v", err)
					return
				}

				if err := my_cos.UploadToCos(ctx, s.cosClient, localfile, req.LessonId, filename); err != nil {
					log.Printf("上传失败: %v", err)
					return
				}

				if err := os.Remove(localfile); err != nil {
					log.Printf("删除临时文件失败: %v", err)
				}

				log.Println("异步录制和上传成功")
			}()

			return &live.RecordLessonResp{Resp: &common.Resp{Data: "success:https://console.cloud.tencent.com/cos/bucket?bucket=lanshan-1338048877&region=ap-chongqing"}}, nil
		}
	}
	return nil, errors.New("你不是当前课程的学生或老师！！！")
}

// CreateSignIn implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) CreateSignIn(ctx context.Context, req *live.CreateSignInReq) (resp *live.CreateSignInResp, err error) {
	lid, err := strconv.Atoi(req.Lessonid)
	if err != nil {
		return nil, err
	}

	linfo, err := dao.SelectLessonById(s.DB, lid)
	if err != nil {
		return nil, err
	}

	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！！！你不是老师")
	} else if username != linfo.Teacher {
		return nil, errors.New("权限不够！！！你不是当前课程老师")
	}

	err = dao.CreateSignIn(s.DB, req.Lessonid, linfo.StudentID)
	if err != nil {
		return nil, err
	}
	return &live.CreateSignInResp{Resp: &common.Resp{Data: "success"}}, nil
}

// SignIn implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) SignIn(ctx context.Context, req *live.SignInReq) (resp *live.SignInResp, err error) {
	lid, err := strconv.Atoi(req.Lessonid)
	if err != nil {
		return nil, err
	}

	linfo, err := dao.SelectLessonById(s.DB, lid)
	if err != nil {
		return nil, err
	}

	for _, stuid := range linfo.StudentID {
		if req.Userid == stuid {
			_, err = dao.StuSignIn(s.DB, req.Lessonid, req.Userid)
			if err != nil {
				return nil, err
			}
		}
		return &live.SignInResp{Resp: &common.Resp{Data: "success"}}, nil
	}
	return nil, errors.New("不是此课程学生")
}

// SelectSignIn implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) SelectSignIn(ctx context.Context, req *live.SelectSignInReq) (resp *live.SelectSignInResp, err error) {
	lid, err := strconv.Atoi(req.Lessonid)
	if err != nil {
		return nil, err
	}
	linfo, err := dao.SelectLessonById(s.DB, lid)
	if err != nil {
		return nil, err
	}

	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！！！你不是老师")
	} else if username != linfo.Teacher {
		return nil, errors.New("权限不够！！！你不是当前课程老师")
	}

	sinfo, err := dao.SelectSignIn(s.DB, req.Lessonid)
	if err != nil {
		return nil, err
	}

	return &live.SelectSignInResp{Resp: &common.Resp{Data: sinfo}}, nil
}

// DelSign implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) DelSign(ctx context.Context, req *live.DelSignInReq) (resp *live.DelSignInResp, err error) {
	lid, err := strconv.Atoi(req.Lessonid)
	if err != nil {
		return nil, err
	}
	linfo, err := dao.SelectLessonById(s.DB, lid)
	if err != nil {
		return nil, err
	}

	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！！！你不是老师")
	} else if username != linfo.Teacher {
		return nil, errors.New("权限不够！！！你不是当前课程老师")
	}

	err = dao.RemoveSignIn(s.DB, req.Lessonid)
	if err != nil {
		return nil, err
	}

	return &live.DelSignInResp{Resp: &common.Resp{Data: "success"}}, nil
}

// RollCallInRandom implements the LiveServiceImpl interface.
func (s *LiveServiceImpl) RollCallInRandom(ctx context.Context, req *live.RollCallInRandomReq) (resp *live.RollCallInRandomResp, err error) {
	lid, err := strconv.Atoi(req.LessonId)
	if err != nil {
		return nil, err
	}
	linfo, err := dao.SelectLessonById(s.DB, lid)
	if err != nil {
		return nil, err
	}

	info, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: req.Userid})
	if err != nil {
		return nil, err
	}

	//拿到信息
	username, auth := cut.SplitInfo(info.Resp.Data)
	if auth != "Teacher" {
		return nil, errors.New("权限不够！！！你不是老师")
	} else if username != linfo.Teacher {
		return nil, errors.New("权限不够！！！你不是当前课程老师")
	}

	randomIndex := rand.Intn(len(linfo.StudentID))

	stuinfo, err := s.userCli.GetUserInfo(ctx, &user.GetUserInfoReq{Userid: linfo.StudentID[randomIndex]})
	if err != nil {
		return nil, err
	}

	stuname, _ := cut.SplitInfo(stuinfo.Resp.Data)

	return &live.RollCallInRandomResp{Resp: &common.Resp{Data: stuname}}, nil
}
