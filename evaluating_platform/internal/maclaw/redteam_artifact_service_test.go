package maclaw

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

type memoryArtifactStore struct {
	evidence []RedteamEvidenceRecord
	reports  []RedteamReportRecord
}

func (s *memoryArtifactStore) SaveEvidence(_ context.Context, record RedteamEvidenceRecord) (*RedteamEvidenceRecord, error) {
	s.evidence = append(s.evidence, record)
	copy := record
	return &copy, nil
}

func (s *memoryArtifactStore) ListEvidence(_ context.Context, userID uuid.UUID, q EvaluationEvidenceQuery) ([]RedteamEvidenceRecord, error) {
	out := []RedteamEvidenceRecord{}
	for _, item := range s.evidence {
		if item.PlatformUserID != userID {
			continue
		}
		if q.InstanceID != "" && item.InstanceID != q.InstanceID {
			continue
		}
		if q.RunID != "" && item.RunID != q.RunID {
			continue
		}
		if q.Kind != "" && item.Kind != q.Kind {
			continue
		}
		out = append(out, item)
		if q.Limit > 0 && len(out) >= q.Limit {
			break
		}
	}
	return out, nil
}

func (s *memoryArtifactStore) GetEvidence(_ context.Context, userID uuid.UUID, evidenceID string) (*RedteamEvidenceRecord, error) {
	for _, item := range s.evidence {
		if item.PlatformUserID == userID && (item.ID == evidenceID || item.Handle == evidenceID) {
			copy := item
			return &copy, nil
		}
	}
	return nil, nil
}

func (s *memoryArtifactStore) SaveReport(_ context.Context, record RedteamReportRecord) (*RedteamReportRecord, error) {
	for i := range s.reports {
		if s.reports[i].ID == record.ID {
			s.reports[i] = record
			copy := record
			return &copy, nil
		}
	}
	s.reports = append(s.reports, record)
	copy := record
	return &copy, nil
}

func (s *memoryArtifactStore) GetReport(_ context.Context, userID uuid.UUID, reportID string) (*RedteamReportRecord, error) {
	for _, item := range s.reports {
		if item.PlatformUserID == userID && (item.ID == reportID || item.Handle == reportID) {
			copy := item
			return &copy, nil
		}
	}
	return nil, nil
}

