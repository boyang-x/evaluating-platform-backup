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

type TemplateRepository struct {
	pool *pgxpool.Pool
}

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

const tplSelectCols = ` id, expert_id, upload_batch_id, upload_batch_name, sub_type, name, description, content, variables,
    status, visibility, created_at, updated_at `

func scanTemplate(row interface {
	Scan(dest ...interface{}) error
}) (*model.Template, error) {
	var t model.Template
	var varsJSON []byte
	err := row.Scan(
		&t.ID,
		&t.ExpertID,
		&t.UploadBatchID,
		&t.UploadBatchName,
		&t.SubType,
		&t.Name,
		&t.Description,
		&t.Content,
		&varsJSON,
		&t.Status,
		&t.Visibility,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err == nil && len(varsJSON) > 0 {
		_ = json.Unmarshal(varsJSON, &t.Variables)
	}
	return &t, err
}

func (r *TemplateRepository) Create(ctx context.Context, t *model.Template) error {
	varsJSON, _ := json.Marshal(t.Variables)
	query := `
        INSERT INTO templates
            (id, expert_id, upload_batch_id, upload_batch_name, sub_type, name, description, content, variables, status, visibility)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
    `
	_, err := r.pool.Exec(ctx, query,
		t.ID,
		t.ExpertID,
		t.UploadBatchID,
		t.UploadBatchName,
		t.SubType,
		t.Name,
		t.Description,
		t.Content,
		varsJSON,
		t.Status,
		t.Visibility,
	)
	if err != nil {
		return fmt.Errorf("insert template: %w", err)
	}
	return nil
}

func (r *TemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	query := "SELECT" + tplSelectCols + "FROM templates WHERE id = $1"
	t, err := scanTemplate(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("query template: %w", err)
	}
	return t, nil
}

func (r *TemplateRepository) Update(ctx context.Context, t *model.Template) error {
	varsJSON, _ := json.Marshal(t.Variables)
	tag, err := r.pool.Exec(ctx,
		`UPDATE templates SET name=$1, description=$2, content=$3, variables=$4,
         sub_type=$5, updated_at=NOW()
         WHERE id=$6 AND expert_id=$7`,
		t.Name, t.Description, t.Content, varsJSON, t.SubType, t.ID, t.ExpertID,
	)
	if err != nil {
		return fmt.Errorf("update template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("template not found or access denied")
	}
	return nil
}

func (r *TemplateRepository) Delete(ctx context.Context, id, expertID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM templates WHERE id = $1 AND expert_id = $2", id, expertID,
	)
	if err != nil {
		return fmt.Errorf("delete template: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("template not found or access denied")
	}
	return nil
}

func (r *TemplateRepository) ListByExpert(ctx context.Context, expertID uuid.UUID, subType string, limit, offset int) ([]model.Template, int, error) {
	conds := []string{"expert_id = $1"}
	args := []interface{}{expertID}

	if subType != "" {
		args = append(args, subType)
		conds = append(conds, fmt.Sprintf("sub_type = $%d", len(args)))
	}
	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM templates "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count templates: %w", err)
	}

	limitN := len(args) + 1
	offsetN := len(args) + 2
	listArgs := append(args, limit, offset)
	query := fmt.Sprintf("SELECT%sFROM templates %s ORDER BY updated_at DESC LIMIT $%d OFFSET $%d",
		tplSelectCols, where, limitN, offsetN)

	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	templates := make([]model.Template, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, *t)
	}
	return templates, total, nil
}

func (r *TemplateRepository) ListPublished(ctx context.Context, subType string, limit, offset int) ([]model.Template, int, error) {
	conds := []string{}
	args := []interface{}{}

	if subType != "" {
		args = append(args, subType)
		conds = append(conds, fmt.Sprintf("sub_type = $%d", len(args)))
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM templates "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count available templates: %w", err)
	}

	limitN := len(args) + 1
	offsetN := len(args) + 2
	listArgs := append(args, limit, offset)
	query := fmt.Sprintf("SELECT%sFROM templates %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		tplSelectCols, where, limitN, offsetN)

	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list available templates: %w", err)
	}
	defer rows.Close()

	templates := make([]model.Template, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan template: %w", err)
		}
		templates = append(templates, *t)
	}
	return templates, total, nil
}
