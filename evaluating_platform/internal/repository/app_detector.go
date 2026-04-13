package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

// AppDetectorRepository 应用检测工具仓库
type AppDetectorRepository struct {
	pool *pgxpool.Pool
}

func NewAppDetectorRepository(pool *pgxpool.Pool) *AppDetectorRepository {
	return &AppDetectorRepository{pool: pool}
}

func (r *AppDetectorRepository) Create(ctx context.Context, d *model.AppDetector) error {
	cfgJSON, _ := json.Marshal(d.TargetConfig)
	_, err := r.pool.Exec(ctx, `
		INSERT INTO app_detectors (id, expert_id, name, sub_type, description, target_config, sample_ids, template_ids, package_ids, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		d.ID, d.ExpertID, d.Name, d.SubType, d.Description, cfgJSON,
		d.SampleIDs, d.TemplateIDs, d.PackageIDs, d.Status,
	)
	return err
}

func (r *AppDetectorRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.AppDetector, error) {
	d := &model.AppDetector{}
	var cfgJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, expert_id, name, sub_type, description, target_config, sample_ids, template_ids, package_ids, status, created_at, updated_at
		FROM app_detectors WHERE id=$1`, id,
	).Scan(&d.ID, &d.ExpertID, &d.Name, &d.SubType, &d.Description, &cfgJSON,
		&d.SampleIDs, &d.TemplateIDs, &d.PackageIDs, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal(cfgJSON, &d.TargetConfig)
	return d, nil
}

func (r *AppDetectorRepository) ListByExpert(ctx context.Context, expertID uuid.UUID, limit, offset int) ([]*model.AppDetector, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM app_detectors WHERE expert_id=$1`, expertID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := r.pool.Query(ctx, `
		SELECT id, expert_id, name, sub_type, description, target_config, sample_ids, template_ids, package_ids, status, created_at, updated_at
		FROM app_detectors WHERE expert_id=$1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		expertID, limit, offset,
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*model.AppDetector
	for rows.Next() {
		d := &model.AppDetector{}
		var cfgJSON []byte
		if err := rows.Scan(&d.ID, &d.ExpertID, &d.Name, &d.SubType, &d.Description, &cfgJSON,
			&d.SampleIDs, &d.TemplateIDs, &d.PackageIDs, &d.Status, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(cfgJSON, &d.TargetConfig)
		items = append(items, d)
	}
	return items, total, nil
}

func (r *AppDetectorRepository) Update(ctx context.Context, d *model.AppDetector) error {
	cfgJSON, _ := json.Marshal(d.TargetConfig)
	_, err := r.pool.Exec(ctx, `
		UPDATE app_detectors SET name=$2, sub_type=$3, description=$4, target_config=$5,
		sample_ids=$6, template_ids=$7, package_ids=$8, status=$9, updated_at=NOW()
		WHERE id=$1`,
		d.ID, d.Name, d.SubType, d.Description, cfgJSON,
		d.SampleIDs, d.TemplateIDs, d.PackageIDs, d.Status,
	)
	return err
}

func (r *AppDetectorRepository) Delete(ctx context.Context, id, expertID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM app_detectors WHERE id=$1 AND expert_id=$2`, id, expertID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("检测工具不存在或无权删除")
	}
	return nil
}
