package model

import (
	"time"

	"github.com/google/uuid"
)

// Report 评估报告
type Report struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	AssessmentID    uuid.UUID       `json:"assessment_id" db:"assessment_id"`
	Title           string          `json:"title" db:"title"`
	Summary         string          `json:"summary" db:"summary"`
	RiskLevel       string          `json:"risk_level" db:"risk_level"` // critical | high | medium | low | info
	Findings        []Finding       `json:"findings" db:"-"`
	AttackExamples  []AttackExample `json:"attack_examples,omitempty" db:"-"`
	Metrics         Metrics         `json:"metrics" db:"-"`
	Scope           string          `json:"scope,omitempty" db:"scope"`
	Methodology     string          `json:"methodology,omitempty" db:"methodology"`
	Recommendations []string        `json:"recommendations,omitempty" db:"-"`
	RawContent      string          `json:"raw_content,omitempty" db:"raw_content"`
	PDFURL          string          `json:"pdf_url,omitempty" db:"pdf_url"`
	HTMLURL         string          `json:"html_url,omitempty" db:"html_url"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
}

// Finding 发现项
type Finding struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Severity    string `json:"severity"` // critical | high | medium | low | info
	Category    string `json:"category"`
	Description string `json:"description"`
	Evidence    string `json:"evidence"`
	Suggestion  string `json:"suggestion"`
	ToolName    string `json:"tool_name"`
}

// AttackExample captures one successful attack example for the final report.
type AttackExample struct {
	SampleQuestion string `json:"sample_question"`
	ModelResponse  string `json:"model_response"`
	Severity       string `json:"severity"`
	Reason         string `json:"reason"`
}

// Metrics 评估指标
type Metrics struct {
	TotalTests   int     `json:"total_tests"`
	PassedTests  int     `json:"passed_tests"`
	FailedTests  int     `json:"failed_tests"`
	SuccessCount int     `json:"success_count"` // 攻击成功数
	SuccessRate  float64 `json:"success_rate"`  // 攻击成功率
	RiskScore    float64 `json:"risk_score"`    // 0-100
	TokensUsed   int     `json:"tokens_used"`
	DurationSecs float64 `json:"duration_secs"`
}
