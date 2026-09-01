package model

import (
	"time"

	"github.com/google/uuid"
)

type MaclawModelDefault struct {
	ID              string
	EncryptedConfig []byte
	ConfigKeyID     string
	UpdatedBy       *uuid.UUID
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
