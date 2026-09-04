package model

import "time"

type Message struct {
	Content         string `json:"content" bson:"content"`
	ClientMessageID string `json:"client_message_id,omitempty" bson:"client_message_id,omitempty"`
}

type ShowMessage struct {
	MessageID       string    `json:"message_id"`
	ClientMessageID string    `json:"client_message_id,omitempty"`
	LessonID        int64     `json:"lesson_id"`
	SenderID        int64     `json:"sender_id"`
	Content         string    `json:"content"`
	CreatedAt       time.Time `json:"created_at"`
}

type MessageAck struct {
	Type            string `json:"type"`
	ClientMessageID string `json:"client_message_id,omitempty"`
	MessageID       string `json:"message_id,omitempty"`
	DeliveryStatus  string `json:"delivery_status,omitempty"`
	Error           string `json:"error,omitempty"`
}

type ResumeStatus struct {
	Type           string `json:"type"`
	AfterMessageID string `json:"after_message_id,omitempty"`
	Recovered      int    `json:"recovered"`
	Truncated      bool   `json:"truncated,omitempty"`
	Error          string `json:"error,omitempty"`
}
