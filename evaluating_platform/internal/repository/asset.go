package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

// AssetRepository 专家资产数据访问层
type AssetRepository struct {
	pool *pgxpool.Pool
}

// NewAssetRepository 创建资产仓库
func NewAssetRepository(pool *pgxpool.Pool) *AssetRepository {
	return &AssetRepository{pool: pool}
}

const assetSelectCols = `
	id, expert_id, name, description, type, visibility, status, version,
	config, price_unit, call_count, created_at, updated_at
`

// Create 创建资产
func (r *AssetRepository) Create(ctx context.Context, asset *model.Asset) error {
	configJSON, _ := json.Marshal(asset.Config)
	query := `
		INSERT INTO assets
			(id, expert_id, name, description, type, visibility, status, version, config, price_unit)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		asset.ID, asset.ExpertID, asset.Name, asset.Description,
		asset.Type, asset.Visibility, asset.Status, asset.Version,
		configJSON, asset.PriceUnit,
	)
	if err != nil {
		return fmt.Errorf("insert asset: %w", err)
	}
	return nil
}

// GetByID 根据 ID 查询资产
func (r *AssetRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Asset, error) {
	query := "SELECT" + assetSelectCols + "FROM assets WHERE id = $1"
	var a model.Asset
	var configJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.ExpertID, &a.Name, &a.Description, &a.Type, &a.Visibility,
		&a.Status, &a.Version, &configJSON, &a.PriceUnit, &a.CallCount,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query asset: %w", err)
	}
	json.Unmarshal(configJSON, &a.Config)
	return &a, nil
}

// ListPublic 列出公开已发布的资产（分页，可按类型过滤）
func (r *AssetRepository) ListPublic(ctx context.Context, assetType string, limit, offset int) ([]model.Asset, int, error) {
	conds := []string{"visibility = 'public'", "status = 'published'"}
	args := []interface{}{}

	if assetType != "" {
		args = append(args, assetType)
		conds = append(conds, fmt.Sprintf("type = $%d", len(args)))
	}
	whereClause := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM assets "+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count public assets: %w", err)
	}

	limitN := len(args) + 1
	offsetN := len(args) + 2
	listArgs := append(args, limit, offset)

	query := fmt.Sprintf(
		"SELECT%sFROM assets %s ORDER BY call_count DESC, created_at DESC LIMIT $%d OFFSET $%d",
		assetSelectCols, whereClause, limitN, offsetN,
	)
	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list public assets: %w", err)
	}
	defer rows.Close()

	assets := make([]model.Asset, 0)
	for rows.Next() {
		var a model.Asset
		var configJSON []byte
		if err := rows.Scan(
			&a.ID, &a.ExpertID, &a.Name, &a.Description, &a.Type, &a.Visibility,
			&a.Status, &a.Version, &configJSON, &a.PriceUnit, &a.CallCount,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan asset: %w", err)
		}
		json.Unmarshal(configJSON, &a.Config)
		assets = append(assets, a)
	}
	return assets, total, nil
}

// ListByExpert 列出指定专家的所有资产（分页，全状态可见）
func (r *AssetRepository) ListByExpert(ctx context.Context, expertID uuid.UUID, limit, offset int) ([]model.Asset, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM assets WHERE expert_id = $1", expertID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count expert assets: %w", err)
	}

	query := "SELECT" + assetSelectCols +
		"FROM assets WHERE expert_id = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3"
	rows, err := r.pool.Query(ctx, query, expertID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list expert assets: %w", err)
	}
	defer rows.Close()

	assets := make([]model.Asset, 0)
	for rows.Next() {
		var a model.Asset
		var configJSON []byte
		if err := rows.Scan(
			&a.ID, &a.ExpertID, &a.Name, &a.Description, &a.Type, &a.Visibility,
			&a.Status, &a.Version, &configJSON, &a.PriceUnit, &a.CallCount,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan asset: %w", err)
		}
		json.Unmarshal(configJSON, &a.Config)
		assets = append(assets, a)
	}
	return assets, total, nil
}

// UpdateStatus 更新资产状态（仅资产所有者可操作）
func (r *AssetRepository) UpdateStatus(ctx context.Context, id, expertID uuid.UUID, status model.AssetStatus) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE assets SET status = $1, updated_at = NOW() WHERE id = $2 AND expert_id = $3",
		status, id, expertID,
	)
	if err != nil {
		return fmt.Errorf("update asset status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("asset not found or access denied")
	}
	return nil
}

// IncrCallCount 增加资产调用计数
func (r *AssetRepository) IncrCallCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE assets SET call_count = call_count + 1, updated_at = NOW() WHERE id = $1", id,
	)
	if err != nil {
		return fmt.Errorf("increment call count: %w", err)
	}
	return nil
}

// UpdateStatusFrom 带来源状态校验的原子状态转换（仅资产所有者可操作）。
// 若当前状态不是 from，或资产不属于 expertID，则返回错误。
func (r *AssetRepository) UpdateStatusFrom(ctx context.Context, id, expertID uuid.UUID, from, to model.AssetStatus) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE assets SET status = $1, updated_at = NOW()
		 WHERE id = $2 AND expert_id = $3 AND status = $4`,
		to, id, expertID, from,
	)
	if err != nil {
		return fmt.Errorf("update asset status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("状态转换无效或无操作权限（当前状态需为 %s）", from)
	}
	return nil
}

