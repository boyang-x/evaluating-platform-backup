package maclaw

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/google/uuid"
)

const (
	pdfContentBottomY = 310.0
	pdfFooterLineY    = 44.0
	pdfFooterTextY    = 26.0
)

type RedteamEvidenceRecord struct {
	ID             string
	Handle         string
	PlatformUserID uuid.UUID
	InstanceID     string
	SessionID      string
	RunID          string
	Kind           EvaluationEvidenceKind
	Title          string
	Summary        string
	Metadata       map[string]string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type RedteamReportRecord struct {
	ID              string
	Handle          string
	PlatformUserID  uuid.UUID
	InstanceID      string
	SessionID       string
	RunID           string
	Title           string
	Summary         string
	RiskLevel       string
	SafetyScore     *float64
	Findings        []EvaluationReportFinding
	EvidenceHandles []string
	Metadata        map[string]string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type RedteamArtifactStore interface {
	SaveEvidence(context.Context, RedteamEvidenceRecord) (*RedteamEvidenceRecord, error)
	ListEvidence(context.Context, uuid.UUID, EvaluationEvidenceQuery) ([]RedteamEvidenceRecord, error)
	GetEvidence(context.Context, uuid.UUID, string) (*RedteamEvidenceRecord, error)
	SaveReport(context.Context, RedteamReportRecord) (*RedteamReportRecord, error)
	GetReport(context.Context, uuid.UUID, string) (*RedteamReportRecord, error)
}

type RedteamArtifactService struct {
	store      RedteamArtifactStore
	now        func() time.Time
	handleSalt string
}

const maxRedteamArtifactTextRunes = 2000

var redteamArtifactSecretPattern = regexp.MustCompile(`(?i)(secret[-_a-z0-9]*|sk-[a-z0-9][a-z0-9_\-]{5,}|bearer\s+[a-z0-9._\-]+|api[_ -]?key\s*[:=]\s*\S+)`)

func NewRedteamArtifactService(store RedteamArtifactStore) *RedteamArtifactService {
	return &RedteamArtifactService{
		store:      store,
		now:        time.Now,
		handleSalt: "evaluating_platform_redteam_artifacts_v1",
	}
}

func (s *RedteamArtifactService) Enabled() bool {
	return s != nil && s.store != nil
}

func (s *RedteamArtifactService) SaveEvidence(ctx context.Context, userID uuid.UUID, instanceID string, in RedteamEvidenceInput) (*RedteamEvidenceOutput, error) {
	if !s.Enabled() {
		return nil, ErrNotConfigured
	}
	if userID == uuid.Nil {
		return nil, errors.New("platform user id is required")
	}
	kind := in.Kind
	if kind == "" {
		kind = EvaluationEvidenceKindArtifact
	}
	now := s.nowUTC()
	handle := s.safeHandle("redteam_evidence", userID.String(), in.RunID, string(kind), in.Title, in.Summary)
	record := RedteamEvidenceRecord{
		ID:             handle,
		Handle:         handle,
		PlatformUserID: userID,
		InstanceID:     strings.TrimSpace(instanceID),
		RunID:          strings.TrimSpace(in.RunID),
		Kind:           kind,
		Title:          sanitizeRedteamArtifactText(in.Title),
		Summary:        sanitizeRedteamArtifactText(in.Summary),
		Metadata:       sanitizeMetadata(in.Metadata),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	saved, err := s.store.SaveEvidence(ctx, record)
	if err != nil {
		return nil, err
	}
	return evidenceOutputFromRecord(saved), nil
}

func (s *RedteamArtifactService) ListEvidence(ctx context.Context, userID uuid.UUID, q EvaluationEvidenceQuery) ([]EvaluationEvidenceSummary, error) {
	if !s.Enabled() {
		return nil, ErrNotConfigured
	}
	if userID == uuid.Nil {
		return nil, errors.New("platform user id is required")
	}
	records, err := s.store.ListEvidence(ctx, userID, q)
	if err != nil {
		return nil, err
	}
	out := make([]EvaluationEvidenceSummary, 0, len(records))
	for i := range records {
		out = append(out, evidenceSummaryFromRecord(&records[i]))
	}
	return out, nil
}

func (s *RedteamArtifactService) GetEvidence(ctx context.Context, userID uuid.UUID, evidenceID string) (*EvaluationEvidenceSummary, error) {
	if !s.Enabled() {
		return nil, ErrNotConfigured
	}
	if userID == uuid.Nil {
		return nil, errors.New("platform user id is required")
	}
	record, err := s.store.GetEvidence(ctx, userID, strings.TrimSpace(evidenceID))
	if err != nil || record == nil {
		return nil, err
	}
	out := evidenceSummaryFromRecord(record)
	return &out, nil
}

func (s *RedteamArtifactService) CompileReport(ctx context.Context, userID uuid.UUID, instanceID string, in CompileRedteamReportInput) (*EvaluationReport, error) {
	if !s.Enabled() {
		return nil, ErrNotConfigured
	}
	if userID == uuid.Nil {
		return nil, errors.New("platform user id is required")
	}
	now := s.nowUTC()
	handle := s.safeHandle("redteam_report", userID.String(), in.RunID, in.Title, in.Summary)
	metadata := sanitizeMetadata(in.Metadata)
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["schema_version"] = "redteam_report_zh_v1"
	metadata["report_template"] = "redteam_report_pdf_layout_v2"
	record := RedteamReportRecord{
		ID:              handle,
		Handle:          handle,
		PlatformUserID:  userID,
		InstanceID:      strings.TrimSpace(instanceID),
		RunID:           strings.TrimSpace(in.RunID),
		Title:           firstNonEmptyString(sanitizeRedteamArtifactText(in.Title), "大模型安全评估报告"),
		Summary:         sanitizeRedteamArtifactText(in.Summary),
		RiskLevel:       sanitizeRedteamArtifactText(in.RiskLevel),
		SafetyScore:     in.SafetyScore,
		Findings:        sanitizeRedteamReportFindings(in.Findings),
		EvidenceHandles: append([]string(nil), in.EvidenceHandles...),
		Metadata:        metadata,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	saved, err := s.store.SaveReport(ctx, record)
	if err != nil {
		return nil, err
	}
	report := reportFromRecord(saved)
	return &report, nil
}

func sanitizeRedteamReportFindings(in []EvaluationReportFinding) []EvaluationReportFinding {
	if len(in) == 0 {
		return nil
	}
	out := make([]EvaluationReportFinding, 0, len(in))
	for _, item := range in {
		item.ID = sanitizeRedteamArtifactText(item.ID)
		item.Title = sanitizeRedteamArtifactText(item.Title)
		item.Severity = sanitizeRedteamArtifactText(item.Severity)
		item.Category = sanitizeRedteamArtifactText(item.Category)
		item.Description = sanitizeRedteamArtifactText(item.Description)
		item.Evidence = sanitizeRedteamArtifactText(item.Evidence)
		item.Suggestion = sanitizeRedteamArtifactText(item.Suggestion)
		item.Metadata = sanitizeMetadata(item.Metadata)
		out = append(out, item)
	}
	return out
}

func sanitizeRedteamArtifactText(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	for _, marker := range []string{
		"raw payload",
		"payload body",
		"raw prompt",
		"prompt body",
		"raw response",
		"target response body",
		"credential",
		"api key",
		"secret",
		"token",
		"local path",
		"evidence content",
	} {
		if strings.Contains(lower, marker) {
			return "[redacted sensitive text]"
		}
	}
	value = redteamArtifactSecretPattern.ReplaceAllString(value, "[redacted]")
	runes := []rune(value)
	if len(runes) > maxRedteamArtifactTextRunes {
		return string(runes[:maxRedteamArtifactTextRunes]) + "..."
	}
	return value
}

func (s *RedteamArtifactService) GetReport(ctx context.Context, userID uuid.UUID, reportID string) (*EvaluationReport, error) {
	if !s.Enabled() {
		return nil, ErrNotConfigured
	}
	if userID == uuid.Nil {
		return nil, errors.New("platform user id is required")
	}
	record, err := s.store.GetReport(ctx, userID, strings.TrimSpace(reportID))
	if err != nil || record == nil {
		return nil, err
	}
	report := reportFromRecord(record)
	return &report, nil
}

func (s *RedteamArtifactService) ExportReport(ctx context.Context, userID uuid.UUID, reportID, format string) (*EvaluationReportExport, error) {
	report, err := s.GetReport(ctx, userID, reportID)
	if err != nil || report == nil {
		return nil, err
	}
	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "pdf"
	}
	switch format {
	case "pdf":
		content := renderReportPDF(report)
		return &EvaluationReportExport{
			ReportID:    report.ID,
			Format:      "pdf",
			Filename:    safeReportFilename(report.ID, "pdf"),
			ContentType: "application/pdf",
			Content:     content,
		}, nil
	case "md", "markdown":
		content := renderReportMarkdown(report)
		return &EvaluationReportExport{
			ReportID:    report.ID,
			Format:      "markdown",
			Filename:    safeReportFilename(report.ID, "md"),
			ContentType: "text/markdown; charset=utf-8",
			Content:     content,
		}, nil
	case "json":
		data := mustMarshalJSON(report)
		return &EvaluationReportExport{
			ReportID:    report.ID,
			Format:      "json",
			Filename:    safeReportFilename(report.ID, "json"),
			ContentType: "application/json; charset=utf-8",
			Content:     data,
		}, nil
	default:
		return nil, errors.New("unsupported report export format")
	}
}

func (s *RedteamArtifactService) safeHandle(parts ...string) string {
	now := s.nowUTC().Format(time.RFC3339Nano)
	sum := sha256.Sum256([]byte(strings.Join(append([]string{s.handleSalt, now}, parts...), "\x00")))
	return parts[0] + "_" + hex.EncodeToString(sum[:])[:24]
}

func (s *RedteamArtifactService) nowUTC() time.Time {
	if s != nil && s.now != nil {
		return s.now().UTC()
	}
	return time.Now().UTC()
}

type PlatformArtifactGateway struct {
	GatewayClient
	artifacts *RedteamArtifactService
	userID    uuid.UUID
}

func NewPlatformArtifactGateway(upstream GatewayClient, artifacts *RedteamArtifactService, userID uuid.UUID) GatewayClient {
	return &PlatformArtifactGateway{GatewayClient: upstream, artifacts: artifacts, userID: userID}
}

func (g *PlatformArtifactGateway) GetEvaluationReport(ctx context.Context, reportID string) (*EvaluationReport, error) {
	if g != nil && g.artifacts != nil && g.artifacts.Enabled() {
		report, err := g.artifacts.GetReport(ctx, g.userID, reportID)
		if err != nil || report != nil {
			return report, err
		}
	}
	return g.GatewayClient.GetEvaluationReport(ctx, reportID)
}

func (g *PlatformArtifactGateway) ExportEvaluationReport(ctx context.Context, reportID, format string) (*EvaluationReportExport, error) {
	if g != nil && g.artifacts != nil && g.artifacts.Enabled() {
		out, err := g.artifacts.ExportReport(ctx, g.userID, reportID, format)
		if err != nil || out != nil {
			return out, err
		}
	}
	return g.GatewayClient.ExportEvaluationReport(ctx, reportID, format)
}

func (g *PlatformArtifactGateway) ListEvaluationEvidence(ctx context.Context, q EvaluationEvidenceQuery) ([]EvaluationEvidenceSummary, error) {
	if g != nil && g.artifacts != nil && g.artifacts.Enabled() {
		return g.artifacts.ListEvidence(ctx, g.userID, q)
	}
	return g.GatewayClient.ListEvaluationEvidence(ctx, q)
}

func (g *PlatformArtifactGateway) GetEvaluationEvidence(ctx context.Context, evidenceID string) (*EvaluationEvidenceSummary, error) {
	if g != nil && g.artifacts != nil && g.artifacts.Enabled() {
		out, err := g.artifacts.GetEvidence(ctx, g.userID, evidenceID)
		if err != nil || out != nil {
			return out, err
		}
	}
	return g.GatewayClient.GetEvaluationEvidence(ctx, evidenceID)
}

func evidenceOutputFromRecord(record *RedteamEvidenceRecord) *RedteamEvidenceOutput {
	if record == nil {
		return nil
	}
	return &RedteamEvidenceOutput{
		Handle:   record.Handle,
		Kind:     record.Kind,
		Title:    record.Title,
		Summary:  record.Summary,
		Metadata: sanitizeMetadata(record.Metadata),
	}
}

func evidenceSummaryFromRecord(record *RedteamEvidenceRecord) EvaluationEvidenceSummary {
	if record == nil {
		return EvaluationEvidenceSummary{}
	}
	return EvaluationEvidenceSummary{
		ID:         record.ID,
		InstanceID: record.InstanceID,
		SessionID:  record.SessionID,
		RunID:      record.RunID,
		Kind:       record.Kind,
		Title:      record.Title,
		Summary:    record.Summary,
		Handle:     record.Handle,
		Metadata:   sanitizeMetadata(record.Metadata),
		CreatedAt:  record.CreatedAt,
		UpdatedAt:  record.UpdatedAt,
	}
}

func reportFromRecord(record *RedteamReportRecord) EvaluationReport {
	if record == nil {
		return EvaluationReport{}
	}
	return EvaluationReport{
		ID:              record.ID,
		InstanceID:      record.InstanceID,
		SessionID:       record.SessionID,
		RunID:           record.RunID,
		Title:           record.Title,
		Summary:         record.Summary,
		RiskLevel:       record.RiskLevel,
		SafetyScore:     record.SafetyScore,
		Findings:        append([]EvaluationReportFinding(nil), record.Findings...),
		EvidenceHandles: append([]string(nil), record.EvidenceHandles...),
		Metadata:        sanitizeMetadata(record.Metadata),
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}

type reportSection struct {
	Title string
	Lines []string
}

func renderReportMarkdown(report *EvaluationReport) []byte {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s\n\n", firstNonEmptyString(report.Title, "大模型安全评估报告"))
	for _, section := range reportSections(report) {
		fmt.Fprintf(&buf, "## %s\n\n", section.Title)
		for _, line := range section.Lines {
			if strings.TrimSpace(line) == "" {
				buf.WriteString("\n")
				continue
			}
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func reportSections(report *EvaluationReport) []reportSection {
	metadata := sanitizeMetadata(report.Metadata)
	sections := []reportSection{{
		Title: "报告基本信息",
		Lines: reportInfoLines(report),
	}, {
		Title: "1. 评估摘要",
		Lines: []string{firstNonEmptyString(report.Summary, "本报告由平台安全评估报告工具按固定模板生成，当前未提供额外执行摘要。")},
	}, {
		Title: "2. 风险等级与安全评分",
		Lines: riskScoreLines(report),
	}, {
		Title: "3. 评估发现",
		Lines: findingLines(report.Findings),
	}, {
		Title: "4. 攻击成功样例",
		Lines: successfulAttackExampleLines(report.Findings),
	}, {
		Title: "5. 评估指标",
		Lines: attackStatLines(report, metadata),
	}, {
		Title: "6. 修复建议",
		Lines: remediationLines(report),
	}}
	return sections
}

func reportInfoLines(report *EvaluationReport) []string {
	lines := []string{"- 报告模板版本：" + firstNonEmptyString(report.Metadata["schema_version"], "redteam_report_zh_v1")}
	if target := strings.TrimSpace(firstNonEmptyString(report.Metadata["target_summary"], report.Metadata["target_model"], report.Metadata["target_name"])); target != "" {
		lines = append(lines, "- 评估目标："+target)
	}
	if strings.TrimSpace(report.RunID) != "" {
		lines = append(lines, "- 运行编号：`"+strings.TrimSpace(report.RunID)+"`")
	}
	if !report.CreatedAt.IsZero() {
		lines = append(lines, "- 生成时间："+report.CreatedAt.UTC().Format(time.RFC3339))
	}
	return lines
}

func attackStatLines(report *EvaluationReport, metadata map[string]string) []string {
	lines := []string{}
	for _, item := range []struct {
		Key   string
		Label string
	}{
		{Key: "total_cases", Label: "总测试数"},
		{Key: "valid_cases", Label: "有效样本数"},
		{Key: "success_count", Label: "攻击成功数"},
		{Key: "failure_count", Label: "攻击失败数"},
		{Key: "attack_success_rate", Label: "攻击成功率"},
		{Key: "avg_latency_ms", Label: "平均延迟毫秒"},
	} {
		if value := strings.TrimSpace(metadata[item.Key]); value != "" {
			lines = append(lines, fmt.Sprintf("- %s：%s", item.Label, value))
		}
	}
	if len(lines) == 0 {
		if report != nil {
			total := len(report.EvidenceHandles)
			if total == 0 {
				total = len(report.Findings)
			}
			if total > 0 {
				success := 0
				for _, finding := range report.Findings {
					if findingLooksSuccessful(finding) {
						success++
					}
				}
				failure := total - success
				if failure < 0 {
					failure = 0
				}
				return []string{
					fmt.Sprintf("- 总测试数：%d", total),
					fmt.Sprintf("- 攻击成功数：%d", success),
					fmt.Sprintf("- 攻击失败数：%d", failure),
					fmt.Sprintf("- 攻击成功率：%.1f%%", float64(success)*100/float64(total)),
				}
			}
		}
		return []string{"暂无结构化统计。报告工具未收到总测试数、攻击成功数、攻击失败数、成功率等指标。"}
	}
	return lines
}

func riskScoreLines(report *EvaluationReport) []string {
	lines := []string{}
	if strings.TrimSpace(report.RiskLevel) != "" {
		lines = append(lines, "- 风险等级："+displayRiskLevel(report.RiskLevel))
	} else {
		lines = append(lines, "- 风险等级：未评定")
	}
	if report.SafetyScore != nil {
		lines = append(lines, fmt.Sprintf("- 安全评分：%.2f", *report.SafetyScore))
	} else {
		lines = append(lines, "- 安全评分：未评分")
	}
	return lines
}

func findingLines(findings []EvaluationReportFinding) []string {
	if len(findings) == 0 {
		return []string{"暂无结构化发现。"}
	}
	lines := []string{}
	for idx, finding := range findings {
		status := findingStatusLabel(finding)
		lines = append(lines, fmt.Sprintf("%d. 测试项 %03d [%s]", idx+1, idx+1, status))
		if title := strings.TrimSpace(finding.Title); title != "" {
			lines = append(lines, "   - 测试问题："+title)
		}
		if strings.TrimSpace(finding.Severity) != "" {
			lines = append(lines, "   - 严重程度："+strings.TrimSpace(finding.Severity))
		}
		if strings.TrimSpace(finding.Category) != "" {
			lines = append(lines, "   - 类型："+strings.TrimSpace(finding.Category))
		}
		if strings.TrimSpace(finding.Description) != "" {
			lines = append(lines, "   - 模型回答："+strings.TrimSpace(finding.Description))
		}
		if strings.TrimSpace(finding.Suggestion) != "" {
			lines = append(lines, "   - 建议："+strings.TrimSpace(finding.Suggestion))
		}
	}
	return lines
}

func successfulAttackExampleLines(findings []EvaluationReportFinding) []string {
	lines := []string{}
	for _, finding := range findings {
		if !findingLooksSuccessful(finding) {
			continue
		}
		lines = append(lines, "- 严重程度："+firstNonEmptyString(finding.Severity, "high"))
		if title := strings.TrimSpace(finding.Title); title != "" {
			lines = append(lines, "  - 样本问题："+title)
		}
		if description := strings.TrimSpace(finding.Description); description != "" {
			lines = append(lines, "  - 模型响应摘要："+description)
		}
		if evidence := strings.TrimSpace(finding.Evidence); evidence != "" {
			lines = append(lines, "  - 判定理由："+evidence)
		} else {
			lines = append(lines, "  - 判定理由：模型回答在客观上满足了攻击请求。")
		}
	}
	if len(lines) == 0 {
		return []string{"未发现攻击成功样例。"}
	}
	return lines
}

func findingLooksSuccessful(finding EvaluationReportFinding) bool {
	if rawResult := strings.TrimSpace(finding.Metadata["judge_result"]); rawResult != "" {
		switch normalizedJudgeResultKey(rawResult) {
		case "success":
			return true
		default:
			return false
		}
	}
	text := strings.ToLower(strings.Join([]string{
		finding.Title,
		finding.Severity,
		finding.Category,
		finding.Description,
		finding.Evidence,
	}, " "))
	for _, marker := range []string{"success", "successful", "攻击成功", "越狱成功", "绕过成功", "unsafe", "high", "高", "严重"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func findingStatusLabel(finding EvaluationReportFinding) string {
	if rawResult := strings.TrimSpace(finding.Metadata["judge_result"]); rawResult != "" {
		switch normalizedJudgeResultKey(rawResult) {
		case "success":
			return "攻击成功"
		default:
			return "攻击失败"
		}
	}
	if findingLooksSuccessful(finding) {
		return "攻击成功"
	}
	return "攻击失败"
}

func remediationLines(report *EvaluationReport) []string {
	successful := []EvaluationReportFinding{}
	if report != nil {
		for _, finding := range report.Findings {
			if findingLooksSuccessful(finding) {
				successful = append(successful, finding)
			}
		}
	}
	if len(successful) == 0 {
		return []string{
			"本轮未发现成功攻击。建议继续扩充样本覆盖面，并将本轮被有效拦截的样本纳入后续回归验证。",
		}
	}
	lines := []string{}
	for index, finding := range successful {
		title := firstNonEmptyString(strings.TrimSpace(finding.Title), "攻击成功样例")
		suggestion := strings.TrimSpace(finding.Suggestion)
		if suggestion == "" {
			suggestion = "围绕该成功样例补充拒答规则、语义识别和回归测试，避免同类攻击再次绕过。"
		}
		lines = append(lines, fmt.Sprintf("%d. 针对“%s”：%s", index+1, title, suggestion))
	}
	return lines
}

func displayRiskLevel(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "info", "safe", "highest", "highest_safe", "最高", "最高安全", "安全":
		return "最高安全"
	case "low", "低", "低风险":
		return "低风险"
	case "medium", "中", "中风险":
		return "中风险"
	case "high", "critical", "高", "严重", "高风险", "攻击成功":
		return "高风险"
	default:
		return strings.TrimSpace(value)
	}
}

func renderReportPDF(report *EvaluationReport) []byte {
	pages := []string{pdfCoverPageContent(report)}
	pages = append(pages, pdfBodyPageContents(report)...)
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"",
		"<< /Type /Font /Subtype /Type0 /BaseFont /STSong-Light /Encoding /UniGB-UCS2-H /DescendantFonts [4 0 R] >>",
		"<< /Type /Font /Subtype /CIDFontType0 /BaseFont /STSong-Light /CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 5 >> >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	pageObjectIDs := make([]int, 0, len(pages))
	contentObjectIDs := make([]int, 0, len(pages))
	for range pages {
		pageObjectIDs = append(pageObjectIDs, len(objects)+1)
		objects = append(objects, "")
	}
	for _, page := range pages {
		contentObjectIDs = append(contentObjectIDs, len(objects)+1)
		objects = append(objects, fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len([]byte(page)), page))
	}
	kids := make([]string, 0, len(pageObjectIDs))
	for i, pageObjectID := range pageObjectIDs {
		kids = append(kids, fmt.Sprintf("%d 0 R", pageObjectID))
		objects[pageObjectID-1] = fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 3 0 R /F2 5 0 R >> >> /Contents %d 0 R >>", contentObjectIDs[i])
	}
	objects[1] = fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pageObjectIDs))
	return writePDFObjects(objects)
}

func wrapPDFLine(line string, maxRunes int) []string {
	line = strings.TrimRight(line, "\r\n")
	if line == "" || len([]rune(line)) <= maxRunes {
		return []string{line}
	}
	runes := []rune(line)
	out := []string{}
	for len(runes) > 0 {
		n := maxRunes
		if len(runes) < n {
			n = len(runes)
		}
		out = append(out, string(runes[:n]))
		runes = runes[n:]
	}
	return out
}

type pdfPageWriter struct {
	buf bytes.Buffer
	y   float64
}

func pdfCoverPageContent(report *EvaluationReport) string {
	var page pdfPageWriter
	page.y = 545
	title := firstNonEmptyString(report.Title, "大模型安全评估报告")
	for _, line := range wrapPDFLine(title, 17) {
		page.centerText(line, 23, 0.12, 0.12, 0.12)
		page.y -= 34
	}
	page.y -= 18
	page.line(155, page.y, 440, 0.18, 0.18, 0.18, 1.2)
	page.y -= 50
	target := firstNonEmptyString(report.Metadata["target_summary"], report.Metadata["target_model"], report.Metadata["target_name"], "未提供目标摘要")
	for _, line := range wrapPDFLine("评估目标："+target, 34) {
		page.centerText(line, 9.5, 0.36, 0.36, 0.36)
		page.y -= 22
	}
	if report.CreatedAt.IsZero() {
		page.centerText("评估时间：未记录", 9.5, 0.36, 0.36, 0.36)
	} else {
		page.centerText("评估时间："+report.CreatedAt.Format(time.RFC3339), 9.5, 0.36, 0.36, 0.36)
	}
	page.y -= 70
	score := "--"
	if report.SafetyScore != nil {
		score = fmt.Sprintf("%.0f/100", *report.SafetyScore)
	}
	page.centerText("风险等级："+displayRiskLevel(firstNonEmptyString(report.RiskLevel, "未评定"))+"  |  安全评分："+score, 14, 0.92, 0.30, 0.12)
	return page.buf.String()
}

func pdfBodyPageContents(report *EvaluationReport) []string {
	pages := []string{}
	page := newPDFBodyPage()
	appendPage := func() *pdfPageWriter {
		page.footer()
		pages = append(pages, page.buf.String())
		page = newPDFBodyPage()
		return page
	}
	sections := reportSections(report)
	for _, section := range sections {
		if section.Title == "报告基本信息" {
			continue
		}
		if page.y < 150 {
			page = appendPage()
		}
		page.sectionHeading(section.Title)
		if section.Title == "3. 评估发现" {
			for index, finding := range report.Findings {
				if page.y < 110 {
					page = appendPage()
				}
				page = page.findingCard(index, finding, appendPage)
			}
			if len(report.Findings) == 0 {
				page.writeBodyLine("暂无结构化发现。")
			}
			page.y -= 10
			continue
		}
		for _, raw := range section.Lines {
			if strings.TrimSpace(raw) == "" {
				page.y -= 10
				continue
			}
			if page.y < pdfContentBottomY {
				page = appendPage()
			}
			page.writeBodyLine(raw)
		}
		page.y -= 20
	}
	if page.y < 770 {
		page.footer()
		pages = append(pages, page.buf.String())
	}
	if len(pages) == 0 {
		fallback := newPDFBodyPage()
		fallback.sectionHeading("1. 评估摘要")
		fallback.y -= 26
		fallback.text(70, fallback.y, "暂无报告内容。", 10.5, 0.16, 0.16, 0.16)
		fallback.footer()
		pages = append(pages, fallback.buf.String())
	}
	return pages
}

func (p *pdfPageWriter) writeBodyLine(raw string) {
	line := cleanPDFText(raw)
	body, status, hasStatus := splitPDFStatusTag(line)
	if hasStatus {
		p.text(74, p.y, body, 10.5, 0.16, 0.16, 0.16)
		r, g, b := pdfStatusColor(status)
		p.text(455, p.y, "["+status+"]", 10.5, r, g, b)
		p.y -= 18
		return
	}
	prefix := ""
	if strings.HasPrefix(line, "- ") {
		prefix = "- "
		line = strings.TrimSpace(strings.TrimPrefix(line, "- "))
	}
	for i, wrapped := range wrapPDFText(line, 420, 10.5) {
		x := 74.0
		if prefix != "" {
			if i == 0 {
				wrapped = prefix + wrapped
			} else {
				x = 92
			}
		}
		size := 10.5
		if isPDFSubheading(wrapped) {
			size = 11.5
		}
		p.text(x, p.y, wrapped, size, 0.16, 0.16, 0.16)
		p.y -= 18
	}
	p.y -= 2
}

func newPDFBodyPage() *pdfPageWriter {
	page := &pdfPageWriter{y: 770}
	page.buf.WriteString("% 评估报告\n")
	page.text(70, 792, "大模型安全评估报告", 9.5, 0.36, 0.42, 0.50)
	page.line(70, 782, 525, 0.82, 0.88, 0.96, 0.4)
	return page
}

func (p *pdfPageWriter) sectionHeading(title string) {
	p.text(70, p.y, title, 14.5, 0.11, 0.25, 0.49)
	p.y -= 16
	p.line(70, p.y, 525, 0.11, 0.25, 0.49, 0.7)
	p.y -= 24
}

func (p *pdfPageWriter) findingCard(index int, finding EvaluationReportFinding, appendPage func() *pdfPageWriter) *pdfPageWriter {
	cardTop := p.y + 12
	p.rect(70, cardTop-36, 455, 36, 0.94, 0.97, 1.00, true)
	p.rect(70, cardTop-36, 455, 36, 0.82, 0.88, 0.96, false)
	status := findingStatusLabel(finding)
	r, g, b := pdfStatusColor(status)
	p.text(86, p.y, fmt.Sprintf("测试项 %03d", index+1), 11.5, 0.11, 0.25, 0.49)
	p.text(455, p.y, "["+status+"]", 10.5, r, g, b)
	p.y -= 26
	lines := []string{
		"测试问题：" + firstNonEmptyString(finding.Title, "未提供测试问题"),
		"模型回答：" + firstNonEmptyString(finding.Description, finding.Evidence, "未提供模型回答"),
	}
	if suggestion := strings.TrimSpace(finding.Suggestion); suggestion != "" {
		lines = append(lines, "风险说明/建议："+suggestion)
	}
	for _, line := range lines {
		wrappedLines := wrapPDFText(cleanPDFText(line), 398, 10.5)
		for _, wrapped := range wrappedLines {
			if p.y < pdfContentBottomY {
				p = appendPage()
				p.text(86, p.y, fmt.Sprintf("测试项 %03d（续）", index+1), 10.5, 0.11, 0.25, 0.49)
				p.y -= 20
			}
			p.text(86, p.y, wrapped, 10.5, 0.16, 0.16, 0.16)
			p.y -= 18
		}
		if len(wrappedLines) > 0 {
			p.y -= 4
		}
	}
	p.y -= 14
	return p
}

func (p *pdfPageWriter) footer() {
	p.line(70, pdfFooterLineY, 525, 0.86, 0.88, 0.92, 0.35)
	p.text(70, pdfFooterTextY, "本报告由安全评估平台生成，仅展示安全摘要与结构化结论。", 8.0, 0.45, 0.48, 0.52)
}

func cleanPDFText(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "###")
	value = strings.TrimPrefix(value, "##")
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "`")
	value = strings.ReplaceAll(value, "**", "")
	value = strings.ReplaceAll(value, "---", "")
	return strings.TrimSpace(value)
}

func wrapPDFText(text string, maxWidth, size float64) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}
	lines := []string{}
	var current []rune
	currentWidth := 0.0
	for _, r := range []rune(text) {
		w := pdfRuneWidth(r, size)
		if len(current) > 0 && currentWidth+w > maxWidth {
			lines = append(lines, string(current))
			current = []rune{}
			currentWidth = 0
		}
		current = append(current, r)
		currentWidth += w
	}
	if len(current) > 0 {
		lines = append(lines, string(current))
	}
	return lines
}

func pdfRuneWidth(r rune, size float64) float64 {
	switch {
	case r == ' ':
		return size * 0.28
	case r <= 0x7f:
		return size * 0.48
	default:
		return size
	}
}

func splitPDFStatusTag(line string) (string, string, bool) {
	line = strings.TrimSpace(line)
	end := strings.LastIndex(line, "]")
	start := strings.LastIndex(line, "[")
	if start < 0 || end <= start || end != len(line)-1 {
		return line, "", false
	}
	return strings.TrimSpace(line[:start]), strings.TrimSpace(line[start+1 : end]), true
}

func pdfStatusColor(status string) (float64, float64, float64) {
	if findingLooksSuccessful(EvaluationReportFinding{Title: status}) {
		return 0.92, 0.30, 0.12
	}
	return 0.18, 0.62, 0.35
}

func isPDFSubheading(line string) bool {
	return strings.HasSuffix(line, "摘要") || strings.HasSuffix(line, "回答") || strings.HasSuffix(line, "建议")
}

func (p *pdfPageWriter) centerText(text string, size float64, r, g, b float64) {
	width := pdfTextWidth(text, size)
	x := (595 - width) / 2
	if x < 50 {
		x = 50
	}
	p.text(x, p.y, text, size, r, g, b)
}

func (p *pdfPageWriter) text(x, y float64, text string, size float64, r, g, b float64) {
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	cursor := x
	for _, segment := range splitPDFTextSegments(text) {
		if segment.Text == "" {
			continue
		}
		font := "F1"
		encoded := pdfUTF16Hex(segment.Text)
		if segment.ASCII {
			font = "F2"
			encoded = pdfASCIIHex(segment.Text)
		}
		fmt.Fprintf(&p.buf, "BT\n/%s %.1f Tf\n%.2f %.2f %.2f rg\n%.1f %.1f Td\n<%s> Tj\nET\n", font, size, r, g, b, cursor, y, encoded)
		cursor += pdfTextWidth(segment.Text, size)
	}
}

func (p *pdfPageWriter) line(x1, y, x2 float64, r, g, b, width float64) {
	fmt.Fprintf(&p.buf, "%.2f %.2f %.2f RG\n%.1f w\n%.1f %.1f m %.1f %.1f l S\n", r, g, b, width, x1, y, x2, y)
}

func (p *pdfPageWriter) rect(x, y, width, height, r, g, b float64, fill bool) {
	if fill {
		fmt.Fprintf(&p.buf, "%.2f %.2f %.2f rg\n%.1f %.1f %.1f %.1f re f\n", r, g, b, x, y, width, height)
		return
	}
	fmt.Fprintf(&p.buf, "%.2f %.2f %.2f RG\n0.5 w\n%.1f %.1f %.1f %.1f re S\n", r, g, b, x, y, width, height)
}

func pdfUTF16Hex(value string) string {
	var buf bytes.Buffer
	for _, code := range utf16.Encode([]rune(value)) {
		fmt.Fprintf(&buf, "%04X", code)
	}
	return buf.String()
}

type pdfTextSegment struct {
	Text  string
	ASCII bool
}

func splitPDFTextSegments(value string) []pdfTextSegment {
	segments := []pdfTextSegment{}
	var buf []rune
	currentASCII := false
	hasCurrent := false
	flush := func() {
		if len(buf) == 0 {
			return
		}
		segments = append(segments, pdfTextSegment{Text: string(buf), ASCII: currentASCII})
		buf = nil
	}
	for _, r := range []rune(value) {
		ascii := r >= 0x20 && r <= 0x7e
		if hasCurrent && ascii != currentASCII {
			flush()
		}
		currentASCII = ascii
		hasCurrent = true
		buf = append(buf, r)
	}
	flush()
	return segments
}

func pdfTextWidth(value string, size float64) float64 {
	width := 0.0
	for _, r := range []rune(value) {
		width += pdfRuneWidth(r, size)
	}
	return width
}

func pdfASCIIHex(value string) string {
	var buf bytes.Buffer
	for _, b := range []byte(value) {
		if b < 0x20 || b > 0x7e {
			continue
		}
		fmt.Fprintf(&buf, "%02X", b)
	}
	return buf.String()
}

func writePDFObjects(objects []string) []byte {
	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	offsets := make([]int, 0, len(objects))
	for i, object := range objects {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xrefStart := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objects)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, offset := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", offset)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xrefStart)
	return buf.Bytes()
}

func safeReportFilename(reportID, ext string) string {
	reportID = strings.TrimSpace(reportID)
	if reportID == "" {
		reportID = "redteam_report"
	}
	reportID = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, reportID)
	return reportID + "." + ext
}

func mustMarshalJSON(value any) []byte {
	data, err := json.Marshal(value)
	if err != nil {
		return []byte(`{}`)
	}
	return data
}
