package billing

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// UsageRecord 用量记录
type UsageRecord struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	AssessmentID uuid.UUID `json:"assessment_id"`
	ToolName     string    `json:"tool_name"`
	ToolCategory string    `json:"tool_category"`
	CallCount    int       `json:"call_count"`
	TokensUsed   int       `json:"tokens_used"`
	Amount       float64   `json:"amount"` // 费用（元）
	CreatedAt    time.Time `json:"created_at"`
}

// Meter 用量计量器
type Meter struct {
	mu      sync.Mutex
	records []UsageRecord
}

// NewMeter 创建计量器
func NewMeter() *Meter {
	return &Meter{records: []UsageRecord{}}
}

// Record 记录一次工具调用
func (m *Meter) Record(userID, assessmentID uuid.UUID, toolName, toolCategory string, tokensUsed int, priceUnit float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Token 费用：按 1000 tokens = 0.01 元计算
	tokenCost := float64(tokensUsed) / 1000 * 0.01
	total := priceUnit + tokenCost

	m.records = append(m.records, UsageRecord{
		ID:           uuid.New(),
		UserID:       userID,
		AssessmentID: assessmentID,
		ToolName:     toolName,
		ToolCategory: toolCategory,
		CallCount:    1,
		TokensUsed:   tokensUsed,
		Amount:       total,
		CreatedAt:    time.Now(),
	})
}

// GetUserUsage 获取用户用量
func (m *Meter) GetUserUsage(userID uuid.UUID) []UsageRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []UsageRecord
	for _, r := range m.records {
		if r.UserID == userID {
			result = append(result, r)
		}
	}
	return result
}

// GetTotalAmount 计算总费用
func (m *Meter) GetTotalAmount(userID uuid.UUID) float64 {
	records := m.GetUserUsage(userID)
	total := 0.0
	for _, r := range records {
		total += r.Amount
	}
	return total
}
