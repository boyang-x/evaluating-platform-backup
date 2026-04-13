package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

// AssessmentRepository 评估任务数据访问层
type AssessmentRepository struct {
	pool *pgxpool.Pool
}

// NewAssessmentRepository 创建评估仓库
func NewAssessmentRepository(pool *pgxpool.Pool) *AssessmentRepository {
	return &AssessmentRepository{pool: pool}
}

// Create 创建评估任务
func (r *AssessmentRepository) Create(ctx context.Context, assessment *model.Assessment) error {
	planJSON, _ := json.Marshal(assessment.Plan)
	query := `
		INSERT INTO assessments (id, user_id, name, description, goal, target_id, template_id, status, plan)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		assessment.ID, assessment.UserID, assessment.Name, assessment.Description,
		assessment.Goal, assessment.TargetID, assessment.TemplateID, assessment.Status, planJSON)
	if err != nil {
		return fmt.Errorf("insert assessment: %w", err)
	}
	return nil
}

// GetByID 根据 ID 查询评估任务
func (r *AssessmentRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Assessment, error) {
	query := `
		SELECT id, user_id, name, description, goal, target_id, template_id, status, plan, error_msg,
		       created_at, updated_at, completed_at
		FROM assessments WHERE id = $1
	`
	var assessment model.Assessment
	var planJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&assessment.ID, &assessment.UserID, &assessment.Name, &assessment.Description,
		&assessment.Goal, &assessment.TargetID, &assessment.TemplateID, &assessment.Status,
		&planJSON, &assessment.ErrorMsg, &assessment.CreatedAt, &assessment.UpdatedAt, &assessment.CompletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query assessment: %w", err)
	}
	json.Unmarshal(planJSON, &assessment.Plan)
	return &assessment, nil
}

// ListByUser 列出用户的评估任务（分页）
func (r *AssessmentRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.Assessment, int, error) {
	// 查询总数
	var total int
	err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM assessments WHERE user_id = $1", userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count assessments: %w", err)
	}

	// 查询列表
	query := `
		SELECT id, user_id, name, description, goal, target_id, template_id, status, plan, error_msg,
		       created_at, updated_at, completed_at
		FROM assessments WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query assessments: %w", err)
	}
	defer rows.Close()

	var assessments []*model.Assessment
	for rows.Next() {
		var a model.Assessment
		var planJSON []byte
		err := rows.Scan(
			&a.ID, &a.UserID, &a.Name, &a.Description, &a.Goal, &a.TargetID, &a.TemplateID,
			&a.Status, &planJSON, &a.ErrorMsg, &a.CreatedAt, &a.UpdatedAt, &a.CompletedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan assessment: %w", err)
		}
		json.Unmarshal(planJSON, &a.Plan)
		assessments = append(assessments, &a)
	}
	return assessments, total, nil
}

// UpdateStatus 更新评估状态
func (r *AssessmentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.AssessmentStatus, errorMsg string) error {
	query := `UPDATE assessments SET status = $1, error_msg = $2, updated_at = NOW() WHERE id = $3`
	_, err := r.pool.Exec(ctx, query, status, errorMsg, id)
	if err != nil {
		return fmt.Errorf("update status: %w", err)
	}
	return nil
}

// Complete 标记评估完成
func (r *AssessmentRepository) Complete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE assessments SET status = $1, completed_at = NOW(), updated_at = NOW() WHERE id = $2`
	_, err := r.pool.Exec(ctx, query, model.StatusCompleted, id)
	if err != nil {
		return fmt.Errorf("complete assessment: %w", err)
	}
	return nil
}

// CreateLog 创建评估日志
func (r *AssessmentRepository) CreateLog(ctx context.Context, log *model.AssessmentLog) error {
	inputJSON, _ := json.Marshal(log.Input)
	outputJSON, _ := json.Marshal(log.Output)
	query := `
		INSERT INTO assessment_logs (id, assessment_id, iteration, tool_name, input, output, thinking, tokens_used, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.pool.Exec(ctx, query,
		log.ID, log.AssessmentID, log.Iteration, log.ToolName,
		inputJSON, outputJSON, log.Thinking, log.TokensUsed, log.DurationMs)
	if err != nil {
		return fmt.Errorf("insert log: %w", err)
	}
	return nil
}

