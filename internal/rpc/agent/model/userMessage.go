package model

import "github.com/cloudwego/eino/schema"

type UserMessage struct {
	ID      int64             `json:"id"`
	Lesson  int64             `json:"lesson_id"`
	Query   string            `json:"query"`
	Facts   string            `json:"facts"`
	Profile string            `json:"profile"`
	Docs    string            `json:"docs"`
	History []*schema.Message `json:"history"`
}
