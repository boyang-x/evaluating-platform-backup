package model

import (
	"time"

	"github.com/google/uuid"
)

// AuxiliaryLLMConfig 辅助 LLM 配置（按用户维度持久化）
type AuxiliaryLLMConfig struct {
	ID        uuid.UUID `json:"id" db:"id"`
	UserID    uuid.UUID `json:"user_id" db:"user_id"`
	BaseURL   string    `json:"base_url" db:"base_url"`
	APIKey    string    `json:"-" db:"api_key"` // 不暴露给前端 GET 响应
	Model     string    `json:"model" db:"model"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
