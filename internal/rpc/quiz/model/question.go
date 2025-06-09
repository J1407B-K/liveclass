package model

import "time"

type Question struct {
	ID         int         `json:"question_id" gorm:"primary_key;auto_increment"`
	LessonId   int         `json:"lesson_id" gorm:"not null"`
	Content    string      `json:"content" gorm:"not null;size:255"`
	OptionsNum int         `json:"options_num" gorm:"not null"`
	Options    StringArray `json:"options" gorm:"type:json"`
	Answer     string      `json:"answer" gorm:"not null;size:255"`
	TeacherId  int         `json:"teacher_id" gorm:"not null;size:255"`
	CloseTime  time.Time   `gorm:"size:255"`
}
