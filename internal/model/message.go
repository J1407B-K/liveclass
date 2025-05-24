package model

import "time"

type Message struct {
	LessonID  string    `json:"lesson_id" bson:"lesson_id"`
	Sender    string    `json:"sender" bson:"sender"`
	Content   string    `json:"content" bson:"content"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}
