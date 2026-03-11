package cdc

import "encoding/json"

type debeziumEnvelope struct {
	Schema  json.RawMessage `json:"schema"`
	Payload string          `json:"payload"`
}

type FactCreatedEvent struct {
	FactID     int64  `json:"fact_id"`
	UserID     int64  `json:"user_id"`
	FactType   string `json:"fact_type"`
	Content    string `json:"content"`
	IsActive   bool   `json:"is_active"`
	SourceConv string `json:"source_conv"`
}
