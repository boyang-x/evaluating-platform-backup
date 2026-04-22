package model

import (
	"time"

	"github.com/google/uuid"
)

type AssetType string

const (
	AssetTypeToolConfig AssetType = "tool_config"
	AssetTypeSuite      AssetType = "suite"
)

type AssetVisibility string

const (
	VisibilityPrivate AssetVisibility = "private"
	VisibilityOrg     AssetVisibility = "org"
	VisibilityPublic  AssetVisibility = "public"
)

type AssetStatus string

const (
	AssetStatusDraft      AssetStatus = "draft"
	AssetStatusTesting    AssetStatus = "testing"
	AssetStatusPublished  AssetStatus = "published"
	AssetStatusDeprecated AssetStatus = "deprecated"
)

type Asset struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	ExpertID    uuid.UUID       `json:"expert_id" db:"expert_id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	Type        AssetType       `json:"type" db:"type"`
	Visibility  AssetVisibility `json:"visibility" db:"visibility"`
	Status      AssetStatus     `json:"status" db:"status"`
	Version     string          `json:"version" db:"version"`
	Config      WorkflowConfig  `json:"config" db:"-"`
	PriceUnit   float64         `json:"price_unit" db:"price_unit"`
	CallCount   int64           `json:"call_count" db:"call_count"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// WorkflowConfig remains as a generic JSON config shape used by stored assets.
type WorkflowConfig struct {
	Nodes  []WorkflowNode         `json:"nodes"`
	Edges  []WorkflowEdge         `json:"edges"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type WorkflowNode struct {
	ID       string                 `json:"id"`
	ToolName string                 `json:"tool_name"`
	Label    string                 `json:"label"`
	Params   map[string]interface{} `json:"params,omitempty"`
	Position NodePosition           `json:"position"`
}

type WorkflowEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}
