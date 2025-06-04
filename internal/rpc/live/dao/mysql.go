package dao

import (
	"errors"
	"fmt"
	"gorm.io/gorm"
	"liveclass/internal/rpc/live/model"
	"strings"
)

func SaveLesson(db *gorm.DB, livename, desc, username, code, teacherid string) error {
	l := model.Lesson{
		Name:        livename,
		Description: desc,
		Teacher:     username,
		Code:        code,
		StudentID:   []string{teacherid},
	}

	if err := db.Create(&l).Error; err != nil {
		return err
	}
	return nil
}

func DeleteLesson(db *gorm.DB, livename, username string) error {
	//直接硬删除
	err := db.Unscoped().Where("name = ? and teacher = ?", livename, username).Delete(&model.Lesson{}).Error
	if err != nil {
		return err
	}
	return nil
}

func SelectLessonByTeacher(db *gorm.DB, username string) ([]model.Lesson, error) {
	var lessons []model.Lesson
	err := db.Where("teacher = ?", username).Find(&lessons).Error
	if err != nil {
		return nil, err
	}
	return lessons, nil
}

func SelectLessonByNandT(db *gorm.DB, name, teacher string) (model.Lesson, error) {
	var lesson model.Lesson
	err := db.Where("teacher = ? and name = ?", teacher, name).First(&lesson).Error
	if err != nil {
		return model.Lesson{}, err
	}
	return lesson, nil
}

func SelectLessonById(db *gorm.DB, id int) (*model.Lesson, error) {
	var lesson model.Lesson
	err := db.Where("lesson_id = ?", id).Find(&lesson).Error
	if err != nil {
		return nil, err
	}
	return &lesson, nil
}

func ChangeUserToLesson(db *gorm.DB, studentId, lessonName, teacher, option string) error {
	// 开启事务（默认使用 MySQL 的 autocommit=false 模式）
	tx := db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	// 确保结束时提交或回滚
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	var lesson model.Lesson

	if err := tx.
		Where("name = ? and teacher = ?", lessonName, teacher).
		First(&lesson).Error; err != nil {
		tx.Rollback()
		return err
	}

	if option == "add" {
		lesson.StudentID = append(lesson.StudentID, studentId)
		err := tx.Save(&lesson).Error
		if err != nil {
			tx.Rollback()
			return err
		}
	} else {
		var newStudentId []string
		for _, id := range lesson.StudentID {
			if id == studentId {
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
	}
	err := tx.Commit().Error
	if err != nil {
		tx.Rollback()
		return err
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

	var lesson model.Lesson

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

func SelectSignIn(db *gorm.DB, lessonId string) (string, error) {
	var s model.SignIn
	if err := db.Where("lesson_id = ?", lessonId).First(&s).Error; err != nil {
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
