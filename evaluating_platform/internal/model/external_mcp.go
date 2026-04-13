package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ExternalMCPServer struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	ExpertID        uuid.UUID  `json:"expert_id" db:"expert_id"`
	Name            string     `json:"name" db:"name"`
	Namespace       string     `json:"namespace" db:"namespace"`
	Description     string     `json:"description" db:"description"`
	BaseURL         string     `json:"base_url" db:"base_url"`
	TransportType   string     `json:"transport_type" db:"transport_type"`
	AuthType        string     `json:"auth_type" db:"auth_type"`
	AuthKey         string     `json:"-" db:"auth_key"`
	AuthHeader      string     `json:"auth_header" db:"auth_header"`
	AuthPrefix      string     `json:"auth_prefix" db:"auth_prefix"`
	UpstreamBaseURL string     `json:"upstream_base_url" db:"upstream_base_url"`
	UpstreamAPIKey  string     `json:"-" db:"upstream_api_key"`
	UpstreamModel   string     `json:"upstream_model" db:"upstream_model"`
	UpstreamTimeout int        `json:"upstream_timeout_seconds" db:"upstream_timeout_seconds"`
	TimeoutSeconds  int        `json:"timeout_seconds" db:"timeout_seconds"`
	Enabled         bool       `json:"enabled" db:"enabled"`
	SkillPrompt     string     `json:"skill_prompt" db:"skill_prompt"`
	Status          string     `json:"status" db:"status"`
	LastSyncAt      *time.Time `json:"last_sync_at" db:"last_sync_at"`
	LastError       string     `json:"last_error" db:"last_error"`
	ToolCount       int        `json:"tool_count" db:"-"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
}

type ExternalMCPTool struct {
	ID                 uuid.UUID       `json:"id" db:"id"`
	ServerID           uuid.UUID       `json:"server_id" db:"server_id"`
	RemoteToolName     string          `json:"remote_tool_name" db:"remote_tool_name"`
	ProxyToolName      string          `json:"proxy_tool_name" db:"proxy_tool_name"`
	Description        string          `json:"description" db:"description"`
	InputSchema        json.RawMessage `json:"input_schema" db:"input_schema"`
	CapabilityMetadata json.RawMessage `json:"capability_metadata" db:"capability_metadata"`
	SkillText          string          `json:"skill_text" db:"skill_text"`
	Enabled            bool            `json:"enabled" db:"enabled"`
	CreatedAt          time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at" db:"updated_at"`
}
