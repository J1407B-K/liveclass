package dao

import (
	"gorm.io/gorm"
	"liveclass/idl/kitex_gen/user"
	"liveclass/internal/rpc/user/model"
	"liveclass/internal/utils/cut"
)

func CreateLesson(db *gorm.DB, name, code string) error {
	// 创建用户
	u := model.Lesson{
		Name: name,
		Code: code,
	}

	if err := db.Create(&u).Error; err != nil {
		return err
	}
	return nil
}

func SaveUser(db *gorm.DB, req *user.RegisterReq) error {
	// 创建用户
	u := model.User{
		Username: req.Username,
		Password: req.Password,
		Auth:     req.Auth,
		Lessons:  cut.OutputLessons(req.Lessons),
	}

	if err := db.Create(&u).Error; err != nil {
		return err
	}
	return nil
}

// true add/false delete
func UpdateUserLessons(db *gorm.DB, k, v string, o bool) (*model.User, error) {
	var u model.User

	err := db.Where("username = ?", k).First(&u).Error
	if err != nil {
		return nil, err
	}

	l := u.Lessons
	sl := cut.CutLessons(l)
	if o {
		sl = append(sl, v)
		u.Lessons = cut.CombineLessons(sl)
	} else {
		var newS []string
		for _, e := range sl {
			if e != v {
				newS = append(newS, v)
			}
		}
		u.Lessons = cut.CombineLessons(newS)
	}

	db.Save(&u)

	return &u, nil
}

func SelectUsername(db *gorm.DB, k string) (*model.User, error) {
	var u model.User

	err := db.Where("username = ?", k).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func SelectUserid(db *gorm.DB, k int) (*model.User, error) {
	var u model.User

	err := db.Where("userid = ?", k).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}
