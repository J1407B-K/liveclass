package model

import "time"

type Message struct {
	Content string `json:"content" bson:"content"`
}

type ShowMessage struct {
	MessageID string    `json:"message_id"`
	LessonID  int64     `json:"lesson_id"`
	SenderID  int64     `json:"sender_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}