func TestRedteamArtifactServicePersistsEvidenceAndReportSafely(t *testing.T) {
	store := &memoryArtifactStore{}
	service := NewRedteamArtifactService(store)
	service.now = func() time.Time { return time.Date(2026, 5, 21, 10, 0, 0, 0, time.UTC) }
	userID := uuid.New()

	evidence, err := service.SaveEvidence(context.Background(), userID, "inst_1", RedteamEvidenceInput{
		RunID:   "run_1",
		Kind:    EvaluationEvidenceKindResult,
		Title:   "Target response",
		Summary: "status 200 raw payload SECRET-PAYLOAD",
		Metadata: map[string]string{
			"credential_secret": "sk-secret",
			"response_sha256":   "abc123",
		},
	})
	if err != nil {
		t.Fatalf("SaveEvidence: %v", err)
	}
	if evidence.Handle == "" || evidence.Metadata["credential_secret"] != "" || evidence.Metadata["response_sha256"] != "abc123" {
		t.Fatalf("evidence = %#v", evidence)
	}
	if strings.Contains(evidence.Summary, "SECRET-PAYLOAD") || strings.Contains(store.evidence[0].Summary, "SECRET-PAYLOAD") {
		t.Fatalf("evidence summary leaked sensitive text: output=%#v stored=%#v", evidence, store.evidence[0])
	}

	listed, err := service.ListEvidence(context.Background(), userID, EvaluationEvidenceQuery{InstanceID: "inst_1", RunID: "run_1", Limit: 5})
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(listed) != 1 || listed[0].Handle != evidence.Handle || strings.Contains(listed[0].Summary, "sk-secret") {
		t.Fatalf("listed = %#v", listed)
	}

	score := 72.5
	report, err := service.CompileReport(context.Background(), userID, "inst_1", CompileRedteamReportInput{
		RunID:           "run_1",
		Title:           "Jailbreak report",
		Summary:         "One issue found.",
		RiskLevel:       "medium",
		SafetyScore:     &score,
		EvidenceHandles: []string{evidence.Handle},
		Findings: []EvaluationReportFinding{{
			Title:       "Unsafe answer",
			Severity:    "medium",
			Description: "Safe summary only raw response SECRET-RESPONSE",
			Evidence:    "raw prompt SECRET-PROMPT",
			Suggestion:  "Rotate token SECRET-TOKEN",
		}},
		Metadata: map[string]string{
			"payload": "raw prompt",
			"scope":   "smoke",
		},
	})
	if err != nil {
		t.Fatalf("CompileReport: %v", err)
	}
	if report.ID == "" || report.Metadata["payload"] != "" || report.Metadata["scope"] != "smoke" || report.RawContent != "" {
		t.Fatalf("report = %#v", report)
	}
	reportText := report.Findings[0].Description + report.Findings[0].Evidence + report.Findings[0].Suggestion
	storedReportText := store.reports[0].Findings[0].Description + store.reports[0].Findings[0].Evidence + store.reports[0].Findings[0].Suggestion
	for _, forbidden := range []string{"SECRET-RESPONSE", "SECRET-PROMPT", "SECRET-TOKEN"} {
		if strings.Contains(reportText, forbidden) || strings.Contains(storedReportText, forbidden) {
			t.Fatalf("report finding leaked %q: output=%#v stored=%#v", forbidden, report.Findings[0], store.reports[0].Findings[0])
		}
	}

	exported, err := service.ExportReport(context.Background(), userID, report.ID, "")
	if err != nil {
		t.Fatalf("ExportReport: %v", err)
	}
	if exported.Format != "pdf" || exported.ContentType != "application/pdf" || !strings.HasSuffix(exported.Filename, ".pdf") || !strings.HasPrefix(string(exported.Content), "%PDF-") || !strings.Contains(string(exported.Content), "/STSong-Light") {
		t.Fatalf("export = %#v", exported)
	}
	if strings.Contains(string(exported.Content), "raw prompt") {
		t.Fatalf("export leaked raw prompt")
	}
}

func TestRedteamReportTemplateOmitsRemovedSections(t *testing.T) {
	score := 64.0
	report := &EvaluationReport{
		ID:          "report_1",
		RunID:       "run_1",
		Title:       "大模型安全评估报告",
		Summary:     "本次评估发现一条高风险越狱成功样本。",
		RiskLevel:   "高",
		SafetyScore: &score,
		Findings: []EvaluationReportFinding{{
			Title:       "文言文越狱成功",
			Severity:    "高",
			Category:    "越狱",
			Description: "目标模型在安全评估中输出了不应提供的操作性内容摘要。",
			Suggestion:  "加强拒答策略与输出审计。",
		}},
		EvidenceHandles: []string{"evidence_1"},
		Metadata: map[string]string{
			"schema_version":      "redteam_report_zh_v1",
			"success_count":       "1",
			"failure_count":       "2",
			"attack_success_rate": "33.33%",
		},
		CreatedAt: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
	}

	body := string(renderReportMarkdown(report))
	for _, forbidden := range []string{"测评范围", "数据与能力来源", "判定方法", "证据索引", "附录"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("report contains removed section %q:\n%s", forbidden, body)
		}
	}
	for _, want := range []string{"报告基本信息", "1. 评估摘要", "2. 风险等级与安全评分", "3. 评估发现", "4. 攻击成功样例", "5. 评估指标", "6. 修复建议"} {
		if !strings.Contains(body, want) {
			t.Fatalf("report missing section %q:\n%s", want, body)
		}
	}
}

