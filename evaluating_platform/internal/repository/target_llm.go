package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

// TargetLLMRepository 被测 LLM 配置数据访问层
type TargetLLMRepository struct {
	pool *pgxpool.Pool
}

func NewTargetLLMRepository(pool *pgxpool.Pool) *TargetLLMRepository {
	return &TargetLLMRepository{pool: pool}
}

// GetByUserID 根据用户 ID 查询被测 LLM 配置
func (r *TargetLLMRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.TargetLLMConfig, error) {
	query := `
		SELECT id, user_id, base_url, api_key, model, connector_type, created_at, updated_at
		FROM target_llm_configs
		WHERE user_id = $1
	`
	row := r.pool.QueryRow(ctx, query, userID)

	var cfg model.TargetLLMConfig
	err := row.Scan(
		&cfg.ID, &cfg.UserID, &cfg.BaseURL, &cfg.APIKey,
		&cfg.Model, &cfg.ConnectorType, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query target_llm_config: %w", err)
	}
	return &cfg, nil
}

// Upsert 创建或更新被测 LLM 配置（ON CONFLICT user_id DO UPDATE）
func (r *TargetLLMRepository) Upsert(ctx context.Context, cfg *model.TargetLLMConfig) error {
	query := `
		INSERT INTO target_llm_configs (id, user_id, base_url, api_key, model, connector_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) DO UPDATE SET
			base_url       = EXCLUDED.base_url,
			api_key        = EXCLUDED.api_key,
			model          = EXCLUDED.model,
			connector_type = EXCLUDED.connector_type,
			updated_at     = NOW()
		RETURNING id, created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, query,
		cfg.ID, cfg.UserID, cfg.BaseURL, cfg.APIKey, cfg.Model, cfg.ConnectorType,
	).Scan(&cfg.ID, &cfg.CreatedAt, &cfg.UpdatedAt)
	if err != nil {
		return fmt.Errorf("upsert target_llm_config: %w", err)
	}
	return nil
}
