package model

import "github.com/cloudwego/eino/schema"

type TemplateConfig struct {
	FormatType schema.FormatType
	Templates  []schema.MessagesTemplate
}
