package model

import (
	"time"

	"github.com/google/uuid"
)

// ChatSessionState 会话状态
type ChatSessionState string

const (
	ChatStateIdle             ChatSessionState = "idle"
	ChatStateCollectingIntent ChatSessionState = "collecting_intent"
	ChatStateConfirmingPlan   ChatSessionState = "confirming_plan"
	ChatStateRunning          ChatSessionState = "running"
	ChatStateCompleted        ChatSessionState = "completed"
	ChatStateFailed           ChatSessionState = "failed"
)

// ChatSession 对话会话
type ChatSession struct {
	ID           uuid.UUID        `json:"id"`
	UserID       uuid.UUID        `json:"user_id"`
	Title        string           `json:"title"`
	State        ChatSessionState `json:"state"`
	AssessmentID *uuid.UUID       `json:"assessment_id,omitempty"`
	TargetInfo   map[string]any   `json:"target_info"`
	PlanInfo     map[string]any   `json:"plan_info"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// ChatMessageRole 消息角色
type ChatMessageRole string

const (
	RoleUser      ChatMessageRole = "user"
	RoleAssistant ChatMessageRole = "assistant"
	RoleSystem    ChatMessageRole = "system"
	RoleToolEvent ChatMessageRole = "tool_event"
)

// ChatMessage 对话消息
type ChatMessage struct {
	ID        uuid.UUID       `json:"id"`
	SessionID uuid.UUID       `json:"session_id"`
	Role      ChatMessageRole `json:"role"`
	Content   string          `json:"content"`
	// Metadata 结构化元数据，用于前端渲染特殊卡片
	// card_type: "plan_confirm" | "progress" | "report" | "skill_launch" | "text"
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}
