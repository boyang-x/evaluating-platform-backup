package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

type SkillConfigValueRepository struct {
	pool *pgxpool.Pool
}

func NewSkillConfigValueRepository(pool *pgxpool.Pool) *SkillConfigValueRepository {
	return &SkillConfigValueRepository{pool: pool}
}

func scanSkillConfigValue(row interface {
	Scan(dest ...interface{}) error
}) (*model.SkillConfigValue, error) {
	var item model.SkillConfigValue
	if err := row.Scan(
		&item.SkillID,
		&item.FieldKey,
		&item.ValueCiphertext,
		&item.KeyID,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *SkillConfigValueRepository) ListBySkill(ctx context.Context, skillID uuid.UUID) ([]model.SkillConfigValue, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT skill_id, field_key, value_ciphertext, key_id, created_at, updated_at
		FROM skill_config_values
		WHERE skill_id = $1
		ORDER BY field_key ASC
	`, skillID)
	if err != nil {
		return nil, fmt.Errorf("list skill config values: %w", err)
	}
	defer rows.Close()

	items := make([]model.SkillConfigValue, 0)
	for rows.Next() {
		item, err := scanSkillConfigValue(rows)
		if err != nil {
			return nil, fmt.Errorf("scan skill config value: %w", err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func (r *SkillConfigValueRepository) Upsert(ctx context.Context, item *model.SkillConfigValue) error {
	return r.pool.QueryRow(ctx, `
		INSERT INTO skill_config_values (
			skill_id, field_key, value_ciphertext, key_id
		) VALUES ($1,$2,$3,$4)
		ON CONFLICT (skill_id, field_key)
		DO UPDATE SET
			value_ciphertext = EXCLUDED.value_ciphertext,
			key_id = EXCLUDED.key_id,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`,
		item.SkillID,
		item.FieldKey,
		item.ValueCiphertext,
		item.KeyID,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
}

func (r *SkillConfigValueRepository) Delete(ctx context.Context, skillID uuid.UUID, fieldKey string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM skill_config_values
		WHERE skill_id = $1 AND field_key = $2
	`, skillID, fieldKey)
	if err != nil {
		return fmt.Errorf("delete skill config value: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil
	}
	return nil
}

func (r *SkillConfigValueRepository) Get(ctx context.Context, skillID uuid.UUID, fieldKey string) (*model.SkillConfigValue, error) {
	item, err := scanSkillConfigValue(r.pool.QueryRow(ctx, `
		SELECT skill_id, field_key, value_ciphertext, key_id, created_at, updated_at
		FROM skill_config_values
		WHERE skill_id = $1 AND field_key = $2
	`, skillID, fieldKey))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get skill config value: %w", err)
	}
	return item, nil
}
