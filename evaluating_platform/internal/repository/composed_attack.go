package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

type ComposedAttackRepository struct {
	pool *pgxpool.Pool
}

func NewComposedAttackRepository(pool *pgxpool.Pool) *ComposedAttackRepository {
	return &ComposedAttackRepository{pool: pool}
}

const composedAttackSelectCols = ` id, expert_id, sub_type, name, description, storage_path, file_hash,
	sample_count, file_size, status, visibility,
	created_at, updated_at `

func scanComposedAttack(row interface {
	Scan(dest ...interface{}) error
}) (*model.ComposedAttack, error) {
	var item model.ComposedAttack
	err := row.Scan(
		&item.ID, &item.ExpertID, &item.SubType, &item.Name, &item.Description,
		&item.StoragePath, &item.FileHash,
		&item.SampleCount, &item.FileSize,
		&item.Status, &item.Visibility, &item.CreatedAt, &item.UpdatedAt,
	)
	return &item, err
}

func (r *ComposedAttackRepository) Create(ctx context.Context, item *model.ComposedAttack) error {
	query := `
		INSERT INTO composed_attacks
			(id, expert_id, sub_type, name, description, storage_path, file_hash,
			 sample_count, file_size, status, visibility)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`
	_, err := r.pool.Exec(ctx, query,
		item.ID, item.ExpertID, item.SubType, item.Name, item.Description,
		item.StoragePath, item.FileHash,
		item.SampleCount, item.FileSize, item.Status, item.Visibility,
	)
	if err != nil {
		return fmt.Errorf("insert composed_attack: %w", err)
	}
	return nil
}

func (r *ComposedAttackRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ComposedAttack, error) {
	query := "SELECT" + composedAttackSelectCols + "FROM composed_attacks WHERE id = $1"
	item, err := scanComposedAttack(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("query composed_attack: %w", err)
	}
	return item, nil
}

func (r *ComposedAttackRepository) ListByExpert(ctx context.Context, expertID uuid.UUID, subType string, limit, offset int) ([]model.ComposedAttack, int, error) {
	conds := []string{"expert_id = $1"}
	args := []interface{}{expertID}

	if subType != "" {
		args = append(args, subType)
		conds = append(conds, fmt.Sprintf("sub_type = $%d", len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM composed_attacks "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count composed_attacks: %w", err)
	}

	limitN := len(args) + 1
	offsetN := len(args) + 2
	listArgs := append(args, limit, offset)
	query := fmt.Sprintf("SELECT%sFROM composed_attacks %s ORDER BY updated_at DESC LIMIT $%d OFFSET $%d",
		composedAttackSelectCols, where, limitN, offsetN)

	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list composed_attacks: %w", err)
	}
	defer rows.Close()

	items := make([]model.ComposedAttack, 0)
	for rows.Next() {
		item, err := scanComposedAttack(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan composed_attack: %w", err)
		}
		items = append(items, *item)
	}
	return items, total, nil
}

func (r *ComposedAttackRepository) Delete(ctx context.Context, id, expertID uuid.UUID) (string, error) {
	var storagePath string
	err := r.pool.QueryRow(ctx,
		"DELETE FROM composed_attacks WHERE id = $1 AND expert_id = $2 RETURNING storage_path",
		id, expertID,
	).Scan(&storagePath)
	if err != nil {
		return "", fmt.Errorf("delete composed_attack: %w", err)
	}
	return storagePath, nil
}

func (r *ComposedAttackRepository) ListPublished(ctx context.Context, subType string, limit, offset int) ([]model.ComposedAttack, int, error) {
	where, args := platformDataPublishedWhereClause("sub_type", subType, 0)

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM composed_attacks "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count available composed_attacks: %w", err)
	}

	limitN := len(args) + 1
	offsetN := len(args) + 2
	listArgs := append(args, limit, offset)
	query := fmt.Sprintf("SELECT%sFROM composed_attacks %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		composedAttackSelectCols, where, limitN, offsetN)

	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list available composed_attacks: %w", err)
	}
	defer rows.Close()

	items := make([]model.ComposedAttack, 0)
	for rows.Next() {
		item, err := scanComposedAttack(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan composed_attack: %w", err)
		}
		items = append(items, *item)
	}
	return items, total, nil
}
