package model

import "time"

// TranscriptEvent is the append-oriented fact source for an agent session.
// Payload contains observable structured data only; private model reasoning is
// never persisted.
type TranscriptEvent struct {
	ID              int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID          int64     `gorm:"not null;index" json:"user_id"`
	SessionID       string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_session_request_event;index:idx_session_event" json:"session_id"`
	RequestID       string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_session_request_event;index" json:"request_id"`
	EventKey        string    `gorm:"type:varchar(96);not null;uniqueIndex:uk_session_request_event" json:"event_key"`
	EventType       string    `gorm:"type:varchar(32);not null;index" json:"event_type"`
	Role            string    `gorm:"type:varchar(32);not null;default:''" json:"role,omitempty"`
	Content         string    `gorm:"type:text;not null;default:''" json:"content,omitempty"`
	Payload         string    `gorm:"type:jsonb;not null;default:'{}'" json:"payload,omitempty"`
	EstimatedTokens int       `gorm:"not null;default:0" json:"estimated_tokens"`
	CreatedAt       time.Time `gorm:"not null;index:idx_session_event,sort:asc" json:"created_at"`
}

type SummaryCheckpoint struct {
	ID                 int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID             int64     `gorm:"not null;index" json:"user_id"`
	SessionID          string    `gorm:"type:varchar(64);not null;index:idx_session_checkpoint;uniqueIndex:uk_checkpoint_range" json:"session_id"`
	Summary            string    `gorm:"type:text;not null" json:"summary"`
	ImportantFacts     string    `gorm:"type:jsonb;not null;default:'[]'" json:"important_facts"`
	Decisions          string    `gorm:"type:jsonb;not null;default:'[]'" json:"decisions"`
	UnfinishedTasks    string    `gorm:"type:jsonb;not null;default:'[]'" json:"unfinished_tasks"`
	SourceEventStartID int64     `gorm:"not null" json:"source_event_start_id"`
	SourceEventEndID   int64     `gorm:"not null;index:idx_session_checkpoint;uniqueIndex:uk_checkpoint_range" json:"source_event_end_id"`
	EstimatedTokens    int       `gorm:"not null;default:0" json:"estimated_tokens"`
	CreatedAt          time.Time `gorm:"not null;index:idx_session_checkpoint,sort:desc" json:"created_at"`
}

type AgentTraceEvent struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	RunID      string    `gorm:"type:varchar(64);not null;index:idx_trace_run" json:"run_id"`
	SessionID  string    `gorm:"type:varchar(64);not null;index" json:"session_id"`
	RequestID  string    `gorm:"type:varchar(64);not null;index" json:"request_id"`
	EventType  string    `gorm:"type:varchar(48);not null;index" json:"event_type"`
	Name       string    `gorm:"type:varchar(128);not null;default:''" json:"name,omitempty"`
	Status     string    `gorm:"type:varchar(32);not null;default:''" json:"status,omitempty"`
	Metadata   string    `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty"`
	DurationMs int64     `gorm:"not null;default:0" json:"duration_ms"`
	CreatedAt  time.Time `gorm:"not null;index:idx_trace_run,sort:asc" json:"created_at"`
}

type TaskPlan struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"not null;index" json:"user_id"`
	SessionID string    `gorm:"type:varchar(64);not null;index;uniqueIndex:uk_task_request" json:"session_id"`
	RequestID string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_task_request" json:"request_id"`
	Goal      string    `gorm:"type:text;not null" json:"goal"`
	Status    string    `gorm:"type:varchar(24);not null;index" json:"status"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

type TaskStep struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	PlanID      int64     `gorm:"not null;index;uniqueIndex:uk_plan_step" json:"plan_id"`
	StepKey     string    `gorm:"type:varchar(64);not null;uniqueIndex:uk_plan_step" json:"step_key"`
	Description string    `gorm:"type:text;not null" json:"description"`
	Status      string    `gorm:"type:varchar(24);not null;index" json:"status"`
	DependsOn   string    `gorm:"type:jsonb;not null;default:'[]'" json:"depends_on"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
}
