package model

import (
	"time"

	"github.com/google/uuid"
)

// ToolCategory 工具大类（四分类）
type ToolCategory string

const (
	CategorySampleTool   ToolCategory = "sample_tool"   // 样本工具
	CategoryTemplateTool ToolCategory = "template_tool"  // 模版工具
	CategoryComboTool    ToolCategory = "combo_tool"     // 组合工具（样本+模版评测包）
	CategoryAppDetector  ToolCategory = "app_detector"   // 应用检测工具
)

// AttackSampleSubType 攻击样本子类型
type AttackSampleSubType string

const (
	SubTypeDirectInjection      AttackSampleSubType = "direct_injection"
	SubTypeMaliciousInstruction AttackSampleSubType = "malicious_instruction"
	SubTypeComplianceDetection  AttackSampleSubType = "compliance_detection"
	SubTypeMaliciousPoisoning   AttackSampleSubType = "malicious_poisoning"
)

// TemplateSubType 模版子类型
type TemplateSubType string

const (
	SubTypeRolePlay        TemplateSubType = "role_play"
	SubTypeMultilingual    TemplateSubType = "multilingual"
	SubTypeEncodingEvasion TemplateSubType = "encoding_evasion"
)

// AppDetectorSubType 应用检测工具子类型
type AppDetectorSubType string

const (
	SubTypeLLMDetector    AppDetectorSubType = "llm_detector"
	SubTypeDifyDetector   AppDetectorSubType = "dify_detector"
	SubTypeCustomDetector AppDetectorSubType = "custom_detector"
)

// ToolDefinition 工具定义（MCP 工具元数据）
type ToolDefinition struct {
	Name        string       `json:"name"`
	Category    ToolCategory `json:"category"`
	Description string       `json:"description"`
	Parameters  []ToolParam  `json:"parameters"`
	PriceUnit   float64      `json:"price_unit"`
}

// ToolParam 工具参数定义
type ToolParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string | int | bool | object
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ExpertTool 专家上传的自定义工具
type ExpertTool struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	ExpertID    uuid.UUID    `json:"expert_id" db:"expert_id"`
	Name        string       `json:"name" db:"name"`
	Category    ToolCategory `json:"category" db:"category"`
	Description string       `json:"description" db:"description"`
	Script      string       `json:"script" db:"script"`
	Config      string       `json:"config" db:"config"`
	IsPublic    bool         `json:"is_public" db:"is_public"`
	Version     string       `json:"version" db:"version"`
	Status      string       `json:"status" db:"status"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}
