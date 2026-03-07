package dao

import (
	"errors"
	"liveclass/idl/kitex_gen/quiz"
	"liveclass/internal/rpc/quiz/model"
	"time"

	"gorm.io/gorm"
)

func (m *DBManager) CreateQuestion(req *quiz.CreateQuestionReq) (*model.Question, error) {
	q := &model.Question{
		LessonID:  req.LessonId,
		Content:   req.Content,
		Options:   req.Options,
		Answer:    req.Answer,
		TeacherID: req.Userid,
		CloseTime: time.Now().Add(time.Duration(req.Duration) * time.Second),
	}

	if err := m.DB.Create(q).Error; err != nil {
		return nil, err
	}
	return q, nil
}

func (m *DBManager) GetQuestion(qid int64) (*model.Question, error) {
	var q model.Question
	if err := m.DB.Where("id = ?", qid).First(&q).Error; err != nil {
		return nil, err
	}
	return &q, nil
}

func (m *DBManager) HasAnswered(questionID, userID int64) (bool, error) {
	var cnt int64
	err := m.DB.Model(&model.Answer{}).
		Where("question_id = ? AND user_id = ?", questionID, userID).
		Count(&cnt).Error
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

func (m *DBManager) CreateUserAnswer(question *model.Question, userID int64, userAnswer string) error {
	a := &model.Answer{
		QuestionID: question.ID,
		UserID:     userID,
		Answer:     userAnswer,
		IsCorrect:  userAnswer == question.Answer,
	}
	return m.DB.Create(a).Error
}

func (m *DBManager) CountAnswersByQuestion(questionID int64) ([]model.AnswerStat, error) {
	var stats []model.AnswerStat
	err := m.DB.Model(&model.Answer{}).
		Select("answer, count(*) as count").
		Where("question_id = ?", questionID).
		Group("answer").
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (m *DBManager) DelQuestionAndAnswer(questionID int64) error {
	return m.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("question_id = ?", questionID).Delete(&model.Answer{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", questionID).Delete(&model.Question{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (m *DBManager) GetQuestionByLesson(lessonID int64) ([]model.Question, error) {
	var questions []model.Question
	err := m.DB.Where("lesson_id = ?", lessonID).
		Order("id DESC").
		Find(&questions).Error
	if err != nil {
		return nil, err
	}
	return questions, nil
}

func (m *DBManager) CheckCloseTime(questionID int64) error {
	q, err := m.GetQuestion(questionID)
	if err != nil {
		return err
	}

	if !time.Now().Before(q.CloseTime) {
		if err = m.DelQuestionAndAnswer(questionID); err != nil {
			return err
		}
		return errors.New("close")
	}
	return nil
}
