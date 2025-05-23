package model

import "time"

type Message struct {
	RoomID    string    `bson:"room_id"`
	Sender    string    `bson:"sender"`
	Receiver  string    `bson:"receiver"`
	Content   string    `bson:"content"`
	Timestamp time.Time `bson:"timestamp"`
}
