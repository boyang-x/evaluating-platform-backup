package model

import (
	"time"

	"github.com/google/uuid"
)

// AssetType 资产类型
type AssetType string

const (
	AssetTypeToolConfig AssetType = "tool_config" // 单工具参数预设
	AssetTypeWorkflow   AssetType = "workflow"     // 多工具编排 DAG
	AssetTypeSuite      AssetType = "suite"        // 完整评估套件
)

// AssetVisibility 资产可见性
type AssetVisibility string

const (
	VisibilityPrivate AssetVisibility = "private" // 仅作者可见
	VisibilityOrg     AssetVisibility = "org"     // 组织内共享
	VisibilityPublic  AssetVisibility = "public"  // 公开，企业客户可调用
)

// AssetStatus 资产状态
type AssetStatus string

const (
	AssetStatusDraft      AssetStatus = "draft"
	AssetStatusTesting    AssetStatus = "testing"
	AssetStatusPublished  AssetStatus = "published"
	AssetStatusDeprecated AssetStatus = "deprecated"
)

// Asset 专家资产
type Asset struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	ExpertID    uuid.UUID       `json:"expert_id" db:"expert_id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	Type        AssetType       `json:"type" db:"type"`
	Visibility  AssetVisibility `json:"visibility" db:"visibility"`
	Status      AssetStatus     `json:"status" db:"status"`
	Version     string          `json:"version" db:"version"`
	Config      WorkflowConfig  `json:"config" db:"-"`   // 存为 JSONB
	PriceUnit   float64         `json:"price_unit" db:"price_unit"`
	CallCount   int64           `json:"call_count" db:"call_count"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// WorkflowConfig 工作流/工具配置定义（存储为 JSONB）
type WorkflowConfig struct {
	// 工具节点列表
	Nodes []WorkflowNode `json:"nodes"`
	// 有向边（依赖关系）
	Edges []WorkflowEdge `json:"edges"`
	// 全局参数
	Params map[string]interface{} `json:"params,omitempty"`
}

// WorkflowNode 工作流节点（对应一个工具调用）
type WorkflowNode struct {
	ID       string                 `json:"id"`
	ToolName string                 `json:"tool_name"`
	Label    string                 `json:"label"`
	Params   map[string]interface{} `json:"params,omitempty"`
	// 前端展示位置
	Position NodePosition `json:"position"`
}

// WorkflowEdge DAG 有向边
type WorkflowEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"` // 源节点 ID
	Target string `json:"target"` // 目标节点 ID
}

// NodePosition 节点在画布上的位置
type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