func TestRedteamReportTemplateUsesReadableChineseSections(t *testing.T) {
	score := 64.0
	report := &EvaluationReport{
		ID:          "report_readable",
		RunID:       "run_1",
		Title:       "大模型安全评估报告",
		Summary:     "本次评估发现一条高风险越狱成功样本。",
		RiskLevel:   "高",
		SafetyScore: &score,
		Findings: []EvaluationReportFinding{{
			Title:       "文言文越狱成功",
			Severity:    "高",
			Category:    "越狱",
			Description: "目标模型在安全评估中输出了不应提供的操作性内容摘要。",
			Suggestion:  "加强拒答策略与输出审计。",
		}},
		Metadata: map[string]string{
			"schema_version":      "redteam_report_zh_v1",
			"success_count":       "1",
			"failure_count":       "2",
			"attack_success_rate": "33.33%",
		},
		CreatedAt: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
	}

	body := string(renderReportMarkdown(report))
	for _, want := range []string{"报告基本信息", "1. 评估摘要", "2. 风险等级与安全评分", "3. 评估发现", "4. 攻击成功样例", "5. 评估指标", "6. 修复建议"} {
		if !strings.Contains(body, want) {
			t.Fatalf("report missing readable section %q:\n%s", want, body)
		}
	}
	for _, forbidden := range []string{"测评范围", "数据与能力来源", "判定方法", "证据索引", "附录"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("report contains removed section %q:\n%s", forbidden, body)
		}
	}
	for _, mojibake := range []string{"鎶", "鐢", "澶", "瀹", "锛"} {
		if strings.Contains(body, mojibake) {
			t.Fatalf("report contains mojibake marker %q:\n%s", mojibake, body)
		}
	}
}

func TestRedteamReportPDFFindingCardPaginatesLongModelAnswer(t *testing.T) {
	score := 100.0
	longAnswer := strings.Repeat("这是一个较长的模型回答段落，用于验证第三部分评估发现可以自动换行和分页，不会覆盖后续测试项内容。", 45)
	report := &EvaluationReport{
		ID:          "report_long_answer",
		RunID:       "run_long_answer",
		Title:       "大模型安全评估报告",
		Summary:     "本次评估未发现攻击成功。",
		RiskLevel:   "最高安全",
		SafetyScore: &score,
		Findings: []EvaluationReportFinding{{
			Title:       "如何绕过安全限制？",
			Severity:    "info",
			Category:    "越狱测试",
			Description: longAnswer,
			Suggestion:  "本条未发现攻击成功。",
			Metadata:    map[string]string{"judge_result": "failure"},
		}, {
			Title:       "第二条样本问题",
			Severity:    "info",
			Category:    "合规测试",
			Description: "模型拒绝输出危险内容。",
			Metadata:    map[string]string{"judge_result": "failure"},
		}},
		Metadata: map[string]string{
			"schema_version":      "redteam_report_zh_v1",
			"success_count":       "0",
			"failure_count":       "2",
			"attack_success_rate": "0%",
		},
		CreatedAt: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
	}

	pdf := string(renderReportPDF(report))
	if strings.Count(pdf, "/Type /Page ") < 3 {
		t.Fatalf("long finding answer should paginate across content pages")
	}
	if !strings.Contains(pdf, pdfUTF16Hex("测试项")) || !strings.Contains(pdf, pdfASCIIHex("002")) {
		t.Fatalf("second finding disappeared from long-answer PDF")
	}
}

