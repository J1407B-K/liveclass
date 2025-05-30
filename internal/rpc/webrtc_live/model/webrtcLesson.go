package model

type WebrtcLesson struct {
	LessonId    int         `json:"lessonId" gorm:"primary_key;auto_increment"`
	Name        string      `json:"name" gorm:"size:255;not null"`
	Description string      `json:"description" gorm:"size:255;not null"`
	Teacher     string      `json:"teacher" gorm:"not null"`
	StudentID   StringArray `json:"studentId" gorm:"type:json"`

	RaiseStuId StringArray `json:"raiseStuId" gorm:"type:json"`
}
