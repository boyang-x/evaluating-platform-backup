package model

import (
	"time"

	"github.com/google/uuid"
)

type MaclawResourcePublication struct {
	SourceExpertUserID   uuid.UUID
	SourceMaclawTenantID string
	SourceResourceID     string
	SourceResourceHandle string
	SourceVersion        string
	Name                 string
	Kind                 string
	Status               string
	Enabled              bool
	Summary              string
	AssessmentTypes      []string
	Tags                 []string
	Metadata             map[string]string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type MaclawResourceShadow struct {
	EnterpriseUserID         uuid.UUID
	EnterpriseMaclawTenantID string
	SourceExpertUserID       uuid.UUID
	SourceResourceID         string
	SourceVersion            string
	ShadowResourceID         string
	ShadowResourceHandle     string
	SyncStatus               string
	LastError                string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}
