package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

// ReportRepository 报告数据访问层
type ReportRepository struct {
	pool *pgxpool.Pool
}

// NewReportRepository 创建报告仓库
func NewReportRepository(pool *pgxpool.Pool) *ReportRepository {
	return &ReportRepository{pool: pool}
}

// Create 创建报告
func (r *ReportRepository) Create(ctx context.Context, report *model.Report) error {
	findingsJSON, _ := json.Marshal(report.Findings)
	metricsJSON, _ := json.Marshal(report.Metrics)
	query := `
		INSERT INTO reports (id, assessment_id, title, summary, risk_level, findings, metrics, raw_content, pdf_url, html_url)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.pool.Exec(ctx, query,
		report.ID, report.AssessmentID, report.Title, report.Summary,
		report.RiskLevel, findingsJSON, metricsJSON, report.RawContent,
		report.PDFURL, report.HTMLURL)
	if err != nil {
		return fmt.Errorf("insert report: %w", err)
	}
	return nil
}

// GetByAssessmentID 根据评估 ID 查询报告
func (r *ReportRepository) GetByAssessmentID(ctx context.Context, assessmentID uuid.UUID) (*model.Report, error) {
	query := `
		SELECT id, assessment_id, title, summary, risk_level, findings, metrics, raw_content, pdf_url, html_url, created_at
		FROM reports WHERE assessment_id = $1
	`
	var report model.Report
	var findingsJSON, metricsJSON []byte
	err := r.pool.QueryRow(ctx, query, assessmentID).Scan(
		&report.ID, &report.AssessmentID, &report.Title, &report.Summary,
		&report.RiskLevel, &findingsJSON, &metricsJSON, &report.RawContent,
		&report.PDFURL, &report.HTMLURL, &report.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query report: %w", err)
	}
	json.Unmarshal(findingsJSON, &report.Findings)
	json.Unmarshal(metricsJSON, &report.Metrics)
	return &report, nil
}

// GetByID 根据报告 ID 查询
func (r *ReportRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Report, error) {
	query := `
		SELECT id, assessment_id, title, summary, risk_level, findings, metrics, raw_content, pdf_url, html_url, created_at
		FROM reports WHERE id = $1
	`
	var report model.Report
	var findingsJSON, metricsJSON []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&report.ID, &report.AssessmentID, &report.Title, &report.Summary,
		&report.RiskLevel, &findingsJSON, &metricsJSON, &report.RawContent,
		&report.PDFURL, &report.HTMLURL, &report.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query report: %w", err)
	}
	json.Unmarshal(findingsJSON, &report.Findings)
	json.Unmarshal(metricsJSON, &report.Metrics)
	return &report, nil
}

// UpdatePDFURL updates the persisted PDF path for an existing report.
func (r *ReportRepository) UpdatePDFURL(ctx context.Context, id uuid.UUID, pdfURL string) error {
	query := `UPDATE reports SET pdf_url = $2 WHERE id = $1`
	if _, err := r.pool.Exec(ctx, query, id, pdfURL); err != nil {
		return fmt.Errorf("update report pdf_url: %w", err)
	}
	return nil
}
