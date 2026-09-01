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

const maclawDefaultModelConfigID = "default"

type MaclawModelConfigRepository struct {
	pool *pgxpool.Pool
}

func NewMaclawModelConfigRepository(pool *pgxpool.Pool) *MaclawModelConfigRepository {
	return &MaclawModelConfigRepository{pool: pool}
}

func (r *MaclawModelConfigRepository) GetDefault(ctx context.Context) (*model.MaclawModelDefault, error) {
	const query = `
		SELECT id, encrypted_config, config_key_id, updated_by, created_at, updated_at
		FROM maclaw_model_defaults
		WHERE id = $1
	`
	var out model.MaclawModelDefault
	err := r.pool.QueryRow(ctx, query, maclawDefaultModelConfigID).Scan(
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
		return nil, fmt.Errorf("query maclaw default model config: %w", err)
	}
	return &out, nil
}

func (r *MaclawModelConfigRepository) UpsertDefault(ctx context.Context, encrypted []byte, keyID string, updatedBy *uuid.UUID) error {
	const query = `
		INSERT INTO maclaw_model_defaults (id, encrypted_config, config_key_id, updated_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			encrypted_config = EXCLUDED.encrypted_config,
			config_key_id = EXCLUDED.config_key_id,
			updated_by = EXCLUDED.updated_by,
			updated_at = NOW()
	`
	if _, err := r.pool.Exec(ctx, query, maclawDefaultModelConfigID, encrypted, keyID, updatedBy); err != nil {
		return fmt.Errorf("upsert maclaw default model config: %w", err)
	}
	return nil
}
