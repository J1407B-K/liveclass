package model

type SignIn struct {
	LessonId      string      `gorm:"not null"`
	AllUserId     StringArray `gorm:"type:json"`
	AlreadyUserId StringArray `gorm:"type:json"`
}
