package dao

import (
	"gorm.io/gorm"
	"liveclass/idl/kitex_gen/live"
	"liveclass/internal/rpc/live/model"
)

func SaveLesson(db *gorm.DB, req *live.CreateLiveReq, username, code string) error {
	l := model.Lesson{
		Name:        req.Livename,
		Description: req.Description,
		Teacher:     username,
		Code:        code,
	}

	if err := db.Create(&l).Error; err != nil {
		return err
	}
	return nil

}
