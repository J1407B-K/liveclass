package model

import "time"

type UserFact struct {
	ID          int64     `gorm:"primaryKey;autoIncrement"`
	UserID      int64     `gorm:"not null;index"`
	FactType    string    `gorm:"type:varchar(64);not null;index"`
	Content     string    `gorm:"type:text;not null"`
	Confidence  float64   `gorm:"not null;default:0"`
	SourceConv  string    `gorm:"type:varchar(64);not null;default:'';index"`
	IsActive    bool      `gorm:"not null;default:true;index"`
	IndexStatus string    `gorm:"type:varchar(32);not null;default:'pending';index"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}
