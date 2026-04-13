package model

import (
	"time"

	"github.com/google/uuid"
)

// ComposedAttack represents a ready-to-run attack dataset whose payloads are already composed.
type ComposedAttack struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ExpertID    uuid.UUID `json:"expert_id" db:"expert_id"`
	SubType     string    `json:"sub_type" db:"sub_type"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	StoragePath string    `json:"-" db:"storage_path"`
	FileHash    string    `json:"file_hash" db:"file_hash"`
	SampleCount int       `json:"sample_count" db:"sample_count"`
	FileSize    int64     `json:"file_size" db:"file_size"`
	Status      string    `json:"status" db:"status"`
	Visibility  string    `json:"visibility" db:"visibility"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}
