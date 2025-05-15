package dao

import (
	"gorm.io/gorm"
	"liveclass/internal/rpc/live/model"
)

func SaveLesson(db *gorm.DB, livename, desc, username, code string) error {
	l := model.Lesson{
		Name:        livename,
		Description: desc,
		Teacher:     username,
		Code:        code,
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
	err := db.Where("teacher = ?", username).Find(lessons).Error
	if err != nil {
		return nil, err
	}
	return lessons, nil
}

func SelectLessonByNandT(db *gorm.DB, name, teacher string) (model.Lesson, error) {
	var lesson model.Lesson
	err := db.Where("teacher = ? and name = ?", teacher, name).Find(&lesson).Error
	if err != nil {
		return model.Lesson{}, err
	}
	return lesson, nil
}
