package model

import "github.com/cloudwego/eino/schema"

type UserMessage struct {
	ID      int64             `json:"id"`
	Query   string            `json:"query"`
	Facts   string            `json:"facts"`
	Profile string            `json:"profile"`
	History []*schema.Message `json:"history"`
}
