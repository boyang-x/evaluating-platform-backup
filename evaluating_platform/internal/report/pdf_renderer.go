package report

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode"

	"github.com/go-pdf/fpdf"
)

type PDFRenderer struct {
	fontPath string
}

func NewPDFRenderer(fontPath string) *PDFRenderer {
	return &PDFRenderer{fontPath: fontPath}
}

func severityColor(severity string) (r, g, b int) {
	switch strings.ToLower(severity) {
	case "critical":
		return 180, 30, 30
	case "high":
		return 220, 80, 40
	case "medium":
		return 230, 160, 30
	case "low":
		return 60, 140, 200
	case "info":
		return 50, 140, 90
	default:
		return 120, 120, 120
	}
}

func riskLevelLabel(level string) string {
	labels := map[string]string{
		"critical": "严重风险",
		"high":     "攻击成功",
		"medium":   "可疑回答",
		"low":      "轻微风险",
		"info":     "良性回答",
	}
	if label, ok := labels[strings.ToLower(level)]; ok {
		return label
	}
	return level
}

func (r *PDFRenderer) Render(report *StandardReport) ([]byte, error) {
	if report == nil {
		return nil, fmt.Errorf("report is nil")
	}

	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 20)

	fontFamily := "Helvetica"
	if r.registerCJKFont(pdf) {
		fontFamily = "CJK"
	}

	pdf.AddPage()
	r.renderTitlePage(pdf, fontFamily, report)

	pdf.AddPage()
	r.renderSummarySection(pdf, fontFamily, report)
	r.renderRiskSection(pdf, fontFamily, report)
	r.renderFindingsSection(pdf, fontFamily, report)
	r.renderAttackExamplesSection(pdf, fontFamily, report)
	r.renderMetricsSection(pdf, fontFamily, report)
	r.renderRecommendationsSection(pdf, fontFamily, report)

	if pdf.Err() {
		return nil, fmt.Errorf("pdf generation error: %w", pdf.Error())
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output error: %w", err)
	}
	return buf.Bytes(), nil
}

func sanitizePDFText(text string) string {
	if text == "" {
		return ""
	}

	sanitized := strings.Map(func(r rune) rune {
		switch {
		case r == '\n' || r == '\r':
			return r
		case r == '\t':
			return ' '
		case r < 32:
			return -1
		case r > unicode.MaxRune:
			return -1
		case r > 0xFFFF:
			return -1
		case unicode.Is(unicode.Cs, r):
			return -1
		default:
			return r
		}
	}, text)

	return strings.TrimSpace(sanitized)
}

func (r *PDFRenderer) registerCJKFont(pdf *fpdf.Fpdf) bool {
	if r.fontPath != "" {
		if _, err := os.Stat(r.fontPath); err == nil {
			pdf.AddUTF8Font("CJK", "", r.fontPath)
			pdf.AddUTF8Font("CJK", "B", r.fontPath)
			if !pdf.Err() {
				return true
			}
			pdf.ClearError()
		}
	}

	if len(defaultCJKFont) > 0 {
		pdf.AddUTF8FontFromBytes("CJK", "", defaultCJKFont)
		pdf.AddUTF8FontFromBytes("CJK", "B", defaultCJKFont)
		if !pdf.Err() {
			return true
		}
		pdf.ClearError()
	}

	return false
}

func (r *PDFRenderer) renderTitlePage(pdf *fpdf.Fpdf, font string, report *StandardReport) {
	pdf.Ln(50)
	pdf.SetFont(font, "B", 26)
	pdf.SetTextColor(30, 30, 30)
	pdf.MultiCell(0, 12, sanitizePDFText(report.Title), "", "C", false)
	pdf.Ln(10)

	pdf.SetDrawColor(60, 60, 60)
	pdf.SetLineWidth(0.5)
	x := pdf.GetX()
	y := pdf.GetY()
	pdf.Line(x+40, y, x+140, y)
	pdf.Ln(10)

	pdf.SetFont(font, "", 12)
	pdf.SetTextColor(80, 80, 80)
	pdf.CellFormat(0, 8, sanitizePDFText(fmt.Sprintf("Assessment Target: %s", report.AssessmentTarget)), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 8, sanitizePDFText(fmt.Sprintf("Assessment Time: %s", report.AssessmentTime)), "", 1, "C", false, 0, "")

	pdf.Ln(15)
	cr, cg, cb := severityColor(report.RiskLevel)
	pdf.SetFont(font, "B", 16)
	pdf.SetTextColor(cr, cg, cb)
	pdf.CellFormat(0, 10, sanitizePDFText(fmt.Sprintf("Risk Level: %s  |  Score: %d/100", riskLevelLabel(report.RiskLevel), report.RiskScore)), "", 1, "C", false, 0, "")
}

