package model

import "time"

const (
	OutboxPending    = "pending"
	OutboxPublishing = "publishing"
	OutboxPublished  = "published"
)

type OutboxState struct {
	Status        string     `json:"-" bson:"status"`
	Attempts      int        `json:"-" bson:"attempts"`
	NextAttemptAt time.Time  `json:"-" bson:"next_attempt_at"`
	LeaseOwner    string     `json:"-" bson:"lease_owner,omitempty"`
	LeaseUntil    *time.Time `json:"-" bson:"lease_until,omitempty"`
	PublishedAt   *time.Time `json:"-" bson:"published_at,omitempty"`
	LastError     string     `json:"-" bson:"last_error,omitempty"`
}

type Message struct {
	MessageID       string      `json:"message_id" bson:"message_id"`
	ClientMessageID string      `json:"client_message_id,omitempty" bson:"client_message_id,omitempty"`
	LessonID        int64       `json:"lesson_id" bson:"lesson_id"`
	SenderID        int64       `json:"sender_id" bson:"sender_id"`
	Content         string      `json:"content" bson:"content"`
	CreatedAt       time.Time   `json:"created_at" bson:"created_at"`
	Outbox          OutboxState `json:"-" bson:"outbox"`
}
