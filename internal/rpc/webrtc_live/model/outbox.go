package model

import "time"

type OutboxEvent struct {
	ID int64 `gorm:"primaryKey;autoIncrement"`

	AggregateType string `gorm:"column:aggregate_type;type:varchar(64);not null"`
	AggregateID   string `gorm:"column:aggregate_id;type:varchar(128);not null"`
	Type          string `gorm:"column:type;type:varchar(64);not null"`

	Payload string `gorm:"column:payload;type:json;not null"`

	CreatedAt time.Time `gorm:"column:created_at;type:datetime(3);autoCreateTime"`
}
