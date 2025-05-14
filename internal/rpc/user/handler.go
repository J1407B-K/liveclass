package main

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"liveclass/idl/kitex_gen/common"
	user "liveclass/idl/kitex_gen/user"
	"liveclass/internal/rpc/user/dao"
	"liveclass/internal/utils/hash"
	"log"
	"strconv"
)

// UserServiceImpl implements the last service interface defined in the IDL.
type UserServiceImpl struct {
	DB *gorm.DB
}

// Register implements the UserServiceImpl interface.
func (s *UserServiceImpl) Register(ctx context.Context, req *user.RegisterReq) (resp *user.RegisterResp, err error) {
	if req.Auth != "Teacher" && req.Auth != "Student" {
		return nil, errors.New("unknown auth")
	}

	req.Password, err = hash.HashedLock(req.Password)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	err = dao.SaveUser(s.DB, req)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return &user.RegisterResp{
		Resp: &common.Resp{
			Data: req.Username,
		},
	}, nil
}

// Login implements the UserServiceImpl interface.
func (s *UserServiceImpl) Login(ctx context.Context, req *user.LoginReq) (resp *user.LoginResp, err error) {
	//获得userinfo
	userinfo, err := dao.SelectUserByUsername(s.DB, req.Username)
	if err != nil {
		return nil, err
	}

	//比较是否一致
	err = hash.CompareHashAndPassword(userinfo.Password, req.Password)
	if err != nil {
		return nil, err
	}

	resp = &user.LoginResp{
		Resp: &common.Resp{
			Data: strconv.Itoa(userinfo.Userid),
		},
	}
	return resp, nil
}

// GetUserInfo implements the UserServiceImpl interface.
// 主要是鉴权之后rpc使用(id搜索)
func (s *UserServiceImpl) GetUserInfo(ctx context.Context, req *user.GetUserInfoReq) (resp *user.GetUserInfoResp, err error) {
	userinfo, err := dao.SelectUser(s.DB, req.Userid)
	if err != nil {
		return nil, err
	}

	return &user.GetUserInfoResp{
		Resp: &common.Resp{
			Data: userinfo.Username + "/" + userinfo.Auth},
	}, nil
}

// GetUserInfoByname implements the UserServiceImpl interface.
// 主要是对外提供api
func (s *UserServiceImpl) GetUserInfoByname(ctx context.Context, req *user.GetUserInfoByNameReq) (resp *user.GetUserInfoByNameResp, err error) {
	userinfo, err := dao.SelectUserByUsername(s.DB, req.Username)
	if err != nil {
		return nil, err
	}

	return &user.GetUserInfoByNameResp{
		Resp: &common.Resp{
			Data: userinfo.Username + "/" + userinfo.Auth},
	}, nil
}
