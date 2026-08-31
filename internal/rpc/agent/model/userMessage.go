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

	// SkillAdvice 由 Advisor 节点填充，表示当前意图对应的技能类型和执行指引
	SkillAdvice *SkillAdvice `json:"skill_advice,omitempty"`
}

// SkillAdvice 是 Advisor 节点的输出：技能类型列表 + 合并后的流程指引
type SkillAdvice struct {
	// Skills 命中的技能类型列表，支持多 tool 组合
	Skills []string `json:"skills"`
	// Guidance 注入给 React Agent 的流程指引（从 .md 文件加载并合并）
	Guidance string `json:"guidance"`
}
