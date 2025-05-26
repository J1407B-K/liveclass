package dao

import (
	"gorm.io/gorm"
	"liveclass/internal/rpc/webrtc_live/model"
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
