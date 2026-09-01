package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

const maclawHubConfigID = "default"

type MaclawHubConfigRepository struct {
	pool *pgxpool.Pool
}

func NewMaclawHubConfigRepository(pool *pgxpool.Pool) *MaclawHubConfigRepository {
	return &MaclawHubConfigRepository{pool: pool}
}

func (r *MaclawHubConfigRepository) GetHubConfig(ctx context.Context) (*model.MaclawHubConfigRecord, error) {
	const query = `
		SELECT id, encrypted_config, config_key_id, updated_by, created_at, updated_at
		FROM maclaw_hub_configs
		WHERE id = $1
	`
	var out model.MaclawHubConfigRecord
	err := r.pool.QueryRow(ctx, query, maclawHubConfigID).Scan(
		&out.ID,
		&out.EncryptedConfig,
		&out.ConfigKeyID,
		&out.UpdatedBy,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query maclaw hub config: %w", err)
	}
	return &out, nil
}

func (r *MaclawHubConfigRepository) UpsertHubConfig(ctx context.Context, encrypted []byte, keyID string, updatedBy *uuid.UUID) error {
	const query = `
		INSERT INTO maclaw_hub_configs (id, encrypted_config, config_key_id, updated_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			encrypted_config = EXCLUDED.encrypted_config,
			config_key_id = EXCLUDED.config_key_id,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
	`
	if _, err := r.pool.Exec(ctx, query, maclawHubConfigID, encrypted, keyID, updatedBy); err != nil {
		return fmt.Errorf("upsert maclaw hub config: %w", err)
	}
	return nil
}
