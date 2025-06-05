package model

import "time"

type SignIn struct {
	ID            int         `gorm:"primary_key;auto_increment"`
	LessonId      string      `gorm:"not null"`
	AllUserId     StringArray `gorm:"type:json"`
	AlreadyUserId StringArray `gorm:"type:json"`
	CloseTime     time.Time   `gorm:"size:255"`
}
