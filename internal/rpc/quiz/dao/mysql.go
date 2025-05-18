package dao

import (
	"gorm.io/gorm"
	"liveclass/idl/kitex_gen/quiz"
	"liveclass/internal/rpc/quiz/model"
)

func SaveQuestion(db *gorm.DB, lessonId int, req *quiz.CreateQuestionReq) error {
	question := model.Question{
		LessonId: lessonId,
		Content:  req.Content,
		Options:  req.Options,
		Answer:   req.Answer,
	}

	return db.Create(&question).Error
}

func GetQuestionId(db *gorm.DB, Content string) (int, error) {
	var q model.Question

	err := db.Where(&model.Question{Content: Content}).First(&q).Error
	if err != nil {
		return 0, err
	}
	return q.LessonId, nil
}
func GetQuestion(db *gorm.DB, qId int) (*model.Question, error) {
	var q model.Question
	err := db.Where("id = ?", qId).First(&q).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}
