package model

type ExcalidrawDoc struct {
	ID       int       `gorm:"primary_key;auto_increment" json:"id"`
	LessonId int64     `gorm:"not null" json:"lesson_id"`
	DocType  string    `gorm:"type:varchar(50)" json:"doctype"`
	Version  int       `gorm:"not null"                    json:"version"`
	Source   string    `gorm:"type:varchar(255)"           json:"source"`
	Elements []Element `gorm:"type:json;serializer:json"   json:"elements"`
	AppState AppState  `gorm:"type:json;serializer:json"   json:"appState"`
	Files    []File    `gorm:"type:json;serializer:json"   json:"files"`
}