func (r *PDFRenderer) renderSummarySection(pdf *fpdf.Fpdf, font string, report *StandardReport) {
	r.sectionTitle(pdf, font, "1. Assessment Summary")
	pdf.SetFont(font, "", 11)
	pdf.SetTextColor(40, 40, 40)
	pdf.MultiCell(0, 6, sanitizePDFText(report.Summary), "", "L", false)
	pdf.Ln(4)

	if report.Scope != "" {
		r.subTitle(pdf, font, "Scope")
		pdf.SetFont(font, "", 10)
		pdf.MultiCell(0, 5.5, sanitizePDFText(report.Scope), "", "L", false)
		pdf.Ln(3)
	}

	if report.Methodology != "" {
		r.subTitle(pdf, font, "Methodology")
		pdf.SetFont(font, "", 10)
		pdf.MultiCell(0, 5.5, sanitizePDFText(report.Methodology), "", "L", false)
		pdf.Ln(3)
	}
}

func (r *PDFRenderer) renderRiskSection(pdf *fpdf.Fpdf, font string, report *StandardReport) {
	r.sectionTitle(pdf, font, "2. Risk Level & Score")
	cr, cg, cb := severityColor(report.RiskLevel)

	pdf.SetFont(font, "B", 12)
	pdf.SetTextColor(cr, cg, cb)
	pdf.CellFormat(40, 8, "Risk Level:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, sanitizePDFText(riskLevelLabel(report.RiskLevel)), "", 1, "L", false, 0, "")

	pdf.SetTextColor(40, 40, 40)
	pdf.SetFont(font, "B", 12)
	pdf.CellFormat(40, 8, "Risk Score:", "", 0, "L", false, 0, "")
	pdf.SetFont(font, "", 12)
	pdf.CellFormat(0, 8, fmt.Sprintf("%d / 100", report.RiskScore), "", 1, "L", false, 0, "")
	pdf.Ln(4)
}

func (r *PDFRenderer) renderFindingsSection(pdf *fpdf.Fpdf, font string, report *StandardReport) {
	r.sectionTitle(pdf, font, "3. Findings")

	if len(report.Findings) == 0 {
		pdf.SetFont(font, "", 11)
		pdf.SetTextColor(100, 100, 100)
		pdf.CellFormat(0, 8, "No findings reported.", "", 1, "L", false, 0, "")
		pdf.Ln(4)
		return
	}

	for i, finding := range report.Findings {
		r.checkPageBreak(pdf, 50)
		cr, cg, cb := severityColor(finding.Severity)

		pdf.SetFont(font, "B", 11)
		pdf.SetTextColor(30, 30, 30)
		pdf.CellFormat(0, 7, sanitizePDFText(fmt.Sprintf("%d. %s", i+1, finding.Title)), "", 0, "L", false, 0, "")
		pdf.SetTextColor(cr, cg, cb)
		pdf.CellFormat(0, 7, sanitizePDFText(fmt.Sprintf("[%s]", riskLevelLabel(finding.Severity))), "", 1, "R", false, 0, "")

		pdf.SetTextColor(40, 40, 40)
		pdf.SetFont(font, "", 10)

		if finding.Description != "" {
			pdf.SetFont(font, "B", 10)
			pdf.CellFormat(0, 5.5, "问题摘要", "", 1, "L", false, 0, "")
			pdf.SetFont(font, "", 10)
			pdf.MultiCell(0, 5.5, sanitizePDFText(finding.Description), "", "L", false)
			pdf.Ln(1)
		}
		if finding.Evidence != "" {
			pdf.SetFont(font, "B", 10)
			pdf.CellFormat(0, 5.5, "模型原回答", "", 1, "L", false, 0, "")
			pdf.SetFont(font, "", 10)
			pdf.MultiCell(0, 5.5, sanitizePDFText(finding.Evidence), "", "L", false)
		}
		pdf.Ln(4)
	}
}

func (r *PDFRenderer) renderAttackExamplesSection(pdf *fpdf.Fpdf, font string, report *StandardReport) {
	r.checkPageBreak(pdf, 40)
	r.sectionTitle(pdf, font, "4. Successful Attack Examples")

	if len(report.AttackExamples) == 0 {
		pdf.SetFont(font, "", 11)
		pdf.SetTextColor(100, 100, 100)
		pdf.CellFormat(0, 8, "No successful attack examples captured.", "", 1, "L", false, 0, "")
		pdf.Ln(4)
		return
	}

	for i, example := range report.AttackExamples {
		r.checkPageBreak(pdf, 45)
		cr, cg, cb := severityColor(example.Severity)

		pdf.SetFont(font, "B", 11)
		pdf.SetTextColor(cr, cg, cb)
		pdf.CellFormat(0, 7, sanitizePDFText(fmt.Sprintf("%d. %s", i+1, strings.ToUpper(example.Severity))), "", 1, "L", false, 0, "")

		pdf.SetTextColor(40, 40, 40)
		pdf.SetFont(font, "B", 10)
		pdf.CellFormat(0, 5.5, "Sample Question", "", 1, "L", false, 0, "")
		pdf.SetFont(font, "", 10)
		pdf.MultiCell(0, 5.5, sanitizePDFText(example.SampleQuestion), "", "L", false)
		pdf.Ln(1)

		pdf.SetFont(font, "B", 10)
		pdf.CellFormat(0, 5.5, "Model Response", "", 1, "L", false, 0, "")
		pdf.SetFont(font, "", 10)
		pdf.MultiCell(0, 5.5, sanitizePDFText(example.ModelResponse), "", "L", false)

		if strings.TrimSpace(example.Reason) != "" {
			pdf.Ln(1)
			pdf.SetFont(font, "B", 10)
			pdf.CellFormat(0, 5.5, "Why It Counts", "", 1, "L", false, 0, "")
			pdf.SetFont(font, "", 10)
			pdf.MultiCell(0, 5.5, sanitizePDFText(example.Reason), "", "L", false)
		}
		pdf.Ln(4)
	}
}

func (r *PDFRenderer) renderMetricsSection(pdf *fpdf.Fpdf, font string, report *StandardReport) {
	r.checkPageBreak(pdf, 50)
	r.sectionTitle(pdf, font, "5. Assessment Metrics")

	m := report.Metrics
	pdf.SetFont(font, "", 11)
	pdf.SetTextColor(40, 40, 40)

	rows := []struct {
		label string
		value string
	}{
		{"Total Tests", fmt.Sprintf("%d", m.TotalTests)},
		{"Successful Attacks", fmt.Sprintf("%d", m.SuccessCount)},
		{"Success Rate", fmt.Sprintf("%.1f%%", m.SuccessRate*100)},
		{"Risk Score", fmt.Sprintf("%d / 100", m.RiskScore)},
	}

	for _, row := range rows {
		pdf.SetFont(font, "B", 10)
		pdf.CellFormat(60, 7, sanitizePDFText(row.label), "1", 0, "L", false, 0, "")
		pdf.SetFont(font, "", 10)
		pdf.CellFormat(50, 7, sanitizePDFText(row.value), "1", 1, "L", false, 0, "")
	}
	pdf.Ln(4)
}

func (r *PDFRenderer) renderRecommendationsSection(pdf *fpdf.Fpdf, font string, report *StandardReport) {
	r.checkPageBreak(pdf, 30)
	r.sectionTitle(pdf, font, "6. Recommendations")

	if len(report.Recommendations) == 0 {
		pdf.SetFont(font, "", 11)
		pdf.SetTextColor(100, 100, 100)
		pdf.CellFormat(0, 8, "No recommendations.", "", 1, "L", false, 0, "")
		return
	}

	pdf.SetFont(font, "", 11)
	pdf.SetTextColor(40, 40, 40)
	for i, rec := range report.Recommendations {
		r.checkPageBreak(pdf, 10)
		pdf.MultiCell(0, 6, sanitizePDFText(fmt.Sprintf("%d. %s", i+1, rec)), "", "L", false)
		pdf.Ln(2)
	}
}

func (r *PDFRenderer) sectionTitle(pdf *fpdf.Fpdf, font, title string) {
	pdf.Ln(4)
	pdf.SetFont(font, "B", 14)
	pdf.SetTextColor(30, 60, 120)
	pdf.CellFormat(0, 9, sanitizePDFText(title), "", 1, "L", false, 0, "")
	pdf.SetDrawColor(30, 60, 120)
	pdf.SetLineWidth(0.3)
	y := pdf.GetY()
	pdf.Line(15, y, 195, y)
	pdf.Ln(4)
}

func (r *PDFRenderer) subTitle(pdf *fpdf.Fpdf, font, title string) {
	pdf.SetFont(font, "B", 11)
	pdf.SetTextColor(50, 50, 50)
	pdf.CellFormat(0, 7, sanitizePDFText(title), "", 1, "L", false, 0, "")
}

func (r *PDFRenderer) checkPageBreak(pdf *fpdf.Fpdf, requiredHeight float64) {
	_, pageH := pdf.GetPageSize()
	_, _, _, marginBottom := pdf.GetMargins()
	if pdf.GetY()+requiredHeight > pageH-marginBottom {
		pdf.AddPage()
	}
}
