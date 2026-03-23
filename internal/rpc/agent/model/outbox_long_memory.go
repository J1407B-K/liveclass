package model

import "time"

type OutboxEvent struct {
	ID            int64     `gorm:"primaryKey;autoIncrement"`
	EventType     string    `gorm:"type:varchar(64);not null;index"`
	AggregateType string    `gorm:"type:varchar(64);not null"`
	AggregateID   string    `gorm:"type:varchar(64);not null"`
	BizKey        string    `gorm:"type:varchar(128);not null;uniqueIndex"`
	Payload       string    `gorm:"type:jsonb;not null"`
	Status        int32     `gorm:"not null;default:0;index"`
	LastError     string    `gorm:"type:text"`
	RetryCount    int32     `gorm:"not null;default:0"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}
