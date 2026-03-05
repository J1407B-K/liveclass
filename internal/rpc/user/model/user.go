package model

import "time"

type User struct {
	UserID int64 `json:"userId" gorm:"primaryKey"`

	Username     string `json:"username" gorm:"type:varchar(64);not null;uniqueIndex:idx_user_username"`
	PasswordHash string `json:"-" gorm:"type:char(60);not null"`

	Auth   string `json:"auth" gorm:"type:varchar(16);not null;default:'student'"`
	Status int8   `json:"status" gorm:"not null;default:1"` // 1=normal,0=disabled

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
