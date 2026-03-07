package model

import "time"

type Question struct {
	ID        int64     `json:"question_id" gorm:"primaryKey;autoIncrement"`
	LessonID  int64     `json:"lesson_id" gorm:"not null;index"`
	Content   string    `json:"content" gorm:"not null;size:255"`
	Options   []string  `json:"options" gorm:"serializer:json"`
	Answer    string    `json:"answer" gorm:"not null;size:255"`
	TeacherID int64     `json:"teacher_id" gorm:"not null;index"`
	CloseTime time.Time `json:"close_time" gorm:"not null;index"`
}
