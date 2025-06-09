package dao

import (
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"liveclass/idl/kitex_gen/quiz"
	"liveclass/internal/rpc/quiz/model"
	"strconv"
	"time"
)

func SaveQuestion(db *gorm.DB, lessonId int, teacherId int, now time.Time, req *quiz.CreateQuestionReq) error {
	question := model.Question{
		LessonId:   lessonId,
		Content:    req.Content,
		OptionsNum: int(req.OptionsNum),
		Options:    req.Options,
		Answer:     req.Answer,
		TeacherId:  teacherId,
		CloseTime:  now.Add(time.Duration(req.Duration) * time.Second),
	}

	return db.Create(&question).Error
}

func CreateAnswer(db *gorm.DB, questionId, optionNums int, right string, now time.Time, duration int32) error {
	arrayStr := make([]string, optionNums)
	for i := 0; i < optionNums; i++ {
		arrayStr[i] = "0"
	}

	answer := model.Answer{
		QuestionId:      questionId,
		Right:           right,
		OptionNums:      optionNums,
		SelectedOptions: arrayStr,
		AnsweredId:      []string{},
		CloseTime:       now.Add(time.Duration(duration) * time.Second),
	}

	return db.Create(&answer).Error
}

func SelectAnswer(db *gorm.DB, questionId int) (model.StringArray, model.StringArray, error) {
	var ans model.Answer
	err := db.Where("question_id = ?", questionId).First(&ans).Error
	if err != nil {
		return nil, nil, err
	}

	return ans.SelectedOptions, ans.AnsweredId, nil
}

func GetQuestionId(db *gorm.DB, Content string) (int, error) {
	var q model.Question

	err := db.Where(&model.Question{Content: Content}).First(&q).Error
	if err != nil {
		return 0, err
	}
	return q.ID, nil
}

func GetQuestion(db *gorm.DB, qId int) (*model.Question, error) {
	var q model.Question
	err := db.Where("id = ?", qId).First(&q).Error
	if err != nil {
		return nil, err
	}
	return &q, nil
}

func AddAnswerWithDefaultTx(db *gorm.DB, userID, questionID int, selectedAns string) error {
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

	var ans model.Answer

	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("question_id = ?", questionID).
		First(&ans).Error; err != nil {
		tx.Rollback()
		return err
	}

	switch selectedAns {
	case "A":
		i, _ := strconv.Atoi(ans.SelectedOptions[0])
		ans.SelectedOptions[0] = strconv.Itoa(i + 1)
	case "B":
		i, _ := strconv.Atoi(ans.SelectedOptions[1])
		ans.SelectedOptions[1] = strconv.Itoa(i + 1)
	case "C":
		i, _ := strconv.Atoi(ans.SelectedOptions[2])
		ans.SelectedOptions[2] = strconv.Itoa(i + 1)
	case "D":
		i, _ := strconv.Atoi(ans.SelectedOptions[3])
		ans.SelectedOptions[3] = strconv.Itoa(i + 1)
	default:
		tx.Rollback()
		return errors.New("invalid option")
	}

	//保存答题id(一人一次)
	ans.AnsweredId = append(ans.AnsweredId, strconv.Itoa(userID))

	//保存
	if err := tx.Save(&ans).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func DelQuestionAndAnswer(db *gorm.DB, questionId int) error {
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

	if err := tx.Where("id = ?", questionId).Delete(&model.Question{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Where("question_id = ?", questionId).Delete(&model.Answer{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func GetQustionByLesson(db *gorm.DB, lessonId int) ([]model.Question, error) {
	var questions []model.Question
	err := db.Where("lesson_id = ?", lessonId).Find(&questions).Error
	if err != nil {
		return nil, err
	}
	return questions, nil
}

func CheckCloseTime(db *gorm.DB, questionId int) error {
	var q model.Question
	err := db.Where("id = ?", questionId).First(&q).Error
	if err != nil {
		return err
	}

	now := time.Now()
	delta := now.Sub(q.CloseTime)
	if delta >= 0 {
		err = DelQuestionAndAnswer(db, questionId)
		if err != nil {
			return err
		}
		return errors.New("close")
	}
	return nil
}
