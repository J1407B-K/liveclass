package dao

import (
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"liveclass/internal/rpc/webrtc_live/model"
	"strings"
)

func CreateLesson(db *gorm.DB, name, desc, teacher, userid string) error {
	var lesson = model.WebrtcLesson{
		Name:        name,
		Description: desc,
		Teacher:     teacher,
		StudentID:   []string{userid},
	}
	return db.Create(&lesson).Error
}

func DelLesson(db *gorm.DB, lessonid int) error {
	return db.Where("lesson_id = ?", lessonid).Unscoped().Delete(&model.WebrtcLesson{}).Error
}

func SelectLesson(db *gorm.DB, lessonid int) (*model.WebrtcLesson, error) {
	var lesson model.WebrtcLesson
	if err := db.Where("lesson_id = ?", lessonid).First(&lesson).Error; err != nil {
		return nil, err
	}
	return &lesson, nil
}

func SelectLessonByNandT(db *gorm.DB, name, teacher string) (*model.WebrtcLesson, error) {
	var lesson model.WebrtcLesson
	if err := db.Where("name = ? and teacher = ?", name, teacher).First(&lesson).Error; err != nil {
		return nil, err
	}
	return &lesson, nil
}

func ChangeUserToLesson(db *gorm.DB, lessonid int, stuid, options string) error {
	tx := db.Begin()
	var lesson model.WebrtcLesson

	if err := tx.Where("lesson_id = ?", lessonid).First(&lesson).Error; err != nil {
		tx.Rollback()
		return err
	}

	if options == "add" {
		lesson.StudentID = append(lesson.StudentID, stuid)
		err := tx.Save(&lesson).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		var newStudentId []string
		for _, id := range lesson.StudentID {
			if id == stuid {
				continue
			}
			newStudentId = append(newStudentId, id)
		}
		lesson.StudentID = newStudentId

		err := tx.Save(&lesson).Error
		if err != nil {
			tx.Rollback()
			return err
		}

		err = tx.Commit().Error
		if err != nil {
			tx.Rollback()
			return err
		}
	}
	return nil
}

func CheckStudentInLesson(db *gorm.DB, studentId, lessonid string) (string, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return "", tx.Error
	}

	// 确保结束时提交或回滚
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var lesson model.WebrtcLesson

	if err := tx.
		Where("lesson_id = ?", lessonid).
		First(&lesson).Error; err != nil {
		tx.Rollback()
		return "", err
	}

	for _, id := range lesson.StudentID {
		if id == studentId {
			tx.Commit()
			return "in", nil
		}
	}

	return "not_in", nil
}

func CreateSignIn(db *gorm.DB, lessonId string, alluserid []string) error {
	tx := db.Begin()
	if tx.Error != nil {
		tx.Rollback()
		return tx.Error
	}

	var search model.SignIn
	err := tx.Where("lesson_id = ?", lessonId).First(&search).Error
	if err == nil {
		tx.Rollback()
		return errors.New("你已创建过签到")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return err
	}

	s := model.SignIn{LessonId: lessonId, AllUserId: alluserid, AlreadyUserId: []string{}}

	if err := tx.Create(&s).Error; err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

func StuSignIn(db *gorm.DB, lessonId, userId string) (string, error) {
	tx := db.Begin()
	if tx.Error != nil {
		return "", tx.Error
	}

	// 确保结束时提交或回滚
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var SignIn model.SignIn
	err := tx.Where("lesson_id = ?", lessonId).First(&SignIn).Error
	if SignIn.LessonId != lessonId {
		tx.Rollback()
		return "", errors.New("课程不匹配")
	}

	if err != nil {
		tx.Rollback()
		return "", err
	}

	for _, id := range SignIn.AlreadyUserId {
		if id == userId {
			tx.Rollback()
			return "<UNK>", errors.New("你已经签过到了")
		}
	}

	SignIn.AlreadyUserId = append(SignIn.AlreadyUserId, userId)
	err = tx.Where("lesson_id = ?", lessonId).Save(&SignIn).Error
	if err != nil {
		tx.Rollback()
		return "", err
	}
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return "", err
	}

	return "success", nil
}

func SelectSignIn(db *gorm.DB, lessonid string) (string, error) {
	var s model.SignIn
	if err := db.Where("lesson_id = ?", lessonid).First(&s).Error; err != nil {
		return "", err
	}

	alreadyUsers := s.AlreadyUserId

	//好查一点点hhh
	signedMap := make(map[string]struct{}, len(alreadyUsers))
	for _, uid := range alreadyUsers {
		signedMap[uid] = struct{}{}
	}

	var notAlreadyUsers []string
	for _, uid := range s.AllUserId {
		if _, ok := signedMap[uid]; !ok {
			notAlreadyUsers = append(notAlreadyUsers, uid)
		}
	}

	alreadyStr := strings.Join(alreadyUsers, "/")
	notAlreadyStr := strings.Join(notAlreadyUsers, "/")

	return fmt.Sprintf("已签到为%v, 未签到为%v", alreadyStr, notAlreadyStr), nil
}

func RemoveSignIn(db *gorm.DB, lessonId string) error {
	return db.Where("lesson_id = ?", lessonId).Unscoped().Delete(&model.SignIn{}).Error
}

func SaveWhiteBoard(db *gorm.DB, lessonid, file string) error {
	var doc model.ExcalidrawDoc

	err := json.Unmarshal([]byte(file), &doc)
	if err != nil {
		return err
	}

	doc.LessonId = lessonid

	err = db.Create(&doc).Error
	if err != nil {
		return err
	}
	return nil
}

func GetWhiteBoardNew(db *gorm.DB, lessonid string) (*model.ExcalidrawDoc, error) {
	var docs model.ExcalidrawDoc

	err := db.Where("lesson_id = ?", lessonid).First(&docs).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &docs, nil
}

func RaiseHand(db *gorm.DB, lessonid int, stuid string) error {
	tx := db.Begin()

	var l model.WebrtcLesson
	err := tx.Where("lesson_id = ?", lessonid).First(&l).Error
	if err != nil {
		tx.Rollback()
		return err
	}

	l.RaiseStuId = append(l.RaiseStuId, stuid)

	err = tx.Save(&l).Error
	if err != nil {
		tx.Rollback()
		return err
	}
	err = tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
	}
	return nil
}

func ApproveHand(db *gorm.DB, l model.WebrtcLesson, stuid string) error {
	tx := db.Begin()

	for i := 0; i < len(l.RaiseStuId); i++ {
		if l.RaiseStuId[i] == stuid {
			l.RaiseStuId = append(l.RaiseStuId[:i], l.RaiseStuId[i+1:]...)

			err := tx.Save(&l).Error
			if err != nil {
				tx.Rollback()
				return err
			}

			err = tx.Commit().Error
			if err != nil {
				tx.Rollback()
				return err
			}

			return nil
		}
	}
	tx.Rollback()
	return errors.New("该学生没有举手！")
}

func removeValueInArray(array []string, value string) []string {
	for i, v := range array {
		if v == value {
			return append(array[:i], array[i+1:]...)
		}
	}
	return nil
}
