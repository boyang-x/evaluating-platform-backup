package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

type SkillRepository struct {
	pool *pgxpool.Pool
}

type SkillVersionRepository struct {
	pool *pgxpool.Pool
}

type SkillRunRepository struct {
	pool *pgxpool.Pool
}

func NewSkillRepository(pool *pgxpool.Pool) *SkillRepository {
	return &SkillRepository{pool: pool}
}

func NewSkillVersionRepository(pool *pgxpool.Pool) *SkillVersionRepository {
	return &SkillVersionRepository{pool: pool}
}

func NewSkillRunRepository(pool *pgxpool.Pool) *SkillRunRepository {
	return &SkillRunRepository{pool: pool}
}

const skillSelectCols = `
	id, expert_id, name, slug, description, skill_type, category,
	capability_profile, status, latest_version_id, published_version_id,
	created_at, updated_at
`

const skillVersionSelectCols = `
	id, skill_id, version, manifest_version, display_name, summary,
	package_object_path, package_hash, package_size, prompt_text,
	input_source_mode, execution_runtime, execution_entrypoint,
	self_test_entrypoint, permissions, embedded_dataset_summary,
	assessment_types, metadata, validation_report, examples, status,
	last_self_test_run_id, created_at, updated_at
`

const skillRunSelectCols = `
	id, skill_id, skill_version_id, assessment_id, run_type, trigger_source,
	status, exit_code, stdout_log, stderr_log, result_payload,
	validation_report, payload_dataset_summary, error_message,
	started_at, completed_at, created_at, updated_at
`

func scanSkill(row interface {
	Scan(dest ...interface{}) error
}) (*model.Skill, error) {
	var item model.Skill
	if err := row.Scan(
		&item.ID,
		&item.ExpertID,
		&item.Name,
		&item.Slug,
		&item.Description,
		&item.SkillType,
		&item.Category,
		&item.CapabilityProfile,
		&item.Status,
		&item.LatestVersionID,
		&item.PublishedVersionID,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func scanSkillVersion(row interface {
	Scan(dest ...interface{}) error
}) (*model.SkillVersion, error) {
	var item model.SkillVersion
	if err := row.Scan(
		&item.ID,
		&item.SkillID,
		&item.Version,
		&item.ManifestVersion,
		&item.DisplayName,
		&item.Summary,
		&item.PackageObjectPath,
		&item.PackageHash,
		&item.PackageSize,
		&item.PromptText,
		&item.InputSourceMode,
		&item.ExecutionRuntime,
		&item.ExecutionEntrypoint,
		&item.SelfTestEntrypoint,
		&item.Permissions,
		&item.EmbeddedDatasetSummary,
		&item.AssessmentTypes,
		&item.Metadata,
		&item.ValidationReport,
		&item.Examples,
		&item.Status,
		&item.LastSelfTestRunID,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func scanSkillRun(row interface {
	Scan(dest ...interface{}) error
}) (*model.SkillRun, error) {
	var item model.SkillRun
	if err := row.Scan(
		&item.ID,
		&item.SkillID,
		&item.SkillVersionID,
		&item.AssessmentID,
		&item.RunType,
		&item.TriggerSource,
		&item.Status,
		&item.ExitCode,
		&item.StdoutLog,
		&item.StderrLog,
		&item.ResultPayload,
		&item.ValidationReport,
		&item.PayloadDatasetSummary,
		&item.ErrorMessage,
		&item.StartedAt,
		&item.CompletedAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *SkillRepository) Create(ctx context.Context, item *model.Skill) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO skills (
			id, expert_id, name, slug, description, skill_type, category,
			capability_profile, status, latest_version_id, published_version_id
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING created_at, updated_at
	`,
		item.ID,
		item.ExpertID,
		item.Name,
		item.Slug,
		item.Description,
		item.SkillType,
		item.Category,
		item.CapabilityProfile,
		item.Status,
		item.LatestVersionID,
		item.PublishedVersionID,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
}

func (r *SkillRepository) UpdateMetadata(ctx context.Context, item *model.Skill) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE skills
		SET name = $1,
			description = $2,
			skill_type = $3,
			category = $4,
			capability_profile = $5,
			updated_at = NOW()
		WHERE id = $6
	`,
		item.Name,
		item.Description,
		item.SkillType,
		item.Category,
		item.CapabilityProfile,
		item.ID,
	)
	if err != nil {
		return fmt.Errorf("update skill metadata: %w", err)
	}
	return nil
}

func (r *SkillRepository) SetLatestVersion(ctx context.Context, skillID, versionID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE skills SET latest_version_id = $1, updated_at = NOW() WHERE id = $2`, versionID, skillID)
	if err != nil {
		return fmt.Errorf("update skill latest version: %w", err)
	}
	return nil
}

func (r *SkillRepository) PublishVersion(ctx context.Context, skillID, versionID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE skills
		SET published_version_id = $1, status = 'published', updated_at = NOW()
		WHERE id = $2
	`, versionID, skillID)
	if err != nil {
		return fmt.Errorf("publish skill: %w", err)
	}
	return nil
}

func (r *SkillRepository) SetStatus(ctx context.Context, skillID, expertID uuid.UUID, status model.SkillStatus) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE skills
		SET status = $1, updated_at = NOW()
		WHERE id = $2 AND expert_id = $3
	`, status, skillID, expertID)
	if err != nil {
		return fmt.Errorf("update skill status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("skill not found or access denied")
	}
	return nil
}

func (r *SkillRepository) Delete(ctx context.Context, skillID, expertID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM skills WHERE id = $1 AND expert_id = $2`, skillID, expertID)
	if err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("skill not found or access denied")
	}
	return nil
}