// GetLogs 获取评估日志
func (r *AssessmentRepository) GetLogs(ctx context.Context, assessmentID uuid.UUID) ([]*model.AssessmentLog, error) {
	query := `
		SELECT id, assessment_id, iteration, tool_name, input, output, thinking, tokens_used, duration_ms, created_at
		FROM assessment_logs WHERE assessment_id = $1 ORDER BY iteration ASC
	`
	rows, err := r.pool.Query(ctx, query, assessmentID)
	if err != nil {
		return nil, fmt.Errorf("query logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.AssessmentLog
	for rows.Next() {
		var log model.AssessmentLog
		var inputJSON, outputJSON []byte
		err := rows.Scan(
			&log.ID, &log.AssessmentID, &log.Iteration, &log.ToolName,
			&inputJSON, &outputJSON, &log.Thinking, &log.TokensUsed, &log.DurationMs, &log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan log: %w", err)
		}
		log.Input = string(inputJSON)
		log.Output = string(outputJSON)
		logs = append(logs, &log)
	}
	return logs, nil
}

// CreateTargetSystem 创建目标系统
func (r *AssessmentRepository) CreateTargetSystem(ctx context.Context, target *model.TargetSystem) error {
	configJSON, _ := json.Marshal(target.Config)
	query := `
		INSERT INTO target_systems (id, user_id, name, type, endpoint, api_key, config)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.pool.Exec(ctx, query,
		target.ID, target.UserID, target.Name, target.Type, target.Endpoint, target.APIKey, configJSON)
	if err != nil {
		return fmt.Errorf("insert target system: %w", err)
	}
	return nil
}

// GetTargetSystem 获取目标系统
func (r *AssessmentRepository) GetTargetSystem(ctx context.Context, id uuid.UUID) (*model.TargetSystem, error) {
	query := `
		SELECT id, user_id, name, type, endpoint, api_key, config, created_at
		FROM target_systems WHERE id = $1
	`
	var target model.TargetSystem
	var configJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&target.ID, &target.UserID, &target.Name, &target.Type,
		&target.Endpoint, &target.APIKey, &configJSON, &target.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query target system: %w", err)
	}
	target.Config = string(configJSON)
	return &target, nil
}

// Cancel 取消评估任务（仅允许 pending 或 running 状态，且限定归属用户）
func (r *AssessmentRepository) Cancel(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `
		UPDATE assessments
		SET status = $1, error_msg = '用户取消', updated_at = NOW()
		WHERE id = $2 AND user_id = $3 AND status IN ('pending', 'running')
	`
	tag, err := r.pool.Exec(ctx, query, model.StatusCanceled, id, userID)
	if err != nil {
		return fmt.Errorf("cancel assessment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("assessment not found or not in a cancelable state")
	}
	return nil
}

// CreateOrGetTargetSystem 创建或获取目标系统（幂等）
func (r *AssessmentRepository) CreateOrGetTargetSystem(ctx context.Context, userID uuid.UUID, targetType, endpoint, apiKey string) (uuid.UUID, error) {
	// 先查询是否已存在
	query := `SELECT id FROM target_systems WHERE user_id = $1 AND type = $2 AND endpoint = $3 LIMIT 1`
	var id uuid.UUID
	err := r.pool.QueryRow(ctx, query, userID, targetType, endpoint).Scan(&id)
	if err == nil {
		return id, nil
	}

	// 不存在则创建
	target := &model.TargetSystem{
		ID:       uuid.New(),
		UserID:   userID,
		Name:     fmt.Sprintf("%s - %s", targetType, time.Now().Format("2006-01-02")),
		Type:     targetType,
		Endpoint: endpoint,
		APIKey:   apiKey,
		Config:   "{}",
	}
	if err := r.CreateTargetSystem(ctx, target); err != nil {
		return uuid.Nil, err
	}
	return target.ID, nil
}

// CountStats 统计平台总评估数和最近 7 天每天评估数（供管理员 dashboard 使用）
func (r *AssessmentRepository) CountStats(ctx context.Context) (int, []int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM assessments").Scan(&total); err != nil {
		return 0, nil, fmt.Errorf("count assessments: %w", err)
	}

	// 最近 7 天每天的评估数
	rows, err := r.pool.Query(ctx, `
		SELECT COALESCE(COUNT(*), 0)
		FROM generate_series(
			(NOW() - INTERVAL '6 days')::date,
			NOW()::date,
			INTERVAL '1 day'
		) AS day
		LEFT JOIN assessments ON DATE(assessments.created_at) = day
		GROUP BY day ORDER BY day
	`)
	if err != nil {
		return total, make([]int, 7), nil
	}
	defer rows.Close()

	weekly := make([]int, 0, 7)
	for rows.Next() {
		var n int
		rows.Scan(&n)
		weekly = append(weekly, n)
	}
	// 补齐至 7 个
	for len(weekly) < 7 {
		weekly = append(weekly, 0)
	}
	return total, weekly, nil
}
