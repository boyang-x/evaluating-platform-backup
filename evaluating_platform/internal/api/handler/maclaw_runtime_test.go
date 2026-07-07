package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evaluating_platform/internal/maclaw"
	"evaluating_platform/internal/model"
)

type fakeMaclawRuntimeGateway struct {
	enabled      bool
	lastInstID   string
	lastSession  string
	lastRun      string
	lastJob      string
	lastMessage  maclaw.RuntimeMessageInput
	lastConfirm  maclaw.RuntimeMessageInput
	confirmCalls int
	messages     []maclaw.RuntimeMessage
	postMessage  *maclaw.RuntimeMessageResponse
	postMessages []*maclaw.RuntimeMessageResponse
	postCalls    []maclaw.RuntimeMessageInput
	jobs         []maclaw.EvaluationJob
	lastJobQuery maclaw.EvaluationJobQuery
	reports      map[string]*maclaw.EvaluationReport
}

func (f *fakeMaclawRuntimeGateway) Enabled() bool { return f.enabled }

func (f *fakeMaclawRuntimeGateway) ListRuntimeSessions(ctx context.Context, instanceID string, q maclaw.RuntimeSessionQuery) ([]maclaw.RuntimeSession, error) {
	_ = ctx
	_ = q
	f.lastInstID = instanceID
	return []maclaw.RuntimeSession{{ID: "sess_1", InstanceID: instanceID, Title: "Enterprise assessment"}}, nil
}

func (f *fakeMaclawRuntimeGateway) CreateRuntimeSession(ctx context.Context, instanceID string, in maclaw.RuntimeSessionInput) (*maclaw.RuntimeSession, error) {
	_ = ctx
	f.lastInstID = instanceID
	return &maclaw.RuntimeSession{ID: "sess_1", InstanceID: instanceID, Title: in.Title}, nil
}

func (f *fakeMaclawRuntimeGateway) GetRuntimeSession(ctx context.Context, instanceID, sessionID string) (*maclaw.RuntimeSession, error) {
	_ = ctx
	f.lastInstID = instanceID
	f.lastSession = sessionID
	return &maclaw.RuntimeSession{ID: sessionID, InstanceID: instanceID, Title: "Enterprise assessment"}, nil
}

func (f *fakeMaclawRuntimeGateway) DeleteRuntimeSession(ctx context.Context, instanceID, sessionID string) error {
	_ = ctx
	f.lastInstID = instanceID
	f.lastSession = sessionID
	return nil
}

func (f *fakeMaclawRuntimeGateway) ListRuntimeMessages(ctx context.Context, instanceID, sessionID string, q maclaw.RuntimeMessageQuery) ([]maclaw.RuntimeMessage, error) {
	_ = ctx
	_ = q
	f.lastInstID = instanceID
	f.lastSession = sessionID
	if f.messages != nil {
		return f.messages, nil
	}
	return []maclaw.RuntimeMessage{{ID: "msg_1", SessionID: sessionID, Role: "assistant", Content: "plan"}}, nil
}

func (f *fakeMaclawRuntimeGateway) PostRuntimeMessage(ctx context.Context, instanceID, sessionID string, in maclaw.RuntimeMessageInput) (*maclaw.RuntimeMessageResponse, error) {
	_ = ctx
	f.lastInstID = instanceID
	f.lastSession = sessionID
	f.lastMessage = in
	f.postCalls = append(f.postCalls, in)
	if len(f.postMessages) > 0 {
		out := f.postMessages[0]
		f.postMessages = f.postMessages[1:]
		return out, nil
	}
	if f.postMessage != nil {
		return f.postMessage, nil
	}
	return &maclaw.RuntimeMessageResponse{
		Run:     &maclaw.RuntimeRun{ID: "run_1", SessionID: sessionID, Status: "running"},
		Message: &maclaw.RuntimeMessage{ID: "msg_1", Role: "assistant", Content: in.Content},
	}, nil
}

func (f *fakeMaclawRuntimeGateway) ConfirmRuntimePlan(ctx context.Context, instanceID, sessionID string, in maclaw.RuntimeMessageInput) (*maclaw.EvaluationJob, error) {
	_ = ctx
	f.lastInstID = instanceID
	f.lastSession = sessionID
	f.lastConfirm = in
	f.confirmCalls++
	return &maclaw.EvaluationJob{ID: "job_1", Kind: maclaw.EvaluationJobKindRun, Status: maclaw.EvaluationJobStatusPending}, nil
}

func (f *fakeMaclawRuntimeGateway) CancelRuntimeRun(ctx context.Context, instanceID, runID string) (*maclaw.RuntimeRun, error) {
	_ = ctx
	f.lastInstID = instanceID
	f.lastRun = runID
	return &maclaw.RuntimeRun{ID: runID, Status: "cancelled"}, nil
}

func (f *fakeMaclawRuntimeGateway) CancelEvaluationRun(ctx context.Context, runID string) (*maclaw.RuntimeRun, error) {
	_ = ctx
	f.lastRun = runID
	return &maclaw.RuntimeRun{ID: runID, Status: "cancelled"}, nil
}

func (f *fakeMaclawRuntimeGateway) StreamRuntimeRunEvents(ctx context.Context, instanceID, runID string) (*maclaw.RuntimeEventStream, error) {
	_ = ctx
	f.lastInstID = instanceID
	f.lastRun = runID
	return &maclaw.RuntimeEventStream{
		Body:        io.NopCloser(strings.NewReader("event: report\ndata: {\"type\":\"report\"}\n\n")),
		ContentType: "text/event-stream",
	}, nil
}

func (f *fakeMaclawRuntimeGateway) StreamEvaluationRunEvents(ctx context.Context, runID string) (*maclaw.RuntimeEventStream, error) {
	_ = ctx
	f.lastRun = runID
	return &maclaw.RuntimeEventStream{
		Body:        io.NopCloser(strings.NewReader("event: report\ndata: {\"type\":\"report\"}\n\n")),
		ContentType: "text/event-stream",
	}, nil
}

func (f *fakeMaclawRuntimeGateway) ListEvaluationJobs(ctx context.Context, q maclaw.EvaluationJobQuery) ([]maclaw.EvaluationJob, error) {
	_ = ctx
	f.lastJobQuery = q
	return f.jobs, nil
}

func (f *fakeMaclawRuntimeGateway) GetEvaluationJob(ctx context.Context, jobID string) (*maclaw.EvaluationJob, error) {
	_ = ctx
	f.lastJob = jobID
	return &maclaw.EvaluationJob{ID: jobID, Kind: maclaw.EvaluationJobKindRun, Status: maclaw.EvaluationJobStatusSucceeded, Result: &maclaw.EvaluationRunResult{Run: &maclaw.RuntimeRun{ID: "run_1", Status: "succeeded"}}}, nil
}