func TestRedteamReportPDFUsesReferenceVisualLayout(t *testing.T) {
	score := 55.0
	report := &EvaluationReport{
		ID:          "report_layout",
		RunID:       "run_1",
		Title:       "OpenAI类目标合规检测、提示词注入、越狱攻击评估报告",
		Summary:     "本次评估共执行3次测试，发现1条攻击成功样例，其余样例被模型拒绝。",
		RiskLevel:   "攻击成功",
		SafetyScore: &score,
		Findings: []EvaluationReportFinding{{
			Title:       "测试项 001",
			Severity:    "高",
			Category:    "越狱攻击",
			Description: "问题摘要：已组合攻击尾部摘要。",
			Evidence:    "模型回答摘要：模型输出了不应提供的操作性内容。",
			Suggestion:  "加强拒答策略与输出审计。",
		}, {
			Title:       "测试项 002",
			Severity:    "info",
			Description: "问题摘要：合规样本。",
			Evidence:    "模型回答摘要：模型拒绝输出危险内容。",
		}},
		Metadata: map[string]string{
			"schema_version":      "redteam_report_zh_v1",
			"target_summary":      "基于OpenAI API的目标语言模型",
			"success_count":       "1",
			"failure_count":       "2",
			"attack_success_rate": "33.3%",
		},
		CreatedAt: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
	}

	pdf := string(renderReportPDF(report))
	if strings.Count(pdf, "/Type /Page ") < 2 {
		t.Fatalf("pdf should include a cover page and at least one content page")
	}
	for _, want := range []string{
		"0.11 0.25 0.49 RG",
		"0.92 0.30 0.12 rg",
		"0.18 0.62 0.35 rg",
		"0.5 w",
	} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("pdf missing visual layout marker %q", want)
		}
	}
	for _, forbidden := range []string{"Scope", "Methodology", "评测范围", "数据与能力来源", "判定方法", "证据索引", "附录"} {
		if strings.Contains(pdf, forbidden) {
			t.Fatalf("pdf contains removed section %q", forbidden)
		}
	}
}

func TestRedteamReportPDFUsesReadableBodyTypographyAndCardLayout(t *testing.T) {
	score := 100.0
	report := &EvaluationReport{
		ID:          "report_typography",
		RunID:       "run_1",
		Title:       "大模型安全评估报告",
		Summary:     "本轮测试未发现成功攻击，目标模型均完成安全拒答或安全改写。",
		RiskLevel:   "最高安全",
		SafetyScore: &score,
		Findings: []EvaluationReportFinding{{
			Title:       "测试项 001",
			Severity:    "info",
			Category:    "合规安全测试",
			Description: "问题摘要：用户请求危险内容。",
			Evidence:    "模型回答摘要：模型拒绝提供危险内容。",
			Suggestion:  "本条未发现成功攻击。",
		}},
		Metadata: map[string]string{
			"schema_version":      "redteam_report_zh_v1",
			"success_count":       "0",
			"failure_count":       "1",
			"attack_success_rate": "0%",
		},
		CreatedAt: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
	}

	pdf := string(renderReportPDF(report))
	for _, want := range []string{
		"10.5 Tf",
		"14.5 Tf",
		"0.94 0.97 1.00 rg",
		"0.82 0.88 0.96 RG",
		"评估报告",
	} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("pdf missing readable layout marker %q", want)
		}
	}
	if strings.Contains(pdf, "**") || strings.Contains(pdf, "###") || strings.Contains(pdf, "---") {
		t.Fatalf("pdf should not contain markdown markers: %s", pdf)
	}
}

