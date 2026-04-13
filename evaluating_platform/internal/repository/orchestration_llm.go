package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

// OrchestrationLLMRepository 编排 LLM 配置数据访问层
type OrchestrationLLMRepository struct {
	pool *pgxpool.Pool
}

func NewOrchestrationLLMRepository(pool *pgxpool.Pool) *OrchestrationLLMRepository {
	return &OrchestrationLLMRepository{pool: pool}
}

// GetByUserID 根据用户 ID 查询编排 LLM 配置
func (r *OrchestrationLLMRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*model.OrchestrationLLMConfig, error) {
	query := `
		SELECT id, user_id, base_url, api_key, model, created_at, updated_at
		FROM orchestration_llm_configs
		WHERE user_id = $1
	`
	row := r.pool.QueryRow(ctx, query, userID)

	var cfg model.OrchestrationLLMConfig
	err := row.Scan(
		&cfg.ID, &cfg.UserID, &cfg.BaseURL, &cfg.APIKey,
		&cfg.Model, &cfg.CreatedAt, &cfg.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query orchestration_llm_config: %w", err)
	}
	return &cfg, nil
}

// Upsert 创建或更新编排 LLM 配置（ON CONFLICT user_id DO UPDATE）
func (r *OrchestrationLLMRepository) Upsert(ctx context.Context, cfg *model.OrchestrationLLMConfig) error {
	query := `
		INSERT INTO orchestration_llm_configs (id, user_id, base_url, api_key, model)
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
		return fmt.Errorf("upsert orchestration_llm_config: %w", err)
	}
	return nil
}