func (f *fakeMaclawRuntimeGateway) GetEvaluationReport(ctx context.Context, reportID string) (*maclaw.EvaluationReport, error) {
	_ = ctx
	if f.reports != nil {
		return f.reports[reportID], nil
	}
	return nil, nil
}

func (f *fakeMaclawRuntimeGateway) CancelEvaluationJob(ctx context.Context, jobID string) (*maclaw.EvaluationJob, error) {
	_ = ctx
	f.lastJob = jobID
	return &maclaw.EvaluationJob{ID: jobID, Kind: maclaw.EvaluationJobKindRun, Status: maclaw.EvaluationJobStatusCanceled}, nil
}

func (f *fakeMaclawRuntimeGateway) RetryEvaluationJob(ctx context.Context, jobID string) (*maclaw.EvaluationJob, error) {
	_ = ctx
	f.lastJob = jobID
	return &maclaw.EvaluationJob{ID: jobID + "_retry", Kind: maclaw.EvaluationJobKindRun, Status: maclaw.EvaluationJobStatusPending}, nil
}

func TestMaclawRuntimeHandlerListsGetsAndDeletesSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{enabled: true}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.GET("/maclaw/evaluation/sessions", handler.ListSessions)
	router.GET("/maclaw/evaluation/sessions/:id", handler.GetSession)
	router.DELETE("/maclaw/evaluation/sessions/:id", handler.DeleteSession)

	req := httptest.NewRequest(http.MethodGet, "/maclaw/evaluation/sessions?limit=10&include_archived=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "sess_1") {
		t.Fatalf("list status = %d body = %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/maclaw/evaluation/sessions/sess_1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "msg_1") {
		t.Fatalf("get status = %d body = %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/maclaw/evaluation/sessions/sess_1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK || gateway.lastSession != "sess_1" {
		t.Fatalf("delete status = %d body = %s last session = %q", w.Code, w.Body.String(), gateway.lastSession)
	}
}

func TestMaclawRuntimeHandlerStripsExecutionGrantFromSessionMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:        "msg_confirm",
			SessionID: "sess_1",
			Role:      "user",
			Content:   "confirm",
			Metadata: map[string]string{
				"evaluation_action":                     "confirm_plan",
				"evaluation_execution_grant":            "must-not-leak",
				"evaluation_execution_grant_expires_at": "9999999999",
			},
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.GET("/maclaw/evaluation/sessions/:id", handler.GetSession)

	req := httptest.NewRequest(http.MethodGet, "/maclaw/evaluation/sessions/sess_1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, "must-not-leak") || strings.Contains(body, "evaluation_execution_grant") {
		t.Fatalf("session response leaked execution grant: %s", body)
	}
}

func TestMaclawRuntimeHandlerRestoresRunningProgressCardOnSessionRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:        "msg_plan",
			SessionID: "sess_1",
			Role:      "assistant",
			Content:   `{"response_source":"plan_confirm","risk_types":["jailbreak"],"test_count":3}`,
		}, {
			ID:        "msg_confirm",
			SessionID: "sess_1",
			Role:      "user",
			Content:   "confirm",
			Metadata: map[string]string{
				"evaluation_action": "confirm_plan",
			},
		}},
		jobs: []maclaw.EvaluationJob{{
			ID:     "run_confirm",
			Kind:   maclaw.EvaluationJobKindRun,
			Status: maclaw.EvaluationJobStatusRunning,
			Progress: &maclaw.EvaluationJobProgress{
				RunID:         "run_confirm",
				SessionID:     "sess_1",
				UserMessageID: "msg_confirm",
				Phase:         "compose_payloads",
				StatusText:    "正在组合评估载荷",
			},
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.GET("/maclaw/evaluation/sessions/:id", handler.GetSession)

	req := httptest.NewRequest(http.MethodGet, "/maclaw/evaluation/sessions/sess_1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.lastJobQuery.SessionID != "sess_1" || gateway.lastJobQuery.Status != maclaw.EvaluationJobStatusRunning {
		t.Fatalf("job query = %#v", gateway.lastJobQuery)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"card_type":"progress"`) || !strings.Contains(body, `"job_id":"run_confirm"`) || !strings.Contains(body, "正在组合评估载荷") {
		t.Fatalf("session response did not include progress card: %s", body)
	}
}

func TestMaclawRuntimeHandlerDoesNotRestoreProgressForNormalMessageRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:        "msg_user",
			SessionID: "sess_1",
			Role:      "user",
			Content:   "请进行文言文越狱测试",
		}},
		jobs: []maclaw.EvaluationJob{{
			ID:     "run_message",
			Kind:   maclaw.EvaluationJobKindRun,
			Status: maclaw.EvaluationJobStatusRunning,
			Progress: &maclaw.EvaluationJobProgress{
				RunID:         "run_message",
				SessionID:     "sess_1",
				UserMessageID: "msg_user",
				Phase:         "assistant_message",
				StatusText:    "评估任务正在执行",
			},
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.GET("/maclaw/evaluation/sessions/:id", handler.GetSession)

	req := httptest.NewRequest(http.MethodGet, "/maclaw/evaluation/sessions/sess_1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if strings.Contains(body, `"card_type":"progress"`) || strings.Contains(body, `"job_id":"run_message"`) || strings.Contains(body, "评估任务正在执行") {
		t.Fatalf("normal message run should not restore evaluation progress card: %s", body)
	}
}

func TestNormalizeRuntimeMessageConvertsAskUserJSONToSafeQuestion(t *testing.T) {
	msg := &maclaw.RuntimeMessage{
		ID:         "msg_ask",
		Role:       "assistant",
		OutputType: "application/json",
		Content:    `{"response_source":"ask_user","question":"How many payloads should be tested?","options":["3","5"]}`,
		Metadata: map[string]string{
			redteamExecutionGrantMetadataKey: "must-not-leak",
		},
	}

	normalizeRuntimeMessage(msg)

	if msg.Content != "How many payloads should be tested?" {
		t.Fatalf("content = %q", msg.Content)
	}
	if msg.OutputType != "text/plain" {
		t.Fatalf("output type = %q", msg.OutputType)
	}
	if msg.Metadata["response_source"] != "ask_user" || msg.Metadata["evaluation_event_type"] != "ask_user" {
		t.Fatalf("metadata = %#v", msg.Metadata)
	}
	if msg.Metadata["ask_user_options_json"] != `["3","5"]` {
		t.Fatalf("options metadata = %q", msg.Metadata["ask_user_options_json"])
	}
	if strings.Contains(msg.Metadata[redteamExecutionGrantMetadataKey], "must-not-leak") {
		t.Fatalf("execution grant leaked: %#v", msg.Metadata)
	}
}

func TestNormalizeRuntimeMessageMarksMarkdownReportAsDownloadableCard(t *testing.T) {
	msg := &maclaw.RuntimeMessage{
		ID:      "msg_report",
		Role:    "assistant",
		Content: "测试执行完毕。\n- 报告 ID：`redteam_report_3606a2a8b1b056c2a2d7cbe9`",
	}

	normalizeRuntimeMessage(msg)

	if msg.Metadata["evaluation_event_type"] != "report" || msg.Metadata["report_id"] != "redteam_report_3606a2a8b1b056c2a2d7cbe9" {
		t.Fatalf("metadata = %#v", msg.Metadata)
	}
	if msg.Metadata["download_format"] != "pdf" || msg.Metadata["downloadable"] != "true" {
		t.Fatalf("download metadata = %#v", msg.Metadata)
	}
}

func TestMaclawRuntimeHandlerKeepsCompletedReportThatMentionsPlanTerms(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:        "msg_plan",
			SessionID: "sess_1",
			Role:      "assistant",
			Content:   `{"response_source":"plan_confirm","risk_types":["jailbreak"],"test_count":5,"requires_confirmation":true}`,
		}, {
			ID:        "msg_confirm",
			SessionID: "sess_1",
			Role:      "user",
			Content:   "confirm",
			Metadata:  map[string]string{"evaluation_action": "confirm_plan"},
		}, {
			ID:        "msg_report",
			SessionID: "sess_1",
			Role:      "assistant",
			Content:   "Evaluation plan finished. Target model, risk, payload selection strategy and confirmation were used. Report ID: `redteam_report_3606a2a8b1b056c2a2d7cbe9`.",
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.GET("/maclaw/evaluation/sessions/:id", handler.GetSession)

	req := httptest.NewRequest(http.MethodGet, "/maclaw/evaluation/sessions/sess_1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, `"evaluation_event_type":"report"`) || !strings.Contains(body, `"report_id":"redteam_report_3606a2a8b1b056c2a2d7cbe9"`) {
		t.Fatalf("completed report message was filtered from session snapshot: %s", body)
	}
}

func TestMaclawRuntimeHandlerHydratesReportCardMetadataFromStoredReport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	score := 42.0
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:        "msg_report",
			SessionID: "sess_1",
			Role:      "assistant",
			Content:   "评估执行完成。\n\n报告 ID：`redteam_report_1`",
		}},
		reports: map[string]*maclaw.EvaluationReport{
			"redteam_report_1": {
				ID:          "redteam_report_1",
				RiskLevel:   "高风险",
				SafetyScore: &score,
				Metadata: map[string]string{
					"executed_count": "5",
					"success_count":  "3",
					"failure_count":  "2",
				},
			},
		},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.GET("/maclaw/evaluation/sessions/:id", handler.GetSession)

	req := httptest.NewRequest(http.MethodGet, "/maclaw/evaluation/sessions/sess_1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	body := w.Body.String()
	for _, want := range []string{`"risk_level":"高风险"`, `"safety_score":"42"`, `"executed_count":"5"`, `"success_count":"3"`, `"failure_count":"2"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("report metadata was not hydrated with %s: %s", want, body)
		}
	}
}

