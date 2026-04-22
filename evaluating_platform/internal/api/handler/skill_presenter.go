package handler

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	"evaluating_platform/internal/model"
	skillpkg "evaluating_platform/internal/skill"
)

const maxSkillLogExcerptChars = 2000

type skillVersionSummaryResponse struct {
	ID                uuid.UUID                `json:"id"`
	Version           string                   `json:"version"`
	ManifestVersion   string                   `json:"manifest_version"`
	DisplayName       string                   `json:"display_name"`
	Summary           string                   `json:"summary"`
	InputSourceMode   string                   `json:"input_source_mode"`
	ExecutionRuntime  string                   `json:"execution_runtime"`
	AssessmentTypes   []string                 `json:"assessment_types"`
	Status            model.SkillVersionStatus `json:"status"`
	LastSelfTestRunID *uuid.UUID               `json:"last_self_test_run_id,omitempty"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
}

type skillVersionDetailResponse struct {
	ID                     uuid.UUID                `json:"id"`
	Version                string                   `json:"version"`
	ManifestVersion        string                   `json:"manifest_version"`
	DisplayName            string                   `json:"display_name"`
	Summary                string                   `json:"summary"`
	InputSourceMode        string                   `json:"input_source_mode"`
	ExecutionRuntime       string                   `json:"execution_runtime"`
	AssessmentTypes        []string                 `json:"assessment_types"`
	Permissions            json.RawMessage          `json:"permissions"`
	EmbeddedDatasetSummary json.RawMessage          `json:"embedded_dataset_summary"`
	ValidationReport       json.RawMessage          `json:"validation_report"`
	Status                 model.SkillVersionStatus `json:"status"`
	LastSelfTestRunID      *uuid.UUID               `json:"last_self_test_run_id,omitempty"`
	CreatedAt              time.Time                `json:"created_at"`
	UpdatedAt              time.Time                `json:"updated_at"`
}

type skillRunResponse struct {
	ID                    uuid.UUID            `json:"id"`
	SkillVersionID        uuid.UUID            `json:"skill_version_id"`
	SkillVersion          string               `json:"skill_version,omitempty"`
	RunType               model.SkillRunType   `json:"run_type"`
	TriggerSource         string               `json:"trigger_source"`
	Status                model.SkillRunStatus `json:"status"`
	ExitCode              *int                 `json:"exit_code,omitempty"`
	ErrorMessage          string               `json:"error_message"`
	ValidationReport      json.RawMessage      `json:"validation_report"`
	PayloadDatasetSummary json.RawMessage      `json:"payload_dataset_summary"`
	LogExcerpt            string               `json:"log_excerpt,omitempty"`
	StartedAt             *time.Time           `json:"started_at,omitempty"`
	CompletedAt           *time.Time           `json:"completed_at,omitempty"`
	CreatedAt             time.Time            `json:"created_at"`
	UpdatedAt             time.Time            `json:"updated_at"`
}

type skillSummaryResponse struct {
	ID                 uuid.UUID                    `json:"id"`
	Name               string                       `json:"name"`
	Slug               string                       `json:"slug"`
	Description        string                       `json:"description"`
	SkillType          string                       `json:"skill_type"`
	Category           string                       `json:"category"`
	CapabilityProfile  string                       `json:"capability_profile"`
	InputSourceMode    string                       `json:"input_source_mode"`
	Status             model.SkillStatus            `json:"status"`
	LatestVersionID    *uuid.UUID                   `json:"latest_version_id,omitempty"`
	PublishedVersionID *uuid.UUID                   `json:"published_version_id,omitempty"`
	VersionCount       int                          `json:"version_count"`
	RunCount           int                          `json:"run_count"`
	LatestVersion      *skillVersionSummaryResponse `json:"latest_version,omitempty"`
	PublishedVersion   *skillVersionSummaryResponse `json:"published_version,omitempty"`
	CreatedAt          time.Time                    `json:"created_at"`
	UpdatedAt          time.Time                    `json:"updated_at"`
}

type skillDetailResponse struct {
	ID                     uuid.UUID                    `json:"id"`
	Name                   string                       `json:"name"`
	Slug                   string                       `json:"slug"`
	Description            string                       `json:"description"`
	SkillType              string                       `json:"skill_type"`
	Category               string                       `json:"category"`
	CapabilityProfile      string                       `json:"capability_profile"`
	InputSourceMode        string                       `json:"input_source_mode"`
	AssessmentTypes        []string                     `json:"assessment_types"`
	Permissions            json.RawMessage              `json:"permissions"`
	EmbeddedDatasetSummary json.RawMessage              `json:"embedded_dataset_summary"`
	Status                 model.SkillStatus            `json:"status"`
	LatestVersionID        *uuid.UUID                   `json:"latest_version_id,omitempty"`
	PublishedVersionID     *uuid.UUID                   `json:"published_version_id,omitempty"`
	VersionCount           int                          `json:"version_count"`
	RunCount               int                          `json:"run_count"`
	LatestVersion          *skillVersionSummaryResponse `json:"latest_version,omitempty"`
	PublishedVersion       *skillVersionSummaryResponse `json:"published_version,omitempty"`
	Versions               []skillVersionDetailResponse `json:"versions"`
	Runs                   []skillRunResponse           `json:"runs"`
	CreatedAt              time.Time                    `json:"created_at"`
	UpdatedAt              time.Time                    `json:"updated_at"`
}

type skillConfigVersionResponse struct {
	ID          uuid.UUID                `json:"id"`
	Version     string                   `json:"version"`
	DisplayName string                   `json:"display_name"`
	Status      model.SkillVersionStatus `json:"status"`
}

type skillConfigFieldResponse struct {
	Key         string                  `json:"key"`
	Label       string                  `json:"label"`
	Type        string                  `json:"type"`
	Required    bool                    `json:"required"`
	Secret      bool                    `json:"secret"`
	Description string                  `json:"description"`
	Placeholder string                  `json:"placeholder,omitempty"`
	Default     interface{}             `json:"default,omitempty"`
	EnvName     string                  `json:"env_name"`
	Options     []skillpkg.ConfigOption `json:"options,omitempty"`
	Configured  bool                    `json:"configured"`
	Value       interface{}             `json:"value,omitempty"`
}

type skillConfigResponse struct {
	SkillID             uuid.UUID                   `json:"skill_id"`
	SelectedVersion     *skillConfigVersionResponse `json:"selected_version,omitempty"`
	ConfigComplete      bool                        `json:"config_complete"`
	MissingRequiredKeys []string                    `json:"missing_required_keys"`
	Fields              []skillConfigFieldResponse  `json:"fields"`
}

func presentSkillBundleSummaries(items []skillpkg.SkillBundle) []skillSummaryResponse {
	result := make([]skillSummaryResponse, 0, len(items))
	for i := range items {
		result = append(result, presentSkillBundleSummary(&items[i]))
	}
	return result
}

func presentSkillBundleSummary(bundle *skillpkg.SkillBundle) skillSummaryResponse {
	if bundle == nil || bundle.Skill == nil {
		return skillSummaryResponse{}
	}

	latest := findLatestSkillVersion(bundle)
	published := findPublishedSkillVersion(bundle)
	preferred := latest
	if preferred == nil {
		preferred = published
	}

	return skillSummaryResponse{
		ID:                 bundle.Skill.ID,
		Name:               bundle.Skill.Name,
		Slug:               bundle.Skill.Slug,
		Description:        bundle.Skill.Description,
		SkillType:          bundle.Skill.SkillType,
		Category:           bundle.Skill.Category,
		CapabilityProfile:  bundle.Skill.CapabilityProfile,
		InputSourceMode:    versionInputSourceMode(preferred),
		Status:             bundle.Skill.Status,
		LatestVersionID:    bundle.Skill.LatestVersionID,
		PublishedVersionID: bundle.Skill.PublishedVersionID,
		VersionCount:       skillVersionCount(bundle),
		RunCount:           len(bundle.Runs),
		LatestVersion:      presentSkillVersionSummary(latest),
		PublishedVersion:   presentSkillVersionSummary(published),
		CreatedAt:          bundle.Skill.CreatedAt,
		UpdatedAt:          bundle.Skill.UpdatedAt,
	}
}

func presentSkillBundleDetail(bundle *skillpkg.SkillBundle) skillDetailResponse {
	if bundle == nil || bundle.Skill == nil {
		return skillDetailResponse{}
	}

	latest := findLatestSkillVersion(bundle)
	published := findPublishedSkillVersion(bundle)
	preferred := latest
	if preferred == nil {
		preferred = published
	}

	versionMap := make(map[uuid.UUID]string, len(bundle.Versions))
	versions := make([]skillVersionDetailResponse, 0, len(bundle.Versions))
	for _, version := range bundle.Versions {
		versionMap[version.ID] = version.Version
		versions = append(versions, presentSkillVersionDetail(&version))
	}

	return skillDetailResponse{
		ID:                     bundle.Skill.ID,
		Name:                   bundle.Skill.Name,
		Slug:                   bundle.Skill.Slug,
		Description:            bundle.Skill.Description,
		SkillType:              bundle.Skill.SkillType,
		Category:               bundle.Skill.Category,
		CapabilityProfile:      bundle.Skill.CapabilityProfile,
		InputSourceMode:        versionInputSourceMode(preferred),
		AssessmentTypes:        versionAssessmentTypes(preferred),
		Permissions:            versionPermissions(preferred),
		EmbeddedDatasetSummary: versionEmbeddedDatasetSummary(preferred),
		Status:                 bundle.Skill.Status,
		LatestVersionID:        bundle.Skill.LatestVersionID,
		PublishedVersionID:     bundle.Skill.PublishedVersionID,
		VersionCount:           skillVersionCount(bundle),
		RunCount:               len(bundle.Runs),
		LatestVersion:          presentSkillVersionSummary(latest),
		PublishedVersion:       presentSkillVersionSummary(published),
		Versions:               versions,
		Runs:                   presentSkillRuns(bundle.Runs, versionMap),
		CreatedAt:              bundle.Skill.CreatedAt,
		UpdatedAt:              bundle.Skill.UpdatedAt,
	}
}

func presentSkillVersionSummary(version *model.SkillVersion) *skillVersionSummaryResponse {
	if version == nil {
		return nil
	}
	return &skillVersionSummaryResponse{
		ID:                version.ID,
		Version:           version.Version,
		ManifestVersion:   version.ManifestVersion,
		DisplayName:       version.DisplayName,
		Summary:           version.Summary,
		InputSourceMode:   version.InputSourceMode,
		ExecutionRuntime:  version.ExecutionRuntime,
		AssessmentTypes:   parseStringArray(version.AssessmentTypes),
		Status:            version.Status,
		LastSelfTestRunID: version.LastSelfTestRunID,
		CreatedAt:         version.CreatedAt,
		UpdatedAt:         version.UpdatedAt,
	}
}

func presentSkillVersionDetail(version *model.SkillVersion) skillVersionDetailResponse {
	if version == nil {
		return skillVersionDetailResponse{}
	}
	return skillVersionDetailResponse{
		ID:                     version.ID,
		Version:                version.Version,
		ManifestVersion:        version.ManifestVersion,
		DisplayName:            version.DisplayName,
		Summary:                version.Summary,
		InputSourceMode:        version.InputSourceMode,
		ExecutionRuntime:       version.ExecutionRuntime,
		AssessmentTypes:        parseStringArray(version.AssessmentTypes),
		Permissions:            rawJSONOrDefault(version.Permissions, `{}`),
		EmbeddedDatasetSummary: rawJSONOrDefault(version.EmbeddedDatasetSummary, `{}`),
		ValidationReport:       rawJSONOrDefault(version.ValidationReport, `{}`),
		Status:                 version.Status,
		LastSelfTestRunID:      version.LastSelfTestRunID,
		CreatedAt:              version.CreatedAt,
		UpdatedAt:              version.UpdatedAt,
	}
}

func presentSkillRuns(items []model.SkillRun, versionMap map[uuid.UUID]string) []skillRunResponse {
	result := make([]skillRunResponse, 0, len(items))
	for i := range items {
		result = append(result, presentSkillRun(&items[i], versionMap))
	}
	return result
}

func presentSkillRun(run *model.SkillRun, versionMap map[uuid.UUID]string) skillRunResponse {
	if run == nil {
		return skillRunResponse{}
	}

	skillVersion := ""
	if versionMap != nil {
		skillVersion = versionMap[run.SkillVersionID]
	}

	return skillRunResponse{
		ID:                    run.ID,
		SkillVersionID:        run.SkillVersionID,
		SkillVersion:          skillVersion,
		RunType:               run.RunType,
		TriggerSource:         run.TriggerSource,
		Status:                run.Status,
		ExitCode:              run.ExitCode,
		ErrorMessage:          run.ErrorMessage,
		ValidationReport:      rawJSONOrDefault(run.ValidationReport, `{}`),
		PayloadDatasetSummary: rawJSONOrDefault(run.PayloadDatasetSummary, `{}`),
		LogExcerpt:            buildSkillLogExcerpt(run),
		StartedAt:             run.StartedAt,
		CompletedAt:           run.CompletedAt,
		CreatedAt:             run.CreatedAt,
		UpdatedAt:             run.UpdatedAt,
	}
}

func findLatestSkillVersion(bundle *skillpkg.SkillBundle) *model.SkillVersion {
	if bundle == nil {
		return nil
	}
	if bundle.Version != nil {
		return bundle.Version
	}
	if bundle.Skill != nil && bundle.Skill.LatestVersionID != nil {
		return findSkillVersionByID(bundle.Versions, *bundle.Skill.LatestVersionID)
	}
	return nil
}

func findPublishedSkillVersion(bundle *skillpkg.SkillBundle) *model.SkillVersion {
	if bundle == nil || bundle.Skill == nil || bundle.Skill.PublishedVersionID == nil {
		return nil
	}
	return findSkillVersionByID(bundle.Versions, *bundle.Skill.PublishedVersionID)
}

func findSkillVersionByID(items []model.SkillVersion, id uuid.UUID) *model.SkillVersion {
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

func presentSkillConfigView(view *skillpkg.SkillConfigView) skillConfigResponse {
	if view == nil {
		return skillConfigResponse{}
	}

	fields := make([]skillConfigFieldResponse, 0, len(view.Fields))
	for _, field := range view.Fields {
		fields = append(fields, skillConfigFieldResponse{
			Key:         field.Key,
			Label:       field.Label,
			Type:        field.Type,
			Required:    field.Required,
			Secret:      field.Secret,
			Description: field.Description,
			Placeholder: field.Placeholder,
			Default:     field.Default,
			EnvName:     field.EnvName,
			Options:     append([]skillpkg.ConfigOption(nil), field.Options...),
			Configured:  field.Configured,
			Value:       field.Value,
		})
	}

	return skillConfigResponse{
		SkillID:             view.SkillID,
		SelectedVersion:     presentSkillConfigVersion(view.SelectedVersion),
		ConfigComplete:      view.ConfigComplete,
		MissingRequiredKeys: append([]string(nil), view.MissingRequiredKeys...),
		Fields:              fields,
	}
}

func presentSkillConfigVersion(version *model.SkillVersion) *skillConfigVersionResponse {
	if version == nil {
		return nil
	}
	return &skillConfigVersionResponse{
		ID:          version.ID,
		Version:     version.Version,
		DisplayName: version.DisplayName,
		Status:      version.Status,
	}
}

func skillVersionCount(bundle *skillpkg.SkillBundle) int {
	if bundle == nil {
		return 0
	}
	if len(bundle.Versions) > 0 {
		return len(bundle.Versions)
	}
	if bundle.Version != nil {
		return 1
	}
	return 0
}

func versionInputSourceMode(version *model.SkillVersion) string {
	if version == nil {
		return ""
	}
	return version.InputSourceMode
}

func versionAssessmentTypes(version *model.SkillVersion) []string {
	if version == nil {
		return []string{}
	}
	return parseStringArray(version.AssessmentTypes)
}

func versionPermissions(version *model.SkillVersion) json.RawMessage {
	if version == nil {
		return json.RawMessage(`{}`)
	}
	return rawJSONOrDefault(version.Permissions, `{}`)
}

func versionEmbeddedDatasetSummary(version *model.SkillVersion) json.RawMessage {
	if version == nil {
		return json.RawMessage(`{}`)
	}
	return rawJSONOrDefault(version.EmbeddedDatasetSummary, `{}`)
}

func parseStringArray(data json.RawMessage) []string {
	if len(strings.TrimSpace(string(data))) == 0 {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal(data, &result); err != nil {
		return []string{}
	}
	return result
}

func rawJSONOrDefault(data json.RawMessage, fallback string) json.RawMessage {
	if len(strings.TrimSpace(string(data))) == 0 {
		return json.RawMessage(fallback)
	}
	return data
}

func buildSkillLogExcerpt(run *model.SkillRun) string {
	if run == nil || run.RunType != model.SkillRunTypeSelfTest {
		return ""
	}

	parts := make([]string, 0, 2)
	if stdout := strings.TrimSpace(run.StdoutLog); stdout != "" {
		parts = append(parts, "stdout:\n"+stdout)
	}
	if stderr := strings.TrimSpace(run.StderrLog); stderr != "" {
		parts = append(parts, "stderr:\n"+stderr)
	}
	if len(parts) == 0 {
		return ""
	}

	combined := strings.Join(parts, "\n\n")
	runes := []rune(combined)
	if len(runes) <= maxSkillLogExcerptChars {
		return combined
	}
	return string(runes[:maxSkillLogExcerptChars]) + "\n...[truncated]"
}
