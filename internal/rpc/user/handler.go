package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"liveclass/idl/kitex_gen/common"
	"liveclass/idl/kitex_gen/user"
	"liveclass/internal/api/utils/password"
	"liveclass/internal/rpc/user/code"
	"liveclass/internal/rpc/user/dao"
	"liveclass/internal/rpc/user/model"
	"log"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
)

// UserServiceImpl implements the last service interface defined in the IDL.
type UserServiceImpl struct {
	DBManager *dao.DBManager

	sfUser singleflight.Group
}

var ErrUserNotExist = errors.New("user not exist")

// Register implements the UserServiceImpl interface.
func (s *UserServiceImpl) Register(ctx context.Context, req *user.RegisterReq) (resp *user.RegisterResp, err error) {
	if req.Auth != "Teacher" && req.Auth != "Student" {
		return nil, errors.New("unknown auth")
	}

	passwordHash, err := password.HashPassword(req.Password)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	sid, err := s.DBManager.CreateUser(model.RegisterParam{
		Username:     req.Username,
		PasswordHash: passwordHash,
		Auth:         req.Auth,
		Status:       code.UserNormal,
	})
	if err != nil {
		log.Println(err)
		return nil, err
	}

	_ = s.DBManager.RDB.Del(ctx, fmt.Sprintf("user:info:%d", sid)).Err()

	err = s.DBManager.AddBloom(ctx, sid)
	if err != nil {
		return nil, err
	}

	return &user.RegisterResp{
		Resp: &common.Resp{
			Code: code.Success,
			Msg:  "success",
			Data: &common.Data{UserInfo: &common.User{
				UserID:   sid,
				UserName: req.Username,
				Auth:     req.Auth,
				Status:   code.UserNormal,
			},
			},
		},
	}, nil
}

// Login implements the UserServiceImpl interface.
func (s *UserServiceImpl) Login(ctx context.Context, req *user.LoginReq) (resp *user.LoginResp, err error) {
	userinfo, err := s.DBManager.SelectUserByUsername(req.Username)
	if err != nil {
		return nil, err
	}

	ok, err := password.VerifyPassword(req.Password, userinfo.PasswordHash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("wrong password")
	}

	resp = &user.LoginResp{
		Resp: &common.Resp{
			Code: code.Success,
			Msg:  "success",
			Data: &common.Data{UserInfo: &common.User{
				UserID:   userinfo.UserID,
				UserName: userinfo.Username,
				Auth:     userinfo.Auth,
				Status:   userinfo.Status,
			},
			},
		},
	}
	return resp, nil
}

// GetUserInfo implements the UserServiceImpl interface.
// 主要是鉴权之后rpc使用(id搜索)
func (s *UserServiceImpl) GetUserInfo(ctx context.Context, req *user.GetUserInfoReq) (resp *user.GetUserInfoResp, err error) {
	userinfo, err := s.ensureUserExists(ctx, req.Userid)
	if err != nil {
		return nil, err
	}

	resp = &user.GetUserInfoResp{
		Resp: &common.Resp{
			Code: code.Success,
			Msg:  "success",
			Data: &common.Data{UserInfo: &common.User{
				UserID:   userinfo.UserID,
				UserName: userinfo.Username,
				Auth:     userinfo.Auth,
				Status:   userinfo.Status,
			},
			},
		},
	}
	return resp, nil
}

// GetUserInfoByname implements the UserServiceImpl interface.
func (s *UserServiceImpl) GetUserInfoByname(ctx context.Context, req *user.GetUserInfoByNameReq) (resp *user.GetUserInfoByNameResp, err error) {
	userinfo, err := s.DBManager.SelectUserByUsername(req.Username)
	if err != nil {
		return nil, err
	}

	resp = &user.GetUserInfoByNameResp{
		Resp: &common.Resp{
			Code: code.Success,
			Msg:  "success",
			Data: &common.Data{UserInfo: &common.User{
				UserID:   userinfo.UserID,
				UserName: userinfo.Username,
				Auth:     userinfo.Auth,
				Status:   userinfo.Status,
			},
			},
		},
	}
	return resp, nil
}

func (s *UserServiceImpl) ensureUserExists(ctx context.Context, userid int64) (*model.User, error) {
	maybe, err := s.DBManager.BloomMaybeUser(ctx, userid)
	if err == nil && !maybe {
		return nil, ErrUserNotExist
	}

	u, err := s.getUserCached(ctx, userid)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotExist
		}
		return nil, err
	}
	return u, nil
}

func (s *UserServiceImpl) getUserCached(ctx context.Context, userid int64) (*model.User, error) {
	cacheKey := fmt.Sprintf("user:info:%d", userid)

	if s.DBManager.RDB != nil {
		b, err := s.DBManager.RDB.Get(ctx, cacheKey).Bytes()
		if err == nil && len(b) > 0 {
			var u model.User
			if jsonErr := json.Unmarshal(b, &u); jsonErr == nil {
				return &u, nil
			}
			_ = s.DBManager.RDB.Del(ctx, cacheKey).Err()
		}
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil, err
		}
	}

	v, err, _ := s.sfUser.Do(cacheKey, func() (interface{}, error) {
		u, err := s.DBManager.SelectUser(userid)
		if err != nil {
			return nil, err
		}
		if s.DBManager.RDB != nil {
			if b, jerr := json.Marshal(u); jerr == nil {
				ttl := 5*time.Minute + time.Duration(rand.Intn(30))*time.Second
				_ = s.DBManager.RDB.Set(ctx, cacheKey, b, ttl).Err()
			}
		}
		return u, nil
	})
	if err != nil {
		return nil, err
	}
	u, ok := v.(*model.User)
	if !ok || u == nil {
		return nil, errors.New("invalid user type")
	}
	return u, nil
}
