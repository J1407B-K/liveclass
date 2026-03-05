package model

import "time"

type WebrtcLesson struct {
	LessonId    int64  `gorm:"primaryKey;not null" json:"lessonId"`
	Name        string `gorm:"type:varchar(255);not null" json:"name"`
	Description string `gorm:"type:varchar(255);not null" json:"description"`

	TeacherName string `gorm:"type:varchar(128);not null" json:"teacher"`
	TeacherUID  int64  `gorm:"not null;index" json:"teacherUid"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
