package model

import (
	"time"

	"github.com/google/uuid"
)

// AssessmentStatus 评估任务状态
type AssessmentStatus string

const (
	StatusPending   AssessmentStatus = "pending"
	StatusRunning   AssessmentStatus = "running"
	StatusCompleted AssessmentStatus = "completed"
	StatusFailed    AssessmentStatus = "failed"
	StatusCanceled  AssessmentStatus = "canceled"
)

// Assessment 评估任务
type Assessment struct {
	ID          uuid.UUID        `json:"id" db:"id"`
	UserID      uuid.UUID        `json:"user_id" db:"user_id"`
	Name        string           `json:"name" db:"name"`
	Description string           `json:"description" db:"description"`
	Goal        string           `json:"goal" db:"goal"` // 自然语言评估目标
	TargetID    uuid.UUID        `json:"target_id" db:"target_id"`
	TemplateID  *uuid.UUID       `json:"template_id,omitempty" db:"template_id"`
	Status      AssessmentStatus `json:"status" db:"status"`
	Plan        []string         `json:"plan" db:"plan"` // LLM 规划的工具序列
	ErrorMsg    string           `json:"error_msg,omitempty" db:"error_msg"`
	CreatedAt   time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at" db:"updated_at"`
	CompletedAt *time.Time       `json:"completed_at,omitempty" db:"completed_at"`
}

// AssessmentLog 评估执行日志（每次工具调用记录）
type AssessmentLog struct {
	ID           uuid.UUID `json:"id" db:"id"`
	AssessmentID uuid.UUID `json:"assessment_id" db:"assessment_id"`
	Iteration    int       `json:"iteration" db:"iteration"`
	ToolName     string    `json:"tool_name" db:"tool_name"`
	Input        string    `json:"input" db:"input"`   // JSON
	Output       string    `json:"output" db:"output"` // JSON
	Thinking     string    `json:"thinking" db:"thinking"` // LLM 推理链
	TokensUsed   int       `json:"tokens_used" db:"tokens_used"`
	DurationMs   int64     `json:"duration_ms" db:"duration_ms"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// TargetSystem 被评估的目标系统
type TargetSystem struct {
	ID          uuid.UUID `json:"id" db:"id"`
	UserID      uuid.UUID `json:"user_id" db:"user_id"`
	Name        string    `json:"name" db:"name"`
	Type        string    `json:"type" db:"type"` // openai | agent | custom
	Endpoint    string    `json:"endpoint" db:"endpoint"`
	APIKey      string    `json:"api_key,omitempty" db:"api_key"`
	Config      string    `json:"config" db:"config"` // JSON，额外配置
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}
