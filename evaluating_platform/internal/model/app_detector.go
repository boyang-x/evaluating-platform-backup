package model

import (
	"time"

	"github.com/google/uuid"
)

// TargetAppConfig 目标应用连接配置
type TargetAppConfig struct {
	Type    string `json:"type"`              // openai | dify | custom
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Model   string `json:"model,omitempty"`   // openai 类型时需要
	AppID   string `json:"app_id,omitempty"`  // dify 类型时需要
}

// AppDetector 应用检测工具定义
type AppDetector struct {
	ID           uuid.UUID          `json:"id" db:"id"`
	ExpertID     uuid.UUID          `json:"expert_id" db:"expert_id"`
	Name         string             `json:"name" db:"name"`
	SubType      AppDetectorSubType `json:"sub_type" db:"sub_type"`
	Description  string             `json:"description" db:"description"`
	TargetConfig TargetAppConfig    `json:"target_config" db:"target_config"`
	SampleIDs    []uuid.UUID        `json:"sample_ids" db:"sample_ids"`
	TemplateIDs  []uuid.UUID        `json:"template_ids" db:"template_ids"`
	PackageIDs   []uuid.UUID        `json:"package_ids" db:"package_ids"`
	Status       string             `json:"status" db:"status"` // draft | active | archived
	CreatedAt    time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" db:"updated_at"`
}
