package model

import (
	"time"

	"github.com/google/uuid"
)

type MaclawResourceRecord struct {
	ID                  uuid.UUID
	OwnerUserID         uuid.UUID
	OwnerMaclawTenantID string
	Handle              string
	Name                string
	Description         string
	Kind                string
	Version             string
	Status              string
	Enabled             bool
	HealthStatus        string
	AssessmentTypes     []string
	Tags                []string
	Summary             string
	EncryptedPayload    []byte
	PayloadKeyID        string
	Metadata            map[string]string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