func (r *SkillRepository) GetByIDForExpert(ctx context.Context, id, expertID uuid.UUID) (*model.Skill, error) {
	item, err := scanSkill(r.pool.QueryRow(ctx, `SELECT `+skillSelectCols+` FROM skills WHERE id = $1 AND expert_id = $2`, id, expertID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query skill: %w", err)
	}
	return item, nil
}

func (r *SkillRepository) GetBySlugForExpert(ctx context.Context, slug string, expertID uuid.UUID) (*model.Skill, error) {
	item, err := scanSkill(r.pool.QueryRow(ctx, `SELECT `+skillSelectCols+` FROM skills WHERE slug = $1 AND expert_id = $2`, slug, expertID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query skill by slug: %w", err)
	}
	return item, nil
}

func (r *SkillRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Skill, error) {
	item, err := scanSkill(r.pool.QueryRow(ctx, `SELECT `+skillSelectCols+` FROM skills WHERE id = $1`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query skill: %w", err)
	}
	return item, nil
}

func (r *SkillRepository) ListByExpert(ctx context.Context, expertID uuid.UUID) ([]model.Skill, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+skillSelectCols+`
		FROM skills
		WHERE expert_id = $1
		ORDER BY updated_at DESC
	`, expertID)
	if err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}
	defer rows.Close()

	items := make([]model.Skill, 0)
	for rows.Next() {
		item, err := scanSkill(rows)
		if err != nil {
			return nil, fmt.Errorf("scan skill: %w", err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func (r *SkillRepository) ListPublished(ctx context.Context) ([]model.Skill, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+skillSelectCols+`
		FROM skills
		WHERE status = 'published' AND published_version_id IS NOT NULL
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list published skills: %w", err)
	}
	defer rows.Close()

	items := make([]model.Skill, 0)
	for rows.Next() {
		item, err := scanSkill(rows)
		if err != nil {
			return nil, fmt.Errorf("scan published skill: %w", err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func (r *SkillVersionRepository) Create(ctx context.Context, item *model.SkillVersion) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO skill_versions (
			id, skill_id, version, manifest_version, display_name, summary,
			package_object_path, package_hash, package_size, prompt_text,
			input_source_mode, execution_runtime, execution_entrypoint,
			self_test_entrypoint, permissions, embedded_dataset_summary,
			assessment_types, metadata, validation_report, examples,
			status, last_self_test_run_id
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,
			$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,
			$21,$22
		)
		RETURNING created_at, updated_at
	`,
		item.ID,
		item.SkillID,
		item.Version,
		item.ManifestVersion,
		item.DisplayName,
		item.Summary,
		item.PackageObjectPath,
		item.PackageHash,
		item.PackageSize,
		item.PromptText,
		item.InputSourceMode,
		item.ExecutionRuntime,
		item.ExecutionEntrypoint,
		item.SelfTestEntrypoint,
		jsonOrDefault(item.Permissions, `{}`),
		jsonOrDefault(item.EmbeddedDatasetSummary, `{}`),
		jsonOrDefault(item.AssessmentTypes, `[]`),
		jsonOrDefault(item.Metadata, `{}`),
		jsonOrDefault(item.ValidationReport, `{}`),
		jsonOrDefault(item.Examples, `{}`),
		item.Status,
		item.LastSelfTestRunID,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
}

func (r *SkillVersionRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.SkillVersion, error) {
	item, err := scanSkillVersion(r.pool.QueryRow(ctx, `SELECT `+skillVersionSelectCols+` FROM skill_versions WHERE id = $1`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query skill version: %w", err)
	}
	return item, nil
}

func (r *SkillVersionRepository) GetPublishedBySkillID(ctx context.Context, skillID uuid.UUID) (*model.SkillVersion, error) {
	item, err := scanSkillVersion(r.pool.QueryRow(ctx, `
		SELECT `+skillVersionSelectCols+`
		FROM skill_versions
		WHERE skill_id = $1 AND status = 'published'
		ORDER BY created_at DESC
		LIMIT 1
	`, skillID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query published skill version: %w", err)
	}
	return item, nil
}

func (r *SkillVersionRepository) GetLatestBySkillID(ctx context.Context, skillID uuid.UUID) (*model.SkillVersion, error) {
	item, err := scanSkillVersion(r.pool.QueryRow(ctx, `
		SELECT `+skillVersionSelectCols+`
		FROM skill_versions
		WHERE skill_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, skillID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest skill version: %w", err)
	}
	return item, nil
}

func (r *SkillVersionRepository) ListBySkill(ctx context.Context, skillID uuid.UUID) ([]model.SkillVersion, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+skillVersionSelectCols+`
		FROM skill_versions
		WHERE skill_id = $1
		ORDER BY created_at DESC
	`, skillID)
	if err != nil {
		return nil, fmt.Errorf("list skill versions: %w", err)
	}
	defer rows.Close()

	items := make([]model.SkillVersion, 0)
	for rows.Next() {
		item, err := scanSkillVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("scan skill version: %w", err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func (r *SkillVersionRepository) UpdateRunResult(ctx context.Context, id uuid.UUID, status model.SkillVersionStatus, runID *uuid.UUID, validationReport json.RawMessage) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE skill_versions
		SET status = $1,
			last_self_test_run_id = $2,
			validation_report = $3,
			updated_at = NOW()
		WHERE id = $4
	`, status, runID, jsonOrDefault(validationReport, `{}`), id)
	if err != nil {
		return fmt.Errorf("update skill version run result: %w", err)
	}
	return nil
}

func (r *SkillVersionRepository) MarkPublished(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `UPDATE skill_versions SET status = 'published', updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("mark skill version published: %w", err)
	}
	return nil
}

func (r *SkillVersionRepository) ClearPublishedBySkill(ctx context.Context, skillID uuid.UUID, keepVersionID *uuid.UUID) error {
	if keepVersionID != nil {
		_, err := r.pool.Exec(ctx, `
			UPDATE skill_versions
			SET status = 'self_test_passed', updated_at = NOW()
			WHERE skill_id = $1 AND status = 'published' AND id <> $2
		`, skillID, *keepVersionID)
		if err != nil {
			return fmt.Errorf("clear published skill versions: %w", err)
		}
		return nil
	}

	_, err := r.pool.Exec(ctx, `
		UPDATE skill_versions
		SET status = 'self_test_passed', updated_at = NOW()
		WHERE skill_id = $1 AND status = 'published'
	`, skillID)
	if err != nil {
		return fmt.Errorf("clear published skill versions: %w", err)
	}
	return nil
}

func (r *SkillVersionRepository) MarkDeprecatedBySkill(ctx context.Context, skillID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE skill_versions
		SET status = CASE WHEN status = 'published' THEN 'deprecated' ELSE status END,
			updated_at = NOW()
		WHERE skill_id = $1
	`, skillID)
	if err != nil {
		return fmt.Errorf("deprecate skill versions: %w", err)
	}
	return nil
}

func (r *SkillRunRepository) Create(ctx context.Context, item *model.SkillRun) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO skill_runs (
			id, skill_id, skill_version_id, assessment_id, run_type, trigger_source,
			status, exit_code, stdout_log, stderr_log, result_payload,
			validation_report, payload_dataset_summary, error_message,
			started_at, completed_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16
		)
		RETURNING created_at, updated_at
	`,
		item.ID,
		item.SkillID,
		item.SkillVersionID,
		item.AssessmentID,
		item.RunType,
		item.TriggerSource,
		item.Status,
		item.ExitCode,
		item.StdoutLog,
		item.StderrLog,
		jsonOrDefault(item.ResultPayload, `{}`),
		jsonOrDefault(item.ValidationReport, `{}`),
		jsonOrDefault(item.PayloadDatasetSummary, `{}`),
		item.ErrorMessage,
		item.StartedAt,
		item.CompletedAt,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
}

func (r *SkillRunRepository) UpdateResult(ctx context.Context, item *model.SkillRun) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE skill_runs
		SET status = $1,
			exit_code = $2,
			stdout_log = $3,
			stderr_log = $4,
			result_payload = $5,
			validation_report = $6,
			payload_dataset_summary = $7,
			error_message = $8,
			started_at = $9,
			completed_at = $10,
			updated_at = NOW()
		WHERE id = $11
	`,
		item.Status,
		item.ExitCode,
		item.StdoutLog,
		item.StderrLog,
		jsonOrDefault(item.ResultPayload, `{}`),
		jsonOrDefault(item.ValidationReport, `{}`),
		jsonOrDefault(item.PayloadDatasetSummary, `{}`),
		item.ErrorMessage,
		item.StartedAt,
		item.CompletedAt,
		item.ID,
	)
	if err != nil {
		return fmt.Errorf("update skill run result: %w", err)
	}
	return nil
}

func (r *SkillRunRepository) MarkTimedOutSelfTests(ctx context.Context, skillID uuid.UUID, startedBefore time.Time, message string) error {
	if strings.TrimSpace(message) == "" {
		message = "skill self-test timed out while waiting for completion"
	}
	_, err := r.pool.Exec(ctx, `
		UPDATE skill_runs
		SET status = 'timeout',
			error_message = $1,
			completed_at = COALESCE(completed_at, NOW()),
			updated_at = NOW()
		WHERE skill_id = $2
		  AND run_type = 'self_test'
		  AND status = 'running'
		  AND started_at IS NOT NULL
		  AND started_at < $3
	`, message, skillID, startedBefore)
	if err != nil {
		return fmt.Errorf("mark timed out self tests: %w", err)
	}
	return nil
}

func (r *SkillRunRepository) ListBySkill(ctx context.Context, skillID uuid.UUID) ([]model.SkillRun, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+skillRunSelectCols+`
		FROM skill_runs
		WHERE skill_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, skillID)
	if err != nil {
		return nil, fmt.Errorf("list skill runs: %w", err)
	}
	defer rows.Close()

	items := make([]model.SkillRun, 0)
	for rows.Next() {
		item, err := scanSkillRun(rows)
		if err != nil {
			return nil, fmt.Errorf("scan skill run: %w", err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func (r *SkillRunRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.SkillRun, error) {
	item, err := scanSkillRun(r.pool.QueryRow(ctx, `SELECT `+skillRunSelectCols+` FROM skill_runs WHERE id = $1`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query skill run: %w", err)
	}
	return item, nil
}

func (r *SkillRepository) SearchPublishedSkills(ctx context.Context, assessmentTypes []string, goal string) ([]model.Skill, error) {
	skills, err := r.ListPublished(ctx)
	if err != nil {
		return nil, err
	}
	if len(skills) == 0 {
		return nil, nil
	}

	if len(assessmentTypes) == 0 && strings.TrimSpace(goal) == "" {
		return skills, nil
	}

	normalizedGoal := strings.ToLower(strings.TrimSpace(goal))
	filtered := make([]model.Skill, 0, len(skills))
	for _, skill := range skills {
		score := 0
		nameDesc := strings.ToLower(skill.Name + " " + skill.Description + " " + skill.CapabilityProfile + " " + skill.Category)
		for _, item := range assessmentTypes {
			if item != "" && strings.Contains(nameDesc, strings.ToLower(item)) {
				score++
			}
		}
		if normalizedGoal != "" {
			for _, token := range strings.Fields(normalizedGoal) {
				if len(token) < 2 {
					continue
				}
				if strings.Contains(nameDesc, token) {
					score++
				}
			}
		}
		if score > 0 || len(assessmentTypes) == 0 {
			filtered = append(filtered, skill)
		}
	}
	if len(filtered) == 0 {
		return skills, nil
	}
	return filtered, nil
}

func jsonOrDefault(data json.RawMessage, fallback string) json.RawMessage {
	if len(strings.TrimSpace(string(data))) == 0 {
		return json.RawMessage(fallback)
	}
	return data
}
