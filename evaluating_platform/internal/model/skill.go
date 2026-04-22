package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SkillStatus string

const (
	SkillStatusDraft      SkillStatus = "draft"
	SkillStatusPublished  SkillStatus = "published"
	SkillStatusDisabled   SkillStatus = "disabled"
	SkillStatusDeprecated SkillStatus = "deprecated"
)

type SkillVersionStatus string

const (
	SkillVersionStatusDraft          SkillVersionStatus = "draft"
	SkillVersionStatusSelfTestPassed SkillVersionStatus = "self_test_passed"
	SkillVersionStatusSelfTestFailed SkillVersionStatus = "self_test_failed"
	SkillVersionStatusPublished      SkillVersionStatus = "published"
	SkillVersionStatusDeprecated     SkillVersionStatus = "deprecated"
)

type SkillRunType string

const (
	SkillRunTypeSelfTest SkillRunType = "self_test"
	SkillRunTypeGenerate SkillRunType = "generate"
)

type SkillRunStatus string

const (
	SkillRunStatusPending   SkillRunStatus = "pending"
	SkillRunStatusRunning   SkillRunStatus = "running"
	SkillRunStatusCompleted SkillRunStatus = "completed"
	SkillRunStatusFailed    SkillRunStatus = "failed"
	SkillRunStatusTimeout   SkillRunStatus = "timeout"
)

type Skill struct {
	ID                 uuid.UUID     `json:"id" db:"id"`
	ExpertID           uuid.UUID     `json:"expert_id" db:"expert_id"`
	Name               string        `json:"name" db:"name"`
	Slug               string        `json:"slug" db:"slug"`
	Description        string        `json:"description" db:"description"`
	SkillType          string        `json:"skill_type" db:"skill_type"`
	Category           string        `json:"category" db:"category"`
	CapabilityProfile  string        `json:"capability_profile" db:"capability_profile"`
	Status             SkillStatus   `json:"status" db:"status"`
	LatestVersionID    *uuid.UUID    `json:"latest_version_id,omitempty" db:"latest_version_id"`
	PublishedVersionID *uuid.UUID    `json:"published_version_id,omitempty" db:"published_version_id"`
	CreatedAt          time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time     `json:"updated_at" db:"updated_at"`
	LatestVersion      *SkillVersion `json:"latest_version,omitempty"`
	PublishedVersion   *SkillVersion `json:"published_version,omitempty"`
}

type SkillVersion struct {
	ID                     uuid.UUID          `json:"id" db:"id"`
	SkillID                uuid.UUID          `json:"skill_id" db:"skill_id"`
	Version                string             `json:"version" db:"version"`
	ManifestVersion        string             `json:"manifest_version" db:"manifest_version"`
	DisplayName            string             `json:"display_name" db:"display_name"`
	Summary                string             `json:"summary" db:"summary"`
	PackageObjectPath      string             `json:"package_object_path" db:"package_object_path"`
	PackageHash            string             `json:"package_hash" db:"package_hash"`
	PackageSize            int64              `json:"package_size" db:"package_size"`
	PromptText             string             `json:"prompt_text" db:"prompt_text"`
	InputSourceMode        string             `json:"input_source_mode" db:"input_source_mode"`
	ExecutionRuntime       string             `json:"execution_runtime" db:"execution_runtime"`
	ExecutionEntrypoint    string             `json:"execution_entrypoint" db:"execution_entrypoint"`
	SelfTestEntrypoint     string             `json:"self_test_entrypoint" db:"self_test_entrypoint"`
	Permissions            json.RawMessage    `json:"permissions" db:"permissions"`
	EmbeddedDatasetSummary json.RawMessage    `json:"embedded_dataset_summary" db:"embedded_dataset_summary"`
	AssessmentTypes        json.RawMessage    `json:"assessment_types" db:"assessment_types"`
	Metadata               json.RawMessage    `json:"metadata" db:"metadata"`
	ValidationReport       json.RawMessage    `json:"validation_report" db:"validation_report"`
	Examples               json.RawMessage    `json:"examples" db:"examples"`
	Status                 SkillVersionStatus `json:"status" db:"status"`
	LastSelfTestRunID      *uuid.UUID         `json:"last_self_test_run_id,omitempty" db:"last_self_test_run_id"`
	CreatedAt              time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at" db:"updated_at"`
}

type SkillRun struct {
	ID                    uuid.UUID       `json:"id" db:"id"`
	SkillID               uuid.UUID       `json:"skill_id" db:"skill_id"`
	SkillVersionID        uuid.UUID       `json:"skill_version_id" db:"skill_version_id"`
	AssessmentID          *uuid.UUID      `json:"assessment_id,omitempty" db:"assessment_id"`
	RunType               SkillRunType    `json:"run_type" db:"run_type"`
	TriggerSource         string          `json:"trigger_source" db:"trigger_source"`
	Status                SkillRunStatus  `json:"status" db:"status"`
	ExitCode              *int            `json:"exit_code,omitempty" db:"exit_code"`
	StdoutLog             string          `json:"stdout_log" db:"stdout_log"`
	StderrLog             string          `json:"stderr_log" db:"stderr_log"`
	ResultPayload         json.RawMessage `json:"result_payload" db:"result_payload"`
	ValidationReport      json.RawMessage `json:"validation_report" db:"validation_report"`
	PayloadDatasetSummary json.RawMessage `json:"payload_dataset_summary" db:"payload_dataset_summary"`
	ErrorMessage          string          `json:"error_message" db:"error_message"`
	StartedAt             *time.Time      `json:"started_at,omitempty" db:"started_at"`
	CompletedAt           *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`
}

type SkillConfigValue struct {
	SkillID         uuid.UUID `json:"skill_id" db:"skill_id"`
	FieldKey        string    `json:"field_key" db:"field_key"`
	ValueCiphertext []byte    `json:"value_ciphertext" db:"value_ciphertext"`
	KeyID           string    `json:"key_id" db:"key_id"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}
