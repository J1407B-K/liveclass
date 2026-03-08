package model

import "time"

type Message struct {
	Content string `json:"content" bson:"content"`
}

type ShowMessage struct {
	LessonID  int64     `json:"lesson_id"`
	Sender    int64     `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}
