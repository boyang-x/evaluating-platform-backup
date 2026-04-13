package model

import (
	"time"

	"github.com/google/uuid"
)

// Template represents a reusable template owned by an expert.
type Template struct {
	ID              uuid.UUID     `json:"id" db:"id"`
	ExpertID        uuid.UUID     `json:"expert_id" db:"expert_id"`
	UploadBatchID   uuid.UUID     `json:"upload_batch_id" db:"upload_batch_id"`
	UploadBatchName string        `json:"upload_batch_name" db:"upload_batch_name"`
	SubType         string        `json:"sub_type" db:"sub_type"`
	Name            string        `json:"name" db:"name"`
	Description     string        `json:"description" db:"description"`
	Content         string        `json:"content" db:"content"`
	Variables       []TemplateVar `json:"variables" db:"-"`
	Status          string        `json:"status" db:"status"`
	Visibility      string        `json:"visibility" db:"visibility"`
	CreatedAt       time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at" db:"updated_at"`
}

// TemplateVar defines a template variable placeholder.
type TemplateVar struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Default     string `json:"default,omitempty"`
	Required    bool   `json:"required"`
}
