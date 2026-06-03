package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/maclaw"
)

type MaclawRedteamArtifactRepository struct {
	pool *pgxpool.Pool
}

func NewMaclawRedteamArtifactRepository(pool *pgxpool.Pool) *MaclawRedteamArtifactRepository {
	return &MaclawRedteamArtifactRepository{pool: pool}
}

func (r *MaclawRedteamArtifactRepository) SaveEvidence(ctx context.Context, record maclaw.RedteamEvidenceRecord) (*maclaw.RedteamEvidenceRecord, error) {
	metadata, _ := json.Marshal(record.Metadata)
	const query = `
		INSERT INTO maclaw_redteam_evidence (
			id, handle, platform_user_id, instance_id, session_id, run_id, kind, title, summary, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET
			handle = EXCLUDED.handle,
			instance_id = EXCLUDED.instance_id,
			session_id = EXCLUDED.session_id,
			run_id = EXCLUDED.run_id,
			kind = EXCLUDED.kind,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, handle, platform_user_id, instance_id, session_id, run_id, kind, title, summary, metadata, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, query,
		record.ID,
		record.Handle,
		record.PlatformUserID,
		record.InstanceID,
		record.SessionID,
		record.RunID,
		string(record.Kind),
		record.Title,
		record.Summary,
		metadata,
	)
	return scanRedteamEvidence(row)
}

func (r *MaclawRedteamArtifactRepository) ListEvidence(ctx context.Context, userID uuid.UUID, q maclaw.EvaluationEvidenceQuery) ([]maclaw.RedteamEvidenceRecord, error) {
	limit := q.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	const query = `
		SELECT id, handle, platform_user_id, instance_id, session_id, run_id, kind, title, summary, metadata, created_at, updated_at
		FROM maclaw_redteam_evidence
		WHERE platform_user_id = $1
		  AND ($2 = '' OR instance_id = $2)
		  AND ($3 = '' OR session_id = $3)
		  AND ($4 = '' OR run_id = $4)
		  AND ($5 = '' OR kind = $5)
		ORDER BY created_at DESC
		LIMIT $6
	`
	rows, err := r.pool.Query(ctx, query, userID, q.InstanceID, q.SessionID, q.RunID, string(q.Kind), limit)
	if err != nil {
		return nil, fmt.Errorf("query maclaw redteam evidence: %w", err)
	}
	defer rows.Close()
	items := []maclaw.RedteamEvidenceRecord{}
	for rows.Next() {
		item, err := scanRedteamEvidence(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *MaclawRedteamArtifactRepository) GetEvidence(ctx context.Context, userID uuid.UUID, evidenceID string) (*maclaw.RedteamEvidenceRecord, error) {
	const query = `
		SELECT id, handle, platform_user_id, instance_id, session_id, run_id, kind, title, summary, metadata, created_at, updated_at
		FROM maclaw_redteam_evidence
		WHERE platform_user_id = $1 AND (id = $2 OR handle = $2)
	`
	out, err := scanRedteamEvidence(r.pool.QueryRow(ctx, query, userID, evidenceID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

func (r *MaclawRedteamArtifactRepository) SaveReport(ctx context.Context, record maclaw.RedteamReportRecord) (*maclaw.RedteamReportRecord, error) {
	findings, _ := json.Marshal(record.Findings)
	evidenceHandles, _ := json.Marshal(record.EvidenceHandles)
	metadata, _ := json.Marshal(record.Metadata)
	const query = `
		INSERT INTO maclaw_redteam_reports (
			id, handle, platform_user_id, instance_id, session_id, run_id, title, summary, risk_level,
			safety_score, findings, evidence_handles, metadata
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		ON CONFLICT (id) DO UPDATE SET
			handle = EXCLUDED.handle,
			instance_id = EXCLUDED.instance_id,
			session_id = EXCLUDED.session_id,
			run_id = EXCLUDED.run_id,
			title = EXCLUDED.title,
			summary = EXCLUDED.summary,
			risk_level = EXCLUDED.risk_level,
			safety_score = EXCLUDED.safety_score,
			findings = EXCLUDED.findings,
			evidence_handles = EXCLUDED.evidence_handles,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, handle, platform_user_id, instance_id, session_id, run_id, title, summary, risk_level,
			safety_score, findings, evidence_handles, metadata, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, query,
		record.ID,
		record.Handle,
		record.PlatformUserID,
		record.InstanceID,
		record.SessionID,
		record.RunID,
		record.Title,
		record.Summary,
		record.RiskLevel,
		record.SafetyScore,
		findings,
		evidenceHandles,
		metadata,
	)
	return scanRedteamReport(row)
}

func (r *MaclawRedteamArtifactRepository) GetReport(ctx context.Context, userID uuid.UUID, reportID string) (*maclaw.RedteamReportRecord, error) {
	const query = `
		SELECT id, handle, platform_user_id, instance_id, session_id, run_id, title, summary, risk_level,
			safety_score, findings, evidence_handles, metadata, created_at, updated_at
		FROM maclaw_redteam_reports
		WHERE platform_user_id = $1 AND (id = $2 OR handle = $2)
	`
	out, err := scanRedteamReport(r.pool.QueryRow(ctx, query, userID, reportID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanRedteamEvidence(row rowScanner) (*maclaw.RedteamEvidenceRecord, error) {
	var out maclaw.RedteamEvidenceRecord
	var kind string
	var metadata []byte
	if err := row.Scan(
		&out.ID,
		&out.Handle,
		&out.PlatformUserID,
		&out.InstanceID,
		&out.SessionID,
		&out.RunID,
		&kind,
		&out.Title,
		&out.Summary,
		&metadata,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	out.Kind = maclaw.EvaluationEvidenceKind(kind)
	_ = json.Unmarshal(metadata, &out.Metadata)
	return &out, nil
}

func scanRedteamReport(row rowScanner) (*maclaw.RedteamReportRecord, error) {
	var out maclaw.RedteamReportRecord
	var findings, evidenceHandles, metadata []byte
	if err := row.Scan(
		&out.ID,
		&out.Handle,
		&out.PlatformUserID,
		&out.InstanceID,
		&out.SessionID,
		&out.RunID,
		&out.Title,
		&out.Summary,
		&out.RiskLevel,
		&out.SafetyScore,
		&findings,
		&evidenceHandles,
		&metadata,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, err
	}
	_ = json.Unmarshal(findings, &out.Findings)
	_ = json.Unmarshal(evidenceHandles, &out.EvidenceHandles)
	_ = json.Unmarshal(metadata, &out.Metadata)
	return &out, nil
}
