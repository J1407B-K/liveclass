package model

import "time"

type AnswerStat struct {
	Answer string `json:"answer"`
	Count  int64  `json:"count"`
}

type Answer struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	QuestionID int64     `gorm:"not null;uniqueIndex:idx_question_user"`
	UserID     int64     `gorm:"not null;uniqueIndex:idx_question_user"`
	Answer     string    `gorm:"not null"`
	IsCorrect  bool      `gorm:"not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}
