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

type MaclawTargetConfigRepository struct {
	pool *pgxpool.Pool
}

func NewMaclawTargetConfigRepository(pool *pgxpool.Pool) *MaclawTargetConfigRepository {
	return &MaclawTargetConfigRepository{pool: pool}
}

func (r *MaclawTargetConfigRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.MaclawTargetConfig, error) {
	const query = `
		SELECT id, user_id, encrypted_config, config_key_id, created_at, updated_at
		FROM maclaw_target_configs
		WHERE user_id = $1
	`
	var out model.MaclawTargetConfig
	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&out.ID,
		&out.UserID,
		&out.EncryptedConfig,
		&out.ConfigKeyID,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query maclaw target config: %w", err)
	}
	return &out, nil
}

func (r *MaclawTargetConfigRepository) Upsert(ctx context.Context, record model.MaclawTargetConfig) (*model.MaclawTargetConfig, error) {
	if record.ID == uuid.Nil {
		record.ID = uuid.New()
	}
	const query = `
		INSERT INTO maclaw_target_configs (id, user_id, encrypted_config, config_key_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id) DO UPDATE SET
			encrypted_config = EXCLUDED.encrypted_config,
			config_key_id = EXCLUDED.config_key_id,
			updated_at = NOW()
		RETURNING id, user_id, encrypted_config, config_key_id, created_at, updated_at
	`
	var out model.MaclawTargetConfig
	if err := r.pool.QueryRow(ctx, query, record.ID, record.UserID, record.EncryptedConfig, record.ConfigKeyID).Scan(
		&out.ID,
		&out.UserID,
		&out.EncryptedConfig,
		&out.ConfigKeyID,
		&out.CreatedAt,
		&out.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("upsert maclaw target config: %w", err)
	}
	return &out, nil
}
