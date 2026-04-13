package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evaluating_platform/internal/model"
	reportpkg "evaluating_platform/internal/report"
	"evaluating_platform/internal/repository"
	"evaluating_platform/pkg/storage"
)

// ReportHandler serves report retrieval and PDF download.
type ReportHandler struct {
	reportRepo *repository.ReportRepository
	store      *storage.MinIOClient
}

func NewReportHandler(reportRepo *repository.ReportRepository, store *storage.MinIOClient) *ReportHandler {
	return &ReportHandler{reportRepo: reportRepo, store: store}
}

func (h *ReportHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	rpt, err := h.findReport(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	c.JSON(http.StatusOK, rpt)
}

func (h *ReportHandler) Download(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	rpt, err := h.findReport(c, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}

	if rpt.PDFURL != "" && h.store != nil {
		data, err := h.store.Download(c.Request.Context(), rpt.PDFURL)
		if err == nil {
			h.writePDFResponse(c, rpt.ID, data)
			return
		}
	}

	data, err := h.renderAndPersistPDF(c.Request.Context(), rpt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("download pdf failed: %v", err)})
		return
	}

	h.writePDFResponse(c, rpt.ID, data)
}

func (h *ReportHandler) findReport(c *gin.Context, id uuid.UUID) (*model.Report, error) {
	rpt, err := h.reportRepo.GetByID(c.Request.Context(), id)
	if err == nil {
		return rpt, nil
	}
	return h.reportRepo.GetByAssessmentID(c.Request.Context(), id)
}

func (h *ReportHandler) writePDFResponse(c *gin.Context, reportID uuid.UUID, data []byte) {
	filename := fmt.Sprintf("report-%s.pdf", reportID.String())
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", data)
}

func (h *ReportHandler) renderAndPersistPDF(ctx context.Context, rpt *model.Report) ([]byte, error) {
	renderer := reportpkg.NewPDFRenderer("")
	stdReport := buildDownloadableStandardReport(rpt)

	data, err := renderer.Render(stdReport)
	if err != nil {
		return nil, err
	}

	if h.store == nil {
		return data, nil
	}

	path := fmt.Sprintf("reports/%s/%s.pdf", rpt.AssessmentID.String(), rpt.ID.String())
	if err := h.store.Upload(ctx, path, data, "application/pdf"); err == nil {
		rpt.PDFURL = path
		_ = h.reportRepo.UpdatePDFURL(ctx, rpt.ID, path)
	}

	return data, nil
}

func buildDownloadableStandardReport(rpt *model.Report) *reportpkg.StandardReport {
	findings := make([]reportpkg.StandardFinding, 0, len(rpt.Findings))
	for _, finding := range rpt.Findings {
		findings = append(findings, reportpkg.StandardFinding{
			ID:          finding.ID,
			Title:       finding.Title,
			Severity:    finding.Severity,
			Category:    finding.Category,
			Description: finding.Description,
			Evidence:    finding.Evidence,
			Suggestion:  finding.Suggestion,
		})
	}

	examples := make([]reportpkg.StandardAttackExample, 0, len(rpt.AttackExamples))
	for _, example := range rpt.AttackExamples {
		examples = append(examples, reportpkg.StandardAttackExample{
			SampleQuestion: example.SampleQuestion,
			ModelResponse:  example.ModelResponse,
			Severity:       example.Severity,
			Reason:         example.Reason,
		})
	}

	return &reportpkg.StandardReport{
		Title:            rpt.Title,
		AssessmentTarget: rpt.AssessmentID.String(),
		AssessmentTime:   rpt.CreatedAt.In(time.Local).Format("2006-01-02 15:04:05"),
		RiskLevel:        rpt.RiskLevel,
		RiskScore:        int(rpt.Metrics.RiskScore),
		Summary:          rpt.Summary,
		Scope:            rpt.Scope,
		Methodology:      rpt.Methodology,
		Findings:         findings,
		AttackExamples:   examples,
		Metrics: reportpkg.StandardMetrics{
			TotalTests:   rpt.Metrics.TotalTests,
			SuccessCount: rpt.Metrics.SuccessCount,
			SuccessRate:  rpt.Metrics.SuccessRate,
			RiskScore:    int(rpt.Metrics.RiskScore),
		},
		Recommendations: rpt.Recommendations,
	}
}
