package model

import (
	"time"

	"github.com/google/uuid"
)

type MaclawSkillPublication struct {
	SourceExpertUserID   uuid.UUID
	SourceMaclawTenantID string
	SourceSkillName      string
	SourceVersion        string
	Name                 string
	Description          string
	Status               string
	Enabled              bool
	Triggers             []string
	Tags                 []string
	AssessmentTypes      []string
	Metadata             map[string]string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type MaclawSkillShadow struct {
	EnterpriseUserID         uuid.UUID
	EnterpriseMaclawTenantID string
	SourceExpertUserID       uuid.UUID
	SourceSkillName          string
	SourceVersion            string
	ShadowSkillName          string
	SyncStatus               string
	LastError                string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}
