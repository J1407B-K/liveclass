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

// SkillAdvice 是 Advisor 节点的输出：技能类型 + 针对本次任务的流程指引
type SkillAdvice struct {
	// Skill 技能类型：student_qa / lesson_plan / quiz_help / lesson_summary / general
	Skill string `json:"skill"`
	// Guidance 注入给 React Agent 的流程指引（1-3句，告诉 agent 怎么做）
	Guidance string `json:"guidance"`
}
