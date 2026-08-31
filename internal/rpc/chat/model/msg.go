package model

import "time"

type Message struct {
	MessageID string    `json:"message_id" bson:"message_id"`
	LessonID  int64     `json:"lesson_id" bson:"lesson_id"`
	SenderID  int64     `json:"sender_id" bson:"sender_id"`
	Content   string    `json:"content" bson:"content"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}
