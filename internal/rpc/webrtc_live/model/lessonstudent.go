package model

import "time"

type LessonStudent struct {
	ID int64 `gorm:"primaryKey;autoIncrement;index:idx_lesson_status_id,priority:3" json:"-"`

	LessonID int64 `gorm:"not null;uniqueIndex:uk_lesson_user;index:idx_lesson_status_id,priority:1" json:"lessonId"`
	UserID   int64 `gorm:"not null;uniqueIndex:uk_lesson_user" json:"userId"`

	Role   string `gorm:"type:varchar(16);not null;default:'student'" json:"role"`
	Status int8   `gorm:"not null;default:1;index:idx_lesson_status_id,priority:2" json:"status"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
