package model

import (
	"time"

	"github.com/google/uuid"
)

type MaclawTargetConfig struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	EncryptedConfig []byte
	ConfigKeyID     string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
