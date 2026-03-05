package main

import (
	"context"
	"errors"
	"liveclass/idl/kitex_gen/common"
	user "liveclass/idl/kitex_gen/user"
	"liveclass/internal/api/utils/bcrypt"
	"liveclass/internal/rpc/user/code"
	"liveclass/internal/rpc/user/dao"
	"liveclass/internal/rpc/user/model"
	"log"
)

// UserServiceImpl implements the last service interface defined in the IDL.
type UserServiceImpl struct {
	Manager *dao.DBManager
}

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

	sid, err := s.Manager.SaveUser(model.RegisterParam{
		Username:     req.Username,
		PasswordHash: passwordHash,
		Auth:         req.Auth,
		Status:       code.UserNormal,
	})
	if err != nil {
		log.Println(err)
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
	userinfo, err := s.Manager.SelectUserByUsername(req.Username)
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
	userinfo, err := s.Manager.SelectUser(req.Userid)
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
	userinfo, err := s.Manager.SelectUserByUsername(req.Username)
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
