package dao

import (
	"gorm.io/gorm"
	"liveclass/idl/kitex_gen/user"
	"liveclass/internal/rpc/user/model"
)

func SaveUser(db *gorm.DB, req *user.RegisterReq) error {
	// 创建用户
	u := model.User{
		Username: req.Username,
		Password: req.Password,
		Auth:     req.Auth,
	}

	if err := db.Create(&u).Error; err != nil {
		return err
	}
	return nil
}

func SelectUser(db *gorm.DB, k string) (*model.User, error) {
	var u model.User

	err := db.Where("userid = ?", k).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func SelectUserByUsername(db *gorm.DB, username string) (*model.User, error) {
	var u model.User

	err := db.Where("username = ?", username).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}
