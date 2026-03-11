package model

import "time"

type Message struct {
	ID     int64 `gorm:"primaryKey;autoIncrement"`
	UserID int64 `gorm:"not null;index"`

	ConvID    string `gorm:"type:varchar(64);not null;uniqueIndex:uk_conv_req_role;index:idx_conv_created"`
	RequestID string `gorm:"type:varchar(64);not null;uniqueIndex:uk_conv_req_role"`
	Role      string `gorm:"type:varchar(32);not null;uniqueIndex:uk_conv_req_role"`

	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"not null;index:idx_conv_created,sort:desc"`
}
