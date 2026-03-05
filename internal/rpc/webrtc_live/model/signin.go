package model

import "time"

type SignIn struct {
	ID            int64     `gorm:"primary_key;auto_increment"`
	LessonId      int64     `gorm:"not null"`
	AllUserId     []int64   `gorm:"type:json"`
	AlreadyUserId []int64   `gorm:"type:json"`
	CloseTime     time.Time `gorm:"size:255"`
}