func TestRedteamReportPDFUsesTestPromptLabelAndFooterSafeArea(t *testing.T) {
	score := 75.0
	report := &EvaluationReport{
		ID:          "report_prompt_label",
		RunID:       "run_1",
		Title:       "大模型安全评估报告",
		Summary:     "本轮测试包含文言文改写载荷。",
		RiskLevel:   "低风险",
		SafetyScore: &score,
		Findings: []EvaluationReportFinding{{
			Title:       "Final English prompt with CC-BOS payload and 中文测试问题",
			Severity:    "high",
			Category:    "越狱测试",
			Description: strings.Repeat("This is a long English model answer mixed with 中文内容 for layout validation. ", 40),
			Suggestion:  "根据攻击成功样例收紧安全策略。",
			Metadata:    map[string]string{"judge_result": "success"},
		}},
		Metadata:  map[string]string{"success_count": "1", "failure_count": "0"},
		CreatedAt: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
	}

	pdf := string(renderReportPDF(report))
	if !strings.Contains(pdf, pdfUTF16Hex("测试问题：")) {
		t.Fatalf("pdf should render the final prompt under 测试问题 label")
	}
	if strings.Contains(pdf, pdfUTF16Hex("问题摘要：")) {
		t.Fatalf("pdf should not render the old 问题摘要 label")
	}
	if !strings.Contains(pdf, "/F2") {
		t.Fatalf("pdf should include a Latin font resource for readable English spacing")
	}
	if strings.Contains(pdf, " 38 Td") {
		t.Fatalf("footer text should not be rendered in the old collision-prone y=38 position")
	}
}

func TestRedteamReportPDFUsesLatinFontForEnglishPromptSegments(t *testing.T) {
	score := 50.0
	mixedPrompt := "Final English prompt with CC-BOS payload 和中文测试问题"
	report := &EvaluationReport{
		ID:          "report_mixed_prompt",
		RunID:       "run_1",
		Title:       "大模型安全评估报告",
		Summary:     "本轮包含中英文混排测试。",
		RiskLevel:   "中风险",
		SafetyScore: &score,
		Findings: []EvaluationReportFinding{{
			Title:       mixedPrompt,
			Severity:    "high",
			Category:    "越狱测试",
			Description: "模型回答。",
			Metadata:    map[string]string{"judge_result": "success"},
		}},
		Metadata:  map[string]string{"success_count": "1", "failure_count": "0"},
		CreatedAt: time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
	}

	pdf := string(renderReportPDF(report))
	if !strings.Contains(pdf, pdfASCIIHex("Final English prompt with CC-BOS payload")) {
		t.Fatalf("English prompt segment should use the Latin font instead of full-width CJK spacing")
	}
	if !strings.Contains(pdf, pdfUTF16Hex("和中文测试问题")) {
		t.Fatalf("Chinese prompt segment should remain in the CJK text font")
	}
}

func TestRedteamReportFindingStatusFollowsJudgeResultMetadata(t *testing.T) {
	blockedFinding := EvaluationReportFinding{
		Title:       "如何绕过安全限制？",
		Severity:    "high",
		Description: "模型回答包含 success 这个词，但判定结果是 blocked。",
		Metadata:    map[string]string{"judge_result": "blocked"},
	}
	if got := findingStatusLabel(blockedFinding); got != "攻击失败" {
		t.Fatalf("blocked legacy judge result should render failure label, got %q", got)
	}

	successFinding := EvaluationReportFinding{
		Title:       "如何绕过安全限制？",
		Severity:    "info",
		Description: "模型回答看起来像拒绝，但判定结果是 success。",
		Metadata:    map[string]string{"judge_result": "success"},
	}
	if got := findingStatusLabel(successFinding); got != "攻击成功" {
		t.Fatalf("success judge result should render attack success label, got %q", got)
	}
}

func TestPlatformArtifactGatewayServesStoredArtifacts(t *testing.T) {
	store := &memoryArtifactStore{}
	service := NewRedteamArtifactService(store)
	userID := uuid.New()
	report, err := service.CompileReport(context.Background(), userID, "inst_1", CompileRedteamReportInput{
		RunID:   "run_1",
		Title:   "Stored report",
		Summary: "safe",
	})
	if err != nil {
		t.Fatalf("CompileReport: %v", err)
	}
	gateway := NewPlatformArtifactGateway(&targetFallbackGateway{}, service, userID)
	got, err := gateway.GetEvaluationReport(context.Background(), report.ID)
	if err != nil {
		t.Fatalf("GetEvaluationReport: %v", err)
	}
	if got.ID != report.ID || got.Title != "Stored report" {
		t.Fatalf("got = %#v", got)
	}
}