// PublishAsset 发布资产：允许从 draft 或 testing 状态转换到 published（仅资产所有者）
func (r *AssetRepository) PublishAsset(ctx context.Context, id, expertID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE assets SET status = $1, updated_at = NOW()
		 WHERE id = $2 AND expert_id = $3 AND status IN ('draft', 'testing')`,
		model.AssetStatusPublished, id, expertID,
	)
	if err != nil {
		return fmt.Errorf("publish asset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("资产不存在、无操作权限，或当前状态不允许发布（需为 draft 或 testing）")
	}
	return nil
}

// ─── Admin 专用方法 ───────────────────────────────────────────────────────────

// ListByStatus 分页查询指定状态的所有资产（无所有权限制，供管理员使用）
func (r *AssetRepository) ListByStatus(ctx context.Context, status model.AssetStatus, limit, offset int) ([]model.Asset, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM assets WHERE status = $1", status,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count assets by status: %w", err)
	}

	query := "SELECT" + assetSelectCols +
		"FROM assets WHERE status = $1 ORDER BY updated_at DESC LIMIT $2 OFFSET $3"
	rows, err := r.pool.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list assets by status: %w", err)
	}
	defer rows.Close()

	assets := make([]model.Asset, 0)
	for rows.Next() {
		var a model.Asset
		var configJSON []byte
		if err := rows.Scan(
			&a.ID, &a.ExpertID, &a.Name, &a.Description, &a.Type, &a.Visibility,
			&a.Status, &a.Version, &configJSON, &a.PriceUnit, &a.CallCount,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan asset: %w", err)
		}
		json.Unmarshal(configJSON, &a.Config)
		assets = append(assets, a)
	}
	return assets, total, nil
}

// AdminApprove 管理员审核通过：testing → published（无所有权限制）
func (r *AssetRepository) AdminApprove(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE assets SET status = $1, visibility = 'public', updated_at = NOW()
		 WHERE id = $2 AND status = 'testing'`,
		model.AssetStatusPublished, id,
	)
	if err != nil {
		return fmt.Errorf("admin approve asset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("资产不存在或当前状态不是 testing")
	}
	return nil
}

// AdminReject 管理员驳回：testing → draft（无所有权限制）
func (r *AssetRepository) AdminReject(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE assets SET status = $1, updated_at = NOW()
		 WHERE id = $2 AND status = 'testing'`,
		model.AssetStatusDraft, id,
	)
	if err != nil {
		return fmt.Errorf("admin reject asset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("资产不存在或当前状态不是 testing")
	}
	return nil
}

// CountPendingAssets 统计待审核（testing 状态）资产数量
func (r *AssetRepository) CountPendingAssets(ctx context.Context) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM assets WHERE status = 'testing'").Scan(&count)
	return count, err
}
