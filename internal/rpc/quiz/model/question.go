package model

type Question struct {
	ID       int      `json:"question_id" gorm:"primary_key;auto_increment"`
	LessonId int      `json:"lesson_id" gorm:"not null"`
	Content  string   `json:"content" gorm:"not null;size:255"`
	Options  []string `json:"options" gorm:"type:json"`
	Answer   string   `json:"answer" gorm:"not null;size:255"`
}
