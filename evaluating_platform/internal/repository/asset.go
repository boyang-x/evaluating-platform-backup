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

type AssetRepository struct {
	pool *pgxpool.Pool
}

func NewAssetRepository(pool *pgxpool.Pool) *AssetRepository {
	return &AssetRepository{pool: pool}
}

const assetSelectCols = `
	id, expert_id, name, description, type, visibility, status, version,
	config, price_unit, call_count, created_at, updated_at
`

func (r *AssetRepository) Create(ctx context.Context, asset *model.Asset) error {
	configJSON, _ := json.Marshal(asset.Config)
	query := `
		INSERT INTO assets
			(id, expert_id, name, description, type, visibility, status, version, config, price_unit)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(
		ctx,
		query,
		asset.ID,
		asset.ExpertID,
		asset.Name,
		asset.Description,
		asset.Type,
		asset.Visibility,
		asset.Status,
		asset.Version,
		configJSON,
		asset.PriceUnit,
	)
	if err != nil {
		return fmt.Errorf("insert asset: %w", err)
	}
	return nil
}

func (r *AssetRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Asset, error) {
	query := "SELECT" + assetSelectCols + "FROM assets WHERE id = $1"

	var asset model.Asset
	var configJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&asset.ID,
		&asset.ExpertID,
		&asset.Name,
		&asset.Description,
		&asset.Type,
		&asset.Visibility,
		&asset.Status,
		&asset.Version,
		&configJSON,
		&asset.PriceUnit,
		&asset.CallCount,
		&asset.CreatedAt,
		&asset.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query asset: %w", err)
	}

	_ = json.Unmarshal(configJSON, &asset.Config)
	return &asset, nil
}

// ListPublic still filters legacy workflow rows so removed workflow assets stay hidden.
func (r *AssetRepository) ListPublic(ctx context.Context, assetType string, limit, offset int) ([]model.Asset, int, error) {
	conds := []string{"visibility = 'public'", "status = 'published'", "type <> 'workflow'"}
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
		var asset model.Asset
		var configJSON []byte
		if err := rows.Scan(
			&asset.ID,
			&asset.ExpertID,
			&asset.Name,
			&asset.Description,
			&asset.Type,
			&asset.Visibility,
			&asset.Status,
			&asset.Version,
			&configJSON,
			&asset.PriceUnit,
			&asset.CallCount,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan asset: %w", err)
		}
		_ = json.Unmarshal(configJSON, &asset.Config)
		assets = append(assets, asset)
	}

	return assets, total, nil
}

// ListByExpert still filters legacy workflow rows so removed workflow assets stay hidden.
func (r *AssetRepository) ListByExpert(ctx context.Context, expertID uuid.UUID, limit, offset int) ([]model.Asset, int, error) {
	var total int
	if err := r.pool.QueryRow(
		ctx,
		"SELECT COUNT(*) FROM assets WHERE expert_id = $1 AND type <> 'workflow'",
		expertID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count expert assets: %w", err)
	}

	query := "SELECT" + assetSelectCols +
		"FROM assets WHERE expert_id = $1 AND type <> 'workflow' ORDER BY updated_at DESC LIMIT $2 OFFSET $3"
	rows, err := r.pool.Query(ctx, query, expertID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list expert assets: %w", err)
	}
	defer rows.Close()

	assets := make([]model.Asset, 0)
	for rows.Next() {
		var asset model.Asset
		var configJSON []byte
		if err := rows.Scan(
			&asset.ID,
			&asset.ExpertID,
			&asset.Name,
			&asset.Description,
			&asset.Type,
			&asset.Visibility,
			&asset.Status,
			&asset.Version,
			&configJSON,
			&asset.PriceUnit,
			&asset.CallCount,
			&asset.CreatedAt,
			&asset.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan asset: %w", err)
		}
		_ = json.Unmarshal(configJSON, &asset.Config)
		assets = append(assets, asset)
	}

	return assets, total, nil
}

func (r *AssetRepository) UpdateStatus(ctx context.Context, id, expertID uuid.UUID, status model.AssetStatus) error {
	tag, err := r.pool.Exec(
		ctx,
		"UPDATE assets SET status = $1, updated_at = NOW() WHERE id = $2 AND expert_id = $3",
		status,
		id,
		expertID,
	)
	if err != nil {
		return fmt.Errorf("update asset status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("asset not found or access denied")
	}
	return nil
}

func (r *AssetRepository) IncrCallCount(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(
		ctx,
		"UPDATE assets SET call_count = call_count + 1, updated_at = NOW() WHERE id = $1",
		id,
	)
	if err != nil {
		return fmt.Errorf("increment call count: %w", err)
	}
	return nil
}
