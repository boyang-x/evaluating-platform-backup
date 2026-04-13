package model

import (
	"time"

	"github.com/google/uuid"
)

// BillingRecord 计费记录（每次工具调用产生一条）
type BillingRecord struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	UserID       uuid.UUID  `json:"user_id" db:"user_id"`
	AssessmentID *uuid.UUID `json:"assessment_id,omitempty" db:"assessment_id"`
	ToolName     string     `json:"tool_name" db:"tool_name"`
	ToolCategory string     `json:"tool_category" db:"tool_category"`
	CallCount    int        `json:"call_count" db:"call_count"`
	TokensUsed   int        `json:"tokens_used" db:"tokens_used"`
	Amount       float64    `json:"amount" db:"amount"` // 费用（元）
	AssetID      *uuid.UUID `json:"asset_id,omitempty" db:"asset_id"`
	ExpertID     *uuid.UUID `json:"expert_id,omitempty" db:"expert_id"`
	ExpertShare  float64    `json:"expert_share" db:"expert_share"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// BalanceTransaction 余额变动记录
type BalanceTransaction struct {
	ID            uuid.UUID `json:"id" db:"id"`
	UserID        uuid.UUID `json:"user_id" db:"user_id"`
	Type          string    `json:"type" db:"type"` // recharge | deduct | refund
	Amount        float64   `json:"amount" db:"amount"`
	BalanceBefore float64   `json:"balance_before" db:"balance_before"`
	BalanceAfter  float64   `json:"balance_after" db:"balance_after"`
	Description   string    `json:"description" db:"description"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}
