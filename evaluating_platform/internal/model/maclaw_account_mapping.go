package model

import (
	"time"

	"github.com/google/uuid"
)

type MaclawProvisioningStatus string

const (
	MaclawProvisioningReady  MaclawProvisioningStatus = "ready"
	MaclawProvisioningFailed MaclawProvisioningStatus = "failed"
)

type MaclawAccountMapping struct {
	PlatformUserID       uuid.UUID
	PlatformRole         UserRole
	PlatformEmail        string
	PlatformOrgName      string
	MaclawTenantID       string
	MaclawUserID         string
	MaclawCredentialID   string
	MaclawInstanceID     string
	EncryptedAPIKey      []byte
	EncryptedAPISecret   []byte
	CredentialKeyID      string
	EncryptedAccessToken []byte
	AccessTokenKeyID     string
	AccessTokenExpiresAt *time.Time
	ProvisioningStatus   MaclawProvisioningStatus
	LastError            string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}