func TestMaclawRuntimeHandlerCreatesSessionWithConfiguredInstance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{enabled: true}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions", handler.CreateSession)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions", strings.NewReader(`{"title":"Enterprise assessment"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.lastInstID != "inst_1" || !strings.Contains(w.Body.String(), "sess_1") {
		t.Fatalf("instance = %q body = %s", gateway.lastInstID, w.Body.String())
	}
}

func TestMaclawRuntimeHandlerResolvesInstanceFromAuthenticatedUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{enabled: true}
	resolver := maclaw.NewInstanceResolver("inst_default", []maclaw.InstanceMapping{
		{Role: "enterprise", InstanceID: "inst_role"},
		{UserID: "user_1", InstanceID: "inst_user"},
	})
	handler := NewMaclawRuntimeHandlerWithResolver(gateway, resolver)
	router := newEnterpriseRoleRouter()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", "user_1")
		c.Set("user_role", "enterprise")
		c.Next()
	})
	router.POST("/maclaw/evaluation/sessions", handler.CreateSession)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions", strings.NewReader(`{"title":"Enterprise assessment"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.lastInstID != "inst_user" {
		t.Fatalf("instance = %q, want user mapping", gateway.lastInstID)
	}
}

func TestMaclawRuntimeHandlerConfirmsPlanByStartingEvaluationJob(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:       "msg_plan",
			Role:     "assistant",
			Metadata: map[string]string{"response_source": "plan_confirm"},
			Content:  `{"response_source":"plan_confirm","target_summary":{"target_type":"llm"},"risk_types":["jailbreak"],"test_count":5,"selection_strategy":"maclaw_selected","selected_capability_refs":[{"source_type":"skill","ref":"skillhub:ccbos-classical-chinese-skill"}],"selection_reasons":["matched request"],"requires_confirmation":true}`,
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/confirm", handler.ConfirmPlan)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/confirm", strings.NewReader(`{"test_count":12}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.lastSession != "sess_1" || gateway.lastConfirm.Metadata["evaluation_action"] != "confirm_plan" || gateway.lastConfirm.Metadata["test_count"] != "12" {
		t.Fatalf("last session = %q confirm = %#v", gateway.lastSession, gateway.lastConfirm)
	}
	if !strings.Contains(w.Body.String(), `"kind":"evaluation.run"`) || strings.Contains(w.Body.String(), `"content"`) {
		t.Fatalf("confirm body = %s", w.Body.String())
	}
}

func TestMaclawRuntimeHandlerConfirmsSpecifiedPlanMessageAndOverridesTestCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sampleRef := "sample:" + uuid.NewString()
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:       "msg_old_plan",
			Role:     "assistant",
			Metadata: map[string]string{"response_source": "plan_confirm"},
			Content:  `{"response_source":"plan_confirm","target_summary":{"target_type":"llm"},"risk_types":["prompt_injection"],"test_count":2,"selection_strategy":"sequential","requires_confirmation":true}`,
		}, {
			ID:       "msg_selected_plan",
			Role:     "assistant",
			Metadata: map[string]string{"response_source": "plan_confirm"},
			Content:  `{"response_source":"plan_confirm","target_summary":{"target_type":"llm"},"risk_types":["jailbreak"],"test_count":5,"selection_strategy":"random","selected_capability_refs":[{"source_type":"skill","ref":"skillhub:ccbos-classical-chinese-skill"},{"source_type":"sample","ref":"` + sampleRef + `"}],"selection_reasons":["classical Chinese jailbreak"],"requires_confirmation":true}`,
		}, {
			ID:      "msg_after_plan_text",
			Role:    "assistant",
			Content: "I can wait for your confirmation.",
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	handler.SetExecutionGrantSecret("mcp-secret")
	userID := uuid.New()
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/confirm", func(c *gin.Context) {
		c.Set("user_id", userID.String())
		handler.ConfirmPlan(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/confirm", strings.NewReader(`{"plan_message_id":"msg_selected_plan","test_count":3}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.confirmCalls != 1 {
		t.Fatalf("confirmCalls = %d", gateway.confirmCalls)
	}
	if gateway.lastConfirm.Metadata["test_count"] != "3" || gateway.lastConfirm.Metadata["plan_message_id"] != "msg_selected_plan" || !strings.Contains(gateway.lastConfirm.Metadata["selected_skill_names_json"], "ccbos-classical-chinese-skill") {
		t.Fatalf("confirm metadata = %#v", gateway.lastConfirm.Metadata)
	}
	grant := gateway.lastConfirm.Metadata[redteamExecutionGrantMetadataKey]
	payload, err := verifyRedteamExecutionGrantPayload("mcp-secret", grant, userID.String(), time.Now())
	if err != nil {
		t.Fatalf("verify grant: %v", err)
	}
	if payload.TestCount != 3 || !containsString(payload.SelectedCapabilityRefs, sampleRef) || !containsString(payload.SelectedSkillNames, "ccbos-classical-chinese-skill") {
		t.Fatalf("grant payload = %#v, want selected sample and skill", payload)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(want)) {
			return true
		}
	}
	return false
}

func TestMaclawRuntimeHandlerReturnsUnstructuredPlanTextWithoutRewrite(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		postMessages: []*maclaw.RuntimeMessageResponse{{
			Message: &maclaw.RuntimeMessage{ID: "msg_unstructured", Role: "assistant", Content: "Execution plan\nTarget: current tested model\nRisk: jailbreak\nRounds: 3\nStrategy: random"},
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/messages", handler.PostMessage)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/messages", strings.NewReader(`{"content":"test current model for jailbreak with 3 random payloads"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if len(gateway.postCalls) != 1 {
		t.Fatalf("post calls = %d, want only original user message", len(gateway.postCalls))
	}
	var out maclaw.RuntimeMessageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Message == nil || out.Message.Metadata["evaluation_event_type"] == "plan_confirm" {
		t.Fatalf("message = %#v", out.Message)
	}
}

func TestMaclawRuntimeHandlerRejectsStalePlanConfirm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:       "msg_plan",
			Role:     "assistant",
			Metadata: map[string]string{"response_source": "plan_confirm"},
			Content:  `{"response_source":"plan_confirm","target_summary":{"target_type":"llm"},"risk_types":["jailbreak"],"test_count":3,"selection_strategy":"random","selected_capability_refs":[{"source_type":"sample","ref":"sample:old"}],"requires_confirmation":true}`,
		}, {
			ID:       "msg_followup",
			Role:     "assistant",
			Metadata: map[string]string{"response_source": "ask_user"},
			Content:  `{"response_source":"ask_user","question":"请选择测试轮次"}`,
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/confirm", handler.ConfirmPlan)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/confirm", strings.NewReader(`{"test_count":3}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.confirmCalls != 0 {
		t.Fatalf("stale plan should not be confirmed, calls = %d", gateway.confirmCalls)
	}
}

func TestMaclawRuntimeHandlerAllowsRetryAfterCanceledConfirmTurn(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:       "msg_plan",
			Role:     "assistant",
			Metadata: map[string]string{"response_source": "plan_confirm"},
			Content:  `{"response_source":"plan_confirm","target_summary":{"target_type":"llm"},"risk_types":["jailbreak"],"test_count":3,"selection_strategy":"maclaw_selected","selected_capability_refs":[{"source_type":"skill","ref":"skillhub:ccbos-classical-chinese-skill"}],"requires_confirmation":true}`,
		}, {
			ID:       "msg_confirm",
			Role:     "user",
			Metadata: map[string]string{"evaluation_action": "confirm_plan"},
			Content:  "confirm",
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/confirm", handler.ConfirmPlan)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/confirm", strings.NewReader(`{"test_count":3}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.confirmCalls != 1 {
		t.Fatalf("ConfirmRuntimePlan calls = %d", gateway.confirmCalls)
	}
}

func TestMaclawRuntimeHandlerRejectsConfirmWithoutPlanConfirm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:       "msg_ask",
			Role:     "assistant",
			Metadata: map[string]string{"response_source": "ask_user"},
			Content:  `{"response_source":"ask_user","question":"Please provide a target model."}`,
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/confirm", handler.ConfirmPlan)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/confirm", strings.NewReader(`{"test_count":3}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.confirmCalls != 0 {
		t.Fatalf("ConfirmRuntimePlan should not be called without plan_confirm, got %d calls", gateway.confirmCalls)
	}
	if !strings.Contains(w.Body.String(), "maclaw_plan_confirm_required") {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestMaclawRuntimeHandlerRejectsIncompletePlanConfirm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:       "msg_plan",
			Role:     "assistant",
			Metadata: map[string]string{"response_source": "plan_confirm"},
			Content:  `{"response_source":"plan_confirm","target_summary":{"target_type":"llm"},"selection_strategy":"random","requires_confirmation":true}`,
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/confirm", handler.ConfirmPlan)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/confirm", strings.NewReader(`{"test_count":0}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.confirmCalls != 0 {
		t.Fatalf("ConfirmRuntimePlan should not be called for incomplete plan, got %d calls", gateway.confirmCalls)
	}
	if !strings.Contains(w.Body.String(), "maclaw_plan_confirm_incomplete") || !strings.Contains(w.Body.String(), "risk_types") || !strings.Contains(w.Body.String(), "test_count") {
		t.Fatalf("body = %s", w.Body.String())
	}
}

func TestMaclawRuntimeHandlerConfirmsNestedPlanConfirmPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{{
			ID:       "msg_plan",
			Role:     "assistant",
			Metadata: map[string]string{"response_source": "plan_confirm"},
			Content: `{
				"response_source":"plan_confirm",
				"plan":{
					"target_summary":"当前被测多模态模型",
					"risk_types":["jailbreak"],
					"selected_capability_refs":["skillhub:figstep-typographic-visual-skill"],
					"selected_skills":["figstep-typographic-visual-skill"],
					"selection_strategy":"random",
					"test_count":1,
					"requires_confirmation":true
				}
			}`,
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/confirm", handler.ConfirmPlan)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/confirm", strings.NewReader(`{"plan_message_id":"msg_plan"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.confirmCalls != 1 {
		t.Fatalf("ConfirmRuntimePlan calls = %d, want 1", gateway.confirmCalls)
	}
	if got := gateway.lastConfirm.Metadata["test_count"]; got != "1" {
		t.Fatalf("test_count metadata = %q", got)
	}
	if got := gateway.lastConfirm.Metadata["selection_strategy"]; got != "random" {
		t.Fatalf("selection_strategy metadata = %q", got)
	}
	if got := gateway.lastConfirm.Metadata["selected_skill_names_json"]; !strings.Contains(got, "figstep-typographic-visual-skill") {
		t.Fatalf("selected_skill_names_json = %q", got)
	}
}

func TestMaclawRuntimeHandlerPreflightsSkillTenantModelBeforeConfirm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var configTestCalled bool
	var confirmCalled bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/evaluation/sessions/sess_1/messages":
			_ = json.NewEncoder(w).Encode(gin.H{"items": []maclaw.RuntimeMessage{{
				ID:       "msg_plan",
				Role:     "assistant",
				Metadata: map[string]string{"response_source": "plan_confirm"},
				Content:  `{"response_source":"plan_confirm","target_summary":{"target_type":"llm"},"risk_types":["jailbreak"],"test_count":20,"selection_strategy":"maclaw_selected","selected_skills":["ccbos-classical-chinese-skill"],"selected_capability_refs":["skillhub:ccbos-classical-chinese-skill"],"requires_confirmation":true}`,
			}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/config/test":
			configTestCalled = true
			_ = json.NewEncoder(w).Encode(maclaw.RuntimeConfigTestResult{
				Success:      false,
				Error:        "connection failed",
				ProviderName: "tenant-llm",
				Model:        "configured-model",
				Endpoint:     "https://model.example/v1",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/evaluation/sessions/sess_1/confirm":
			confirmCalled = true
			_ = json.NewEncoder(w).Encode(maclaw.EvaluationJob{ID: "job_1", Kind: maclaw.EvaluationJobKindRun, Status: maclaw.EvaluationJobStatusPending})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer upstream.Close()
	client, err := maclaw.NewClient(maclaw.Config{BaseURL: upstream.URL, APIToken: "tenant-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	provider := &fakeGatewayProvider{session: &maclaw.GatewaySession{Client: client, InstanceID: "inst_enterprise"}}
	handler := NewMaclawRuntimeHandlerWithProvider(provider)
	router := newEnterpriseRoleRouter()
	userID := uuid.New().String()
	router.POST("/maclaw/evaluation/sessions/:id/confirm", func(c *gin.Context) {
		c.Set("user_id", userID)
		handler.ConfirmPlan(c)
	})

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/confirm", strings.NewReader(`{"test_count":20,"plan_message_id":"msg_plan"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if !configTestCalled {
		t.Fatalf("expected tenant model preflight before skill-backed confirm")
	}
	if confirmCalled {
		t.Fatalf("ConfirmRuntimePlan should not be called when tenant model preflight fails")
	}
	body := w.Body.String()
	if !strings.Contains(body, "maclaw_model_unreachable") || strings.Contains(body, "tenant-token") || strings.Contains(body, "connection failed") {
		t.Fatalf("body should be safe and actionable, got %s", body)
	}
}

func TestMaclawRuntimeHandlerPreparesOnlySelectedResourcesOnConfirm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx := context.Background()
	expertID := uuid.New()
	enterpriseID := uuid.New()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/evaluation/resources/materialize" {
			t.Fatalf("unexpected source request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(maclaw.EvaluationResourceMaterialization{
			ResourceID: "res_pub",
			Handle:     "expert_handle",
			Name:       "Expert prompt",
			Kind:       maclaw.EvaluationResourceKindSample,
			Version:    "v1",
			Payload:    "SECRET_PAYLOAD",
		})
	}))
	defer source.Close()

	var shadowCreated bool
	var confirmCalled bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/evaluation/sessions/sess_1/messages":
			_ = json.NewEncoder(w).Encode(gin.H{"items": []maclaw.RuntimeMessage{{
				ID:         "msg_plan",
				SessionID:  "sess_1",
				Role:       "assistant",
				OutputType: "application/vnd.maclaw.plan-confirm+json",
				Content: `{
					"response_source":"plan_confirm",
					"target_summary":{"target_type":"llm"},
					"risk_types":["jailbreak"],
					"test_count":3,
					"selection_strategy":"maclaw_selected",
					"selected_capabilities":[
						{"source_type":"resource","source_ref":"expert_handle"}
					],
					"requires_confirmation":true
				}`,
			}}})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/evaluation/resources":
			shadowCreated = true
			var in maclaw.EvaluationResourceInput
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Fatalf("decode shadow resource: %v", err)
			}
			if in.Payload != "SECRET_PAYLOAD" {
				t.Fatalf("shadow payload = %q", in.Payload)
			}
			_ = json.NewEncoder(w).Encode(maclaw.EvaluationResourceSummary{
				ID:      "shadow_res",
				Handle:  "shadow_handle",
				Name:    in.Name,
				Kind:    in.Kind,
				Version: in.Version,
				Status:  in.Status,
				Enabled: in.Enabled,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/evaluation/sessions/sess_1/confirm":
			confirmCalled = true
			if !shadowCreated {
				t.Fatalf("confirm called before selected resource shadow was prepared")
			}
			_ = json.NewEncoder(w).Encode(maclaw.EvaluationJob{ID: "job_1", Kind: maclaw.EvaluationJobKindRun, Status: maclaw.EvaluationJobStatusPending})
		default:
			t.Fatalf("unexpected enterprise request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer target.Close()

	sourceClient, err := maclaw.NewClient(maclaw.Config{BaseURL: source.URL, APIToken: "source-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("source client: %v", err)
	}
	targetClient, err := maclaw.NewClient(maclaw.Config{BaseURL: target.URL, APIToken: "target-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("target client: %v", err)
	}
	provider := &projectionHandlerProvider{sessions: map[string]*maclaw.GatewaySession{
		expertID.String():     {Client: sourceClient, Mapping: &model.MaclawAccountMapping{PlatformUserID: expertID, PlatformRole: model.RoleExpert, MaclawTenantID: "tenant_expert"}},
		enterpriseID.String(): {Client: targetClient, InstanceID: "inst_enterprise", Mapping: &model.MaclawAccountMapping{PlatformUserID: enterpriseID, PlatformRole: model.RoleEnterprise, MaclawTenantID: "tenant_enterprise"}},
	}}
	publications := newProjectionHandlerPublicationStore()
	if err := publications.UpsertPublication(ctx, &model.MaclawResourcePublication{
		SourceExpertUserID:   expertID,
		SourceMaclawTenantID: "tenant_expert",
		SourceResourceID:     "res_pub",
		SourceResourceHandle: "expert_handle",
		SourceVersion:        "v1",
		Name:                 "Expert prompt",
		Kind:                 string(maclaw.EvaluationResourceKindSample),
		Status:               string(maclaw.EvaluationResourceStatusPublished),
		Enabled:              true,
		Summary:              "safe summary",
	}); err != nil {
		t.Fatalf("seed publication: %v", err)
	}
	projection := maclaw.NewResourceProjectionService(provider, publications, newProjectionHandlerShadowStore())
	handler := NewMaclawRuntimeHandlerWithProviderAndProjections(provider, projection, nil)

	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/confirm", func(c *gin.Context) {
		c.Set("user_id", enterpriseID.String())
		c.Set("user_role", "enterprise")
		handler.ConfirmPlan(c)
	})
	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/confirm", strings.NewReader(`{"test_count":3}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if !shadowCreated || !confirmCalled {
		t.Fatalf("shadowCreated=%v confirmCalled=%v", shadowCreated, confirmCalled)
	}
}

func TestMaclawRuntimeHandlerDoesNotRestoreOrphanRuntimeRunAsEvaluationProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{
			{
				ID:        "msg_plan",
				SessionID: "sess_1",
				Role:      "assistant",
				Content:   `{"response_source":"plan_confirm","target_summary":"target","risk_types":["jailbreak"],"test_count":5,"requires_confirmation":true}`,
			},
			{
				ID:        "msg_confirm",
				SessionID: "sess_1",
				Role:      "user",
				Content:   "confirm",
				Metadata:  map[string]string{"evaluation_action": "confirm_plan"},
			},
		},
		jobs: []maclaw.EvaluationJob{{
			ID:     "run_chat_turn",
			Kind:   maclaw.EvaluationJobKindRun,
			Status: maclaw.EvaluationJobStatusRunning,
			Progress: &maclaw.EvaluationJobProgress{
				RunID:     "run_chat_turn",
				SessionID: "sess_1",
			},
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.GET("/maclaw/evaluation/sessions/:id", handler.GetSession)

	req := httptest.NewRequest(http.MethodGet, "/maclaw/evaluation/sessions/sess_1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "job-progress:run_chat_turn") || strings.Contains(w.Body.String(), `"card_type":"progress"`) {
		t.Fatalf("orphan runtime run should not be restored as evaluation progress: %s", w.Body.String())
	}
}

func TestMaclawRuntimeHandlerRestoresConfirmedRuntimeRunProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		messages: []maclaw.RuntimeMessage{
			{
				ID:        "msg_confirm",
				SessionID: "sess_1",
				Role:      "user",
				Content:   "confirm",
				Metadata:  map[string]string{"evaluation_action": "confirm_plan"},
			},
		},
		jobs: []maclaw.EvaluationJob{{
			ID:     "run_eval",
			Kind:   maclaw.EvaluationJobKindRun,
			Status: maclaw.EvaluationJobStatusRunning,
			Progress: &maclaw.EvaluationJobProgress{
				RunID:         "run_eval",
				SessionID:     "sess_1",
				UserMessageID: "msg_confirm",
			},
		}},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.GET("/maclaw/evaluation/sessions/:id", handler.GetSession)

	req := httptest.NewRequest(http.MethodGet, "/maclaw/evaluation/sessions/sess_1", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "job-progress:run_eval") || !strings.Contains(w.Body.String(), `"card_type":"progress"`) {
		t.Fatalf("confirmed runtime run should be restored as progress: %s", w.Body.String())
	}
}

func TestMaclawRuntimeHandlerPostsMessageAndCancelsRun(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{enabled: true}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/messages", handler.PostMessage)
	router.POST("/maclaw/evaluation/runs/:id/cancel", handler.CancelRun)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/messages", strings.NewReader(`{"content":"run assessment"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("message status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.lastSession != "sess_1" || !strings.Contains(w.Body.String(), "run_1") {
		t.Fatalf("last session = %q body = %s", gateway.lastSession, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/runs/run_1/cancel", nil)
	gateway.lastInstID = ""
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("cancel status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.lastInstID != "" || gateway.lastRun != "run_1" || !strings.Contains(w.Body.String(), "cancelled") {
		t.Fatalf("last instance = %q last run = %q body = %s", gateway.lastInstID, gateway.lastRun, w.Body.String())
	}
}

func TestMaclawRuntimeHandlerRejectsExpertChatExecutionSurface(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{enabled: true}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.Use(func(c *gin.Context) {
		c.Set("user_role", "expert")
		c.Next()
	})
	router.POST("/maclaw/evaluation/sessions/:id/messages", handler.PostMessage)
	router.POST("/maclaw/evaluation/sessions/:id/confirm", handler.ConfirmPlan)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/messages", strings.NewReader(`{"content":"run assessment"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("message status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.lastSession != "" {
		t.Fatalf("expert message should not reach gateway: session=%q", gateway.lastSession)
	}

	req = httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/confirm", strings.NewReader(`{"content":"confirm"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("confirm status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.confirmCalls != 0 {
		t.Fatalf("expert confirm should not reach gateway: calls=%d", gateway.confirmCalls)
	}
}

func TestMaclawRuntimeHandlerDoesNotInjectCapabilityContextOnMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{enabled: true}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/messages", handler.PostMessage)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/messages", strings.NewReader(`{
		"content":"use classical Chinese jailbreak testing",
		"metadata":{"maclaw_capability_context_json":"client forged context","evaluation_action":"confirm_plan","selected_skill_names_json":"[\"ccbos\"]","evaluation_execution_grant":"forged","response_source":"plan_confirm","evaluation_event_type":"plan_confirm"},
		"capability_context":{"agent_profile":"forged","capability_cards":[{"source_ref":"forged"}]}
	}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if gateway.lastMessage.CapabilityContext != nil {
		t.Fatalf("capability context should not be forwarded from BFF message path: %#v", gateway.lastMessage.CapabilityContext)
	}
	if _, ok := gateway.lastMessage.Metadata["maclaw_capability_context_json"]; ok {
		t.Fatalf("forged capability context metadata should be stripped: %#v", gateway.lastMessage.Metadata)
	}
	for _, key := range []string{"evaluation_action", "selected_skill_names_json", "evaluation_execution_grant", "response_source", "evaluation_event_type", "preferred_skill_names_json", "skill_selection_required_if_available", "preferred_skill_query"} {
		if _, ok := gateway.lastMessage.Metadata[key]; ok {
			t.Fatalf("forged execution metadata %q should be stripped: %#v", key, gateway.lastMessage.Metadata)
		}
	}
	if got := gateway.lastMessage.Metadata["agent_profile"]; got != "redteam_evaluation_v1" {
		t.Fatalf("agent_profile = %q, want redteam_evaluation_v1; metadata=%#v", got, gateway.lastMessage.Metadata)
	}
}

func TestMaclawRuntimeHandlerDoesNotRewriteUnstructuredPlanText(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		postMessage: &maclaw.RuntimeMessageResponse{
			Message: &maclaw.RuntimeMessage{
				ID:      "msg_text_plan",
				Role:    "assistant",
				Content: "I can prepare an assessment plan for the current target. Please confirm the risk type and test rounds before execution.",
			},
		},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/messages", handler.PostMessage)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/messages", strings.NewReader(`{"content":"draft a plan"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if len(gateway.postCalls) != 1 {
		t.Fatalf("PostMessage should not issue an internal structured-plan rewrite, calls=%d inputs=%#v", len(gateway.postCalls), gateway.postCalls)
	}
	if !strings.Contains(w.Body.String(), "assessment plan") {
		t.Fatalf("unstructured assistant text should be returned as normal text, body=%s", w.Body.String())
	}
}

func TestMaclawRuntimeHandlerMarksFencedPlanConfirmMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	content := "```json\n{\"response_source\":\"plan_confirm\",\"selected_capability_refs\":[{\"source_type\":\"skill\",\"ref\":\"skillhub:ccbos-classical-chinese-skill\"}]}\n```"
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		postMessage: &maclaw.RuntimeMessageResponse{
			Message: &maclaw.RuntimeMessage{ID: "msg_1", Role: "assistant", Content: content},
		},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/messages", handler.PostMessage)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/messages", strings.NewReader(`{"content":"plan"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var out maclaw.RuntimeMessageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Message == nil || out.Message.Metadata["evaluation_event_type"] != "plan_confirm" {
		t.Fatalf("message metadata = %#v", out.Message)
	}
	if out.Message.OutputType != "application/vnd.maclaw.plan-confirm+json" {
		t.Fatalf("output type = %q", out.Message.OutputType)
	}
}

func TestMaclawRuntimeHandlerMarksProseWrappedFencedPlanConfirmMessages(t *testing.T) {
	gin.SetMode(gin.TestMode)
	content := "信息已齐备。以下是评估计划：\n\n```json\n{\"response_source\":\"plan_confirm\",\"target_summary\":\"current model\",\"risk_types\":[\"jailbreak\"],\"selected_capability_refs\":[\"skillhub:ccbos-classical-chinese-skill\"],\"selected_skills\":[\"CCBOS Classical Chinese Skill\"],\"test_count\":5,\"selection_strategy\":\"random\",\"requires_confirmation\":true}\n```"
	gateway := &fakeMaclawRuntimeGateway{
		enabled: true,
		postMessage: &maclaw.RuntimeMessageResponse{
			Message: &maclaw.RuntimeMessage{ID: "msg_1", Role: "assistant", Content: content},
		},
	}
	handler := NewMaclawRuntimeHandler(gateway, "inst_1")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions/:id/messages", handler.PostMessage)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions/sess_1/messages", strings.NewReader(`{"content":"plan"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	var out maclaw.RuntimeMessageResponse
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if out.Message == nil || out.Message.Metadata["evaluation_event_type"] != "plan_confirm" {
		t.Fatalf("message metadata = %#v", out.Message)
	}
	if out.Message.OutputType != "application/vnd.maclaw.plan-confirm+json" {
		t.Fatalf("output type = %q", out.Message.OutputType)
	}
	if strings.Contains(out.Message.Content, "信息已齐备") || strings.Contains(out.Message.Content, "```") {
		t.Fatalf("plan content should be normalized to JSON only, got %q", out.Message.Content)
	}
}

func TestFilterBrowserRuntimeMessagesDowngradesNormalRunProgressMessages(t *testing.T) {
	messages := []maclaw.RuntimeMessage{{
		ID:        "msg_user",
		SessionID: "sess_1",
		Role:      "user",
		Content:   "请对当前被测模型进行文言文越狱测试",
	}, {
		ID:        "msg_progress",
		SessionID: "sess_1",
		Role:      "assistant",
		Content:   "",
		Metadata: map[string]string{
			"card_type":             "progress",
			"evaluation_event_type": "progress",
			"status_text":           "评估任务正在执行",
			"job_id":                "run_normal_message",
			"phase":                 "assistant_message",
		},
	}}

	filtered := filterBrowserRuntimeMessages(messages)
	if len(filtered) != 2 {
		t.Fatalf("filtered messages = %#v", filtered)
	}
	msg := filtered[1]
	if msg.Metadata["card_type"] == "progress" || msg.Metadata["job_id"] != "" || msg.Metadata["evaluation_event_type"] == "progress" {
		t.Fatalf("normal runtime progress should be downgraded to safe text: %#v", msg)
	}
	if strings.TrimSpace(msg.Content) == "" {
		t.Fatalf("downgraded message should keep a safe waiting text: %#v", msg)
	}
}

func TestFilterBrowserRuntimeMessagesKeepsUnstructuredPlanTextAsOrdinaryReply(t *testing.T) {
	messages := []maclaw.RuntimeMessage{{
		ID:      "msg_user",
		Role:    "user",
		Content: "内容安全合规，10条",
	}, {
		ID:      "msg_raw_plan",
		Role:    "assistant",
		Content: "方案已明确，以下为执行计划确认：\n\n```json\n{\"target_summary\":\"current model\",\"risk_types\":[\"jailbreak\"],\"test_count\":10,\"selection_strategy\":\"random\",\"requires_confirmation\":true}\n```",
	}, {
		ID:      "msg_plan",
		Role:    "assistant",
		Content: "```json\n{\"response_source\":\"plan_confirm\",\"target_summary\":{\"target_type\":\"llm\"},\"risk_types\":[\"jailbreak\"],\"test_count\":10,\"selection_strategy\":\"random\",\"requires_confirmation\":true}\n```",
	}}

	filtered := filterBrowserRuntimeMessages(messages)
	if len(filtered) != 3 {
		t.Fatalf("filtered messages = %#v", filtered)
	}
	foundRaw := false
	for _, msg := range filtered {
		if msg.ID == "msg_raw_plan" {
			foundRaw = true
		}
	}
	if !foundRaw {
		t.Fatalf("unstructured assistant text should not be hidden by keyword guessing: %#v", filtered)
	}
}

func TestFilterBrowserRuntimeMessagesKeepsRawPlanTextBeforeStructuredPlanArrives(t *testing.T) {
	messages := []maclaw.RuntimeMessage{{
		ID:      "msg_user",
		Role:    "user",
		Content: "请进行文言文越狱测试",
	}, {
		ID:      "msg_raw_plan",
		Role:    "assistant",
		Content: "已确认 CCBOS Classical Chinese Skill 已安装。以下是执行计划：\n\n```json\n{\"response_source\":\"plan_confirm\",\"target_summary\":\"current model\",\"risk_types\":[\"jailbreak\"],\"selected_capability_refs\":[\"skillhub:CCBOS Classical Chinese Skill\"],\"test_count\":5,\"requires_confirmation\":true}\n```",
	}}

	filtered := filterBrowserRuntimeMessages(messages)
	if len(filtered) != 2 {
		t.Fatalf("filtered messages = %#v", filtered)
	}
	foundRaw := false
	for _, msg := range filtered {
		if msg.ID == "msg_raw_plan" {
			foundRaw = true
		}
	}
	if !foundRaw {
		t.Fatalf("raw assistant plan prose should remain visible as ordinary assistant text: %#v", filtered)
	}
}

func TestSelectedCapabilityRefsFromPlanConfirmContent(t *testing.T) {
	resources, skills := selectedCapabilityRefs([]maclaw.RuntimeMessage{{
		OutputType: "application/vnd.maclaw.plan-confirm+json",
		Content: `{
			"resource_handles":["expert_resource_handle"],
			"skill_name":"legacy-skill",
			"selected_capabilities":[
				{"source_type":"skill","source_ref":"ccbos-classical-chinese-skill"},
				{"source_type":"resource","source_ref":"expert_resource_handle"}
			]
		}`,
	}})
	if len(resources) != 1 || resources[0] != "expert_resource_handle" {
		t.Fatalf("resources = %#v", resources)
	}
	if len(skills) != 2 || skills[0] != "legacy-skill" || skills[1] != "ccbos-classical-chinese-skill" {
		t.Fatalf("skills = %#v", skills)
	}
}

func TestSelectedCapabilityRefsFromFencedPlanConfirmContent(t *testing.T) {
	resources, skills := selectedCapabilityRefs([]maclaw.RuntimeMessage{{
		Content: "```json\n{\n  \"response_source\":\"plan_confirm\",\n  \"selected_capability_refs\":[\n    {\"source_type\":\"skill\",\"ref\":\"skillhub:ccbos-classical-chinese-skill\"},\n    {\"source_type\":\"resource\",\"ref\":\"expert-resource-handle\"}\n  ]\n}\n```",
	}})
	if len(resources) != 1 || resources[0] != "expert-resource-handle" {
		t.Fatalf("resources = %#v", resources)
	}
	if len(skills) != 1 || skills[0] != "ccbos-classical-chinese-skill" {
		t.Fatalf("skills = %#v", skills)
	}
}

func TestSelectedCapabilityRefsFromSelectedSkills(t *testing.T) {
	resources, skills := selectedCapabilityRefs([]maclaw.RuntimeMessage{{
		Content: "```json\n{\n  \"response_source\":\"plan_confirm\",\n  \"selected_skills\":[{\"skill_name\":\"ccbos-classical-chinese-skill\"}],\n  \"selected_capability_refs\":[\"skillhub:ccbos-classical-chinese-skill\"]\n}\n```",
	}})
	if len(resources) != 0 {
		t.Fatalf("resources = %#v", resources)
	}
	if len(skills) != 1 || skills[0] != "ccbos-classical-chinese-skill" {
		t.Fatalf("skills = %#v", skills)
	}
}

func TestSelectedCapabilityRefsUsesOnlyConcreteCapabilityRefs(t *testing.T) {
	resources, skills := selectedCapabilityRefs([]maclaw.RuntimeMessage{{
		Content: "```json\n{\n  \"response_source\":\"plan_confirm\",\n  \"selected_capability_refs\":[\n    {\"capability_ref\":\"sample:sample-id\",\"name\":\"sample display\"},\n    {\"capability_ref\":\"template:template-id\",\"name\":\"template display\"},\n    {\"capability_ref\":\"expert_samples:invented\",\"name\":\"invented display\"},\n    {\"name\":\"plain display name\"}\n  ]\n}\n```",
	}})
	if len(resources) != 2 || resources[0] != "sample:sample-id" || resources[1] != "template:template-id" {
		t.Fatalf("resources = %#v", resources)
	}
	if len(skills) != 0 {
		t.Fatalf("skills = %#v", skills)
	}
}

func TestCopySanitizedRuntimeEventStreamRedactsPayloadAndSecrets(t *testing.T) {
	raw := "event: snapshot\n" +
		"data: {\"type\":\"snapshot\",\"snapshot\":{\"run\":{\"id\":\"run_1\"},\"steps\":[{\"test_payload\":\"SECRET-PAYLOAD\",\"target_response\":\"SECRET-RESPONSE\",\"summary\":\"safe\"}],\"metadata\":{\"api_key\":\"SECRET-KEY\",\"evidence_content\":\"SECRET-EVIDENCE\",\"evaluation_execution_grant\":\"SECRET-GRANT\"}}}\n\n"
	var out strings.Builder
	if err := copySanitizedRuntimeEventStream(&out, strings.NewReader(raw), nil); err != nil {
		t.Fatalf("copySanitizedRuntimeEventStream: %v", err)
	}
	got := out.String()
	for _, forbidden := range []string{"SECRET-PAYLOAD", "SECRET-RESPONSE", "SECRET-KEY", "SECRET-EVIDENCE", "SECRET-GRANT"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("sanitized stream leaked %q in %s", forbidden, got)
		}
	}
	if !strings.Contains(got, `"summary":"safe"`) || !strings.Contains(got, "[redacted by bff]") {
		t.Fatalf("sanitized stream = %s", got)
	}
}

type flushingRuntimeEventWriter struct {
	strings.Builder
	flushes int
}

func (w *flushingRuntimeEventWriter) Flush() {
	w.flushes++
}

func TestCopySanitizedRuntimeEventStreamFlushesEachEvent(t *testing.T) {
	raw := "event: progress\n" +
		"data: {\"type\":\"snapshot\",\"snapshot\":{\"run\":{\"id\":\"run_1\"}}}\n\n" +
		"event: report\n" +
		"data: {\"type\":\"done\",\"snapshot\":{\"run\":{\"id\":\"run_1\"}}}\n\n"
	var out flushingRuntimeEventWriter

	if err := copySanitizedRuntimeEventStream(&out, strings.NewReader(raw), &out); err != nil {
		t.Fatalf("copySanitizedRuntimeEventStream: %v", err)
	}

	if out.flushes < 2 {
		t.Fatalf("flushes = %d, want at least one flush per SSE event", out.flushes)
	}
}

func TestMaclawRuntimeHandlerStreamsRunEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	gateway := &fakeMaclawRuntimeGateway{enabled: true}
	handler := NewMaclawRuntimeHandler(gateway, "")
	router := newEnterpriseRoleRouter()
	router.GET("/maclaw/evaluation/runs/:id/events", handler.StreamRunEvents)

	req := httptest.NewRequest(http.MethodGet, "/maclaw/evaluation/runs/run_1/events", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("content type = %q", got)
	}
	if !strings.Contains(w.Body.String(), "event: report") || gateway.lastRun != "run_1" {
		t.Fatalf("stream body = %q last run = %q", w.Body.String(), gateway.lastRun)
	}
}

func TestMaclawRuntimeHandlerRequiresConfiguredInstance(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewMaclawRuntimeHandler(&fakeMaclawRuntimeGateway{enabled: true}, "")
	router := newEnterpriseRoleRouter()
	router.POST("/maclaw/evaluation/sessions", handler.CreateSession)

	req := httptest.NewRequest(http.MethodPost, "/maclaw/evaluation/sessions", strings.NewReader(`{"title":"x"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
	}
}
