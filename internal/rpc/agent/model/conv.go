package model

import "time"

type Conversation struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	UserID    int64     `gorm:"not null;index"`
	ConvID    string    `gorm:"type:varchar(64);not null;uniqueIndex"`
	Title     string    `gorm:"type:varchar(255);not null;default:''"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}
