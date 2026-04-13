package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

// AuxiliaryLLMRepository 辅助 LLM 配置数据访问层
type AuxiliaryLLMRepository struct {
	pool *pgxpool.Pool
}

func NewAuxiliaryLLMRepository(pool *pgxpool.Pool) *AuxiliaryLLMRepository {
	return &AuxiliaryLLMRepository{pool: pool}
}

// GetByUserID 根据用户 ID 查询辅助 LLM 配置
func (r *AuxiliaryLLMRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.AuxiliaryLLMConfig, error) {
	query := `
		SELECT id, user_id, base_url, api_key, model, created_at, updated_at
		FROM auxiliary_llm_configs
		WHERE user_id = $1
	`
	row := r.pool.QueryRow(ctx, query, userID)

	var cfg model.AuxiliaryLLMConfig
	err := row.Scan(
		&cfg.ID, &cfg.UserID, &cfg.BaseURL, &cfg.APIKey,
		&cfg.Model, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query auxiliary_llm_config: %w", err)
	}
	return &cfg, nil
}

// Upsert 创建或更新辅助 LLM 配置（ON CONFLICT user_id DO UPDATE）
func (r *AuxiliaryLLMRepository) Upsert(ctx context.Context, cfg *model.AuxiliaryLLMConfig) error {
	query := `
		INSERT INTO auxiliary_llm_configs (id, user_id, base_url, api_key, model)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			base_url   = EXCLUDED.base_url,
			api_key    = EXCLUDED.api_key,
			model      = EXCLUDED.model,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		cfg.ID, cfg.UserID, cfg.BaseURL, cfg.APIKey, cfg.Model,
	).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert auxiliary_llm_config: %w", err)
	}
	return nil
}
