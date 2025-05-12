package model

type Lesson struct {
	Name string `json:"name" gorm:"size:255"`
	Code string `json:"code" gorm:"size:255"`
}
