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

	err := db.Create(&question).Error
	if err != nil {
		return err
	}
	return nil
}
