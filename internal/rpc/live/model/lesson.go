package model

type Lesson struct {
	LessonId    string `json:"lessonId" gorm:"primary_key;auto_increment"`
	Name        string `json:"name" gorm:"size:255;not null"`
	Description string `json:"description" gorm:"size:255;not null"`
	Teacher     string `json:"teacher" gorm:"not null"`
	Code        string `json:"code" gorm:"not null"`
}
