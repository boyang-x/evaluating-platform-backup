package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

// AttackSampleRepository 攻击样本数据访问层
type AttackSampleRepository struct {
	pool *pgxpool.Pool
}

func NewAttackSampleRepository(pool *pgxpool.Pool) *AttackSampleRepository {
	return &AttackSampleRepository{pool: pool}
}

const sampleSelectCols = ` id, expert_id, sub_type, name, description, storage_path, file_hash,
	sample_count, file_size, status, visibility,
	created_at, updated_at `

func scanSample(row interface {
	Scan(dest ...interface{}) error
}) (*model.AttackSample, error) {
	var s model.AttackSample
	err := row.Scan(
		&s.ID, &s.ExpertID, &s.SubType, &s.Name, &s.Description,
		&s.StoragePath, &s.FileHash,
		&s.SampleCount, &s.FileSize,
		&s.Status, &s.Visibility, &s.CreatedAt, &s.UpdatedAt,
	)
	return &s, err
}

// Create 创建攻击样本记录
func (r *AttackSampleRepository) Create(ctx context.Context, s *model.AttackSample) error {
	query := `
		INSERT INTO attack_samples
			(id, expert_id, sub_type, name, description, storage_path, file_hash,
			 sample_count, file_size, status, visibility)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`
	_, err := r.pool.Exec(ctx, query,
		s.ID, s.ExpertID, s.SubType, s.Name, s.Description,
		s.StoragePath, s.FileHash,
		s.SampleCount, s.FileSize, s.Status, s.Visibility,
	)
	if err != nil {
		return fmt.Errorf("insert attack_sample: %w", err)
	}
	return nil
}

// GetByID 根据 ID 查询
func (r *AttackSampleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AttackSample, error) {
	query := "SELECT" + sampleSelectCols + "FROM attack_samples WHERE id = $1"
	row := r.pool.QueryRow(ctx, query, id)
	s, err := scanSample(row)
	if err != nil {
		return nil, fmt.Errorf("query attack_sample: %w", err)
	}
	return s, nil
}

// ListByExpert 列出专家的样本（可按 sub_type 过滤）
func (r *AttackSampleRepository) ListByExpert(ctx context.Context, expertID uuid.UUID, subType string, limit, offset int) ([]model.AttackSample, int, error) {
	conds := []string{"expert_id = $1"}
	args := []interface{}{expertID}

	if subType != "" {
		args = append(args, subType)
		conds = append(conds, fmt.Sprintf("sub_type = $%d", len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM attack_samples "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count samples: %w", err)
	}

	limitN := len(args) + 1
	offsetN := len(args) + 2
	listArgs := append(args, limit, offset)
	query := fmt.Sprintf("SELECT%sFROM attack_samples %s ORDER BY updated_at DESC LIMIT $%d OFFSET $%d",
		sampleSelectCols, where, limitN, offsetN)

	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list samples: %w", err)
	}
	defer rows.Close()

	var samples []model.AttackSample
	for rows.Next() {
		s, err := scanSample(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan sample: %w", err)
		}
		samples = append(samples, *s)
	}
	return samples, total, nil
}

// Delete 删除样本记录（仅所有者）
func (r *AttackSampleRepository) Delete(ctx context.Context, id, expertID uuid.UUID) (string, error) {
	var storagePath string
	err := r.pool.QueryRow(ctx,
		"DELETE FROM attack_samples WHERE id = $1 AND expert_id = $2 RETURNING storage_path",
		id, expertID,
	).Scan(&storagePath)
	if err != nil {
		return "", fmt.Errorf("delete attack_sample: %w", err)
	}
	return storagePath, nil
}

// ListPublished 列出当前可用于评估编排的样本
func (r *AttackSampleRepository) ListPublished(ctx context.Context, subType string, limit, offset int) ([]model.AttackSample, int, error) {
	where, args := platformDataPublishedWhereClause("sub_type", subType, 0)

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM attack_samples "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count available samples: %w", err)
	}

	limitN := len(args) + 1
	offsetN := len(args) + 2
	listArgs := append(args, limit, offset)
	query := fmt.Sprintf("SELECT%sFROM attack_samples %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		sampleSelectCols, where, limitN, offsetN)

	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list available samples: %w", err)
	}
	defer rows.Close()

	var samples []model.AttackSample
	for rows.Next() {
		s, err := scanSample(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan sample: %w", err)
		}
		samples = append(samples, *s)
	}
	return samples, total, nil
}
