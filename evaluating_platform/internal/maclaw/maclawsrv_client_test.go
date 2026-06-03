package maclaw

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMaclawSrvClientUsesInstanceScopedSessionAndMessageAPIs(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.String())
		if got := r.Header.Get("Authorization"); got != "Bearer user-token" {
			t.Fatalf("authorization = %q", got)
		}
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/instances/inst_1/sessions":
			var in RuntimeSessionInput
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Fatalf("decode session input: %v", err)
			}
			if in.Metadata["agent_profile"] != "redteam_evaluation_v1" {
				t.Fatalf("session metadata = %#v", in.Metadata)
			}
			_ = json.NewEncoder(w).Encode(RuntimeSession{ID: "sess_1", InstanceID: "inst_1", Title: in.Title, Metadata: in.Metadata})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/instances/inst_1/sessions/sess_1/messages":
			var in RuntimeMessageInput
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Fatalf("decode message input: %v", err)
			}
			if in.CapabilityContext == nil || len(in.CapabilityContext.Cards) != 1 {
				t.Fatalf("capability context = %#v", in.CapabilityContext)
			}
			_ = json.NewEncoder(w).Encode(RuntimeMessageResponse{
				Run:     &RuntimeRun{ID: "run_1", SessionID: "sess_1", Status: "succeeded", ResponseSource: "chat"},
				Message: &RuntimeMessage{ID: "msg_1", SessionID: "sess_1", Role: "assistant", Content: "hello", Metadata: map[string]string{"response_source": "chat"}},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	session, err := client.CreateRuntimeSession(context.Background(), "inst_1", RuntimeSessionInput{
		Title:    "Red-team workspace",
		Metadata: map[string]string{"agent_profile": "redteam_evaluation_v1"},
	})
	if err != nil {
		t.Fatalf("CreateRuntimeSession: %v", err)
	}
	if session.ID != "sess_1" {
		t.Fatalf("session = %#v", session)
	}
	out, err := client.PostRuntimeMessage(context.Background(), "inst_1", "sess_1", RuntimeMessageInput{
		Content: "你好",
		CapabilityContext: &RuntimeCapabilityContext{
			AgentProfile: "redteam_evaluation_v1",
			Cards:        []CapabilityCard{{SourceType: CapabilitySourceSkill, SourceRef: "ccbos-classical-chinese-skill", Name: "CCBOS"}},
		},
	})
	if err != nil {
		t.Fatalf("PostRuntimeMessage: %v", err)
	}
	if out.Run.ID != "run_1" || out.Message.Metadata["response_source"] != "chat" {
		t.Fatalf("message response = %#v", out)
	}
	if strings.Join(seen, "\n") != "POST /api/v1/instances/inst_1/sessions\nPOST /api/v1/instances/inst_1/sessions/sess_1/messages" {
		t.Fatalf("seen requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestMaclawSrvClientPostMessageMapsHardExitEmptyContentToSafeAskUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/instances/inst_1/sessions/sess_1/messages" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(RuntimeMessageResponse{
			Run: &RuntimeRun{ID: "run_1", SessionID: "sess_1", Status: "succeeded"},
			Message: &RuntimeMessage{
				ID:         "msg_hard_exit",
				SessionID:  "sess_1",
				Role:       "assistant",
				OutputType: "text/plain",
				Metadata:   map[string]string{"hard_exit": "true"},
			},
		})
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	out, err := client.PostRuntimeMessage(context.Background(), "inst_1", "sess_1", RuntimeMessageInput{Content: "用文言文越狱测试我的目标 LLM"})
	if err != nil {
		t.Fatalf("PostRuntimeMessage: %v", err)
	}
	if out.Message == nil || !strings.Contains(out.Message.Content, "MaClaw") {
		t.Fatalf("message was not mapped to safe fallback: %#v", out.Message)
	}
	if strings.Contains(out.Message.Content, "\u8bf7\u8865\u5145\u88ab\u6d4b\u76ee\u6807") {
		t.Fatalf("fallback should not assume the user is missing target details, got: %q", out.Message.Content)
	}
	if out.Message.Metadata["response_source"] != "ask_user" || out.Message.Metadata["evaluation_event_type"] != "ask_user" {
		t.Fatalf("metadata = %#v", out.Message.Metadata)
	}
	if strings.Contains(out.Message.Content, "payload") || strings.Contains(out.Message.Content, "secret") {
		t.Fatalf("fallback leaked unsafe wording: %q", out.Message.Content)
	}
}

func TestMaclawSrvClientConfirmMapsRunToEvaluationJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/instances/inst_1/sessions/sess_1/messages" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("async") != "true" {
			t.Fatalf("confirm should use async message run, got %s", r.URL.String())
		}
		var in RuntimeMessageInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Fatalf("decode message input: %v", err)
		}
		if in.Metadata["evaluation_action"] != "confirm_plan" {
			t.Fatalf("confirm metadata = %#v", in.Metadata)
		}
		_ = json.NewEncoder(w).Encode(RuntimeMessageResponse{
			Run:     &RuntimeRun{ID: "run_confirm", SessionID: "sess_1", Status: "running", ResponseSource: "plan_confirm"},
			Message: &RuntimeMessage{ID: "msg_confirm", Role: "assistant", Content: `{"status":"accepted"}`},
		})
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	job, err := client.ConfirmRuntimePlan(context.Background(), "inst_1", "sess_1", RuntimeMessageInput{
		Content:  "确认执行",
		Metadata: map[string]string{"evaluation_action": "confirm_plan"},
	})
	if err != nil {
		t.Fatalf("ConfirmRuntimePlan: %v", err)
	}
	if job.ID != "run_confirm" || job.Kind != EvaluationJobKindRun || job.Status != EvaluationJobStatusRunning {
		t.Fatalf("job = %#v", job)
	}
	if job.Progress == nil || job.Progress.RunID != "run_confirm" || job.Result != nil {
		t.Fatalf("job progress/result = %#v / %#v", job.Progress, job.Result)
	}
	if job.Progress.DurationMs <= 0 || !strings.Contains(job.Progress.StageDurationsJSON, "maclaw_confirm") {
		t.Fatalf("progress should include safe confirm duration metadata: %#v", job.Progress)
	}
}

func TestMaclawSrvClientConfirmMapsCompletedMessageToEvaluationJob(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/instances/inst_1/sessions/sess_1/messages" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		if r.URL.Query().Get("async") != "true" {
			t.Fatalf("confirm should use async message run, got %s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(RuntimeMessageResponse{
			Message: &RuntimeMessage{
				ID:        "msg_done",
				SessionID: "sess_1",
				Role:      "assistant",
				Metadata:  map[string]string{"response_source": "chat"},
				Content:   `{"response_source":"chat","status":"completed","run_id":"run_done","report_id":"redteam_report_1","test_count":"3"}`,
			},
		})
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	job, err := client.ConfirmRuntimePlan(context.Background(), "inst_1", "sess_1", RuntimeMessageInput{
		Content:  "confirm execution",
		Metadata: map[string]string{"evaluation_action": "confirm_plan"},
	})
	if err != nil {
		t.Fatalf("ConfirmRuntimePlan: %v", err)
	}
	if job.ID != "run_done" || job.Kind != EvaluationJobKindRun || job.Status != EvaluationJobStatusSucceeded {
		t.Fatalf("job = %#v", job)
	}
	if job.Progress == nil || job.Progress.RunID != "run_done" || job.Progress.Phase != "chat" {
		t.Fatalf("progress = %#v", job.Progress)
	}
	if job.Result == nil || job.Result.Run == nil || job.Result.Run.Metadata["report_id"] != "redteam_report_1" || job.Result.Run.Metadata["test_count"] != "3" {
		t.Fatalf("result = %#v", job.Result)
	}
}

func TestMaclawSrvClientConfirmRecoversCompletedMessageAfterRuntimeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/instances/inst_1/sessions/sess_1/messages":
			http.Error(w, "runtime returned after completing message", http.StatusServiceUnavailable)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/instances/inst_1/sessions/sess_1/messages":
			_ = json.NewEncoder(w).Encode(struct {
				Items []RuntimeMessage `json:"items"`
			}{
				Items: []RuntimeMessage{{
					ID:        "msg_confirm",
					SessionID: "sess_1",
					Role:      "user",
					Content:   "confirm execution",
					Metadata:  map[string]string{"evaluation_action": "confirm_plan"},
				}, {
					ID:        "msg_done",
					SessionID: "sess_1",
					Role:      "assistant",
					Content:   `{"response_source":"chat","status":"completed","run_id":"run_recovered","report_id":"redteam_report_recovered"}`,
				}},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	job, err := client.ConfirmRuntimePlan(context.Background(), "inst_1", "sess_1", RuntimeMessageInput{
		Content:  "confirm execution",
		Metadata: map[string]string{"evaluation_action": "confirm_plan"},
	})
	if err != nil {
		t.Fatalf("ConfirmRuntimePlan: %v", err)
	}
	if job.ID != "run_recovered" || job.Status != EvaluationJobStatusSucceeded || job.Result.Run.Metadata["report_id"] != "redteam_report_recovered" {
		t.Fatalf("job = %#v", job)
	}
}

func TestMaclawSrvClientConfirmDoesNotRecoverStaleCompletedMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/instances/inst_1/sessions/sess_1/messages":
			http.Error(w, "runtime unavailable before current confirm was recorded", http.StatusServiceUnavailable)
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/instances/inst_1/sessions/sess_1/messages":
			_ = json.NewEncoder(w).Encode(struct {
				Items []RuntimeMessage `json:"items"`
			}{
				Items: []RuntimeMessage{{
					ID:        "msg_old_done",
					SessionID: "sess_1",
					Role:      "assistant",
					Content:   `{"response_source":"chat","status":"completed","run_id":"run_old","report_id":"redteam_report_old"}`,
				}},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	job, err := client.ConfirmRuntimePlan(context.Background(), "inst_1", "sess_1", RuntimeMessageInput{
		Content:  "confirm execution",
		Metadata: map[string]string{"evaluation_action": "confirm_plan"},
	})
	if err == nil || job != nil {
		t.Fatalf("expected original runtime error, job=%#v err=%v", job, err)
	}
}

func TestMaclawSrvClientListsAndGetsEvaluationJobsFromInstanceRuns(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.String())
		if got := r.Header.Get("Authorization"); got != "Bearer user-token" {
			t.Fatalf("authorization = %q", got)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/instances/inst_1/runs":
			if r.URL.Query().Get("status") != "succeeded" || r.URL.Query().Get("limit") != "2" {
				t.Fatalf("run list query = %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(struct {
				Items []RuntimeRun `json:"items"`
			}{
				Items: []RuntimeRun{{ID: "run_1", SessionID: "sess_1", Status: "succeeded", ResponseSource: "plan_confirm"}},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/instances/inst_1/runs/run_1":
			_ = json.NewEncoder(w).Encode(RuntimeRun{ID: "run_1", SessionID: "sess_1", Status: "failed", Error: "target timeout"})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	client = client.WithInstanceID("inst_1")

	jobs, err := client.ListEvaluationJobs(context.Background(), EvaluationJobQuery{Status: EvaluationJobStatusSucceeded, Limit: 2})
	if err != nil {
		t.Fatalf("ListEvaluationJobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].ID != "run_1" || jobs[0].Status != EvaluationJobStatusSucceeded || jobs[0].Progress == nil || jobs[0].Progress.RunID != "run_1" {
		t.Fatalf("jobs = %#v", jobs)
	}
	job, err := client.GetEvaluationJob(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("GetEvaluationJob: %v", err)
	}
	if job.ID != "run_1" || job.Status != EvaluationJobStatusFailed || job.Error != "target timeout" {
		t.Fatalf("job = %#v", job)
	}
	if strings.Join(seen, "\n") != "GET /api/v1/instances/inst_1/runs?limit=2&status=succeeded\nGET /api/v1/instances/inst_1/runs/run_1" {
		t.Fatalf("seen requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestMaclawSrvClientDoesNotCallLegacyEvaluationOnlyAPIs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("maclawsrv client should not call legacy evaluation API, got %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	client = client.WithInstanceID("inst_1")

	checks := []struct {
		name string
		fn   func(context.Context) error
	}{
		{name: "search resources", fn: func(ctx context.Context) error {
			_, err := client.SearchEvaluationResources(ctx, EvaluationResourceQuery{})
			return err
		}},
		{name: "save resource", fn: func(ctx context.Context) error {
			_, err := client.SaveEvaluationResource(ctx, EvaluationResourceInput{Name: "Resource"})
			return err
		}},
		{name: "start run", fn: func(ctx context.Context) error {
			_, err := client.StartEvaluationRun(ctx, EvaluationRunInput{})
			return err
		}},
		{name: "report", fn: func(ctx context.Context) error {
			_, err := client.GetEvaluationReport(ctx, "report_1")
			return err
		}},
		{name: "evidence", fn: func(ctx context.Context) error {
			_, err := client.ListEvaluationEvidence(ctx, EvaluationEvidenceQuery{})
			return err
		}},
		{name: "materialize resource", fn: func(ctx context.Context) error {
			_, err := client.MaterializeEvaluationResource(ctx, "resource_1")
			return err
		}},
	}
	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			if err := check.fn(context.Background()); !errors.Is(err, ErrUnsupported) {
				t.Fatalf("error = %v, want ErrUnsupported", err)
			}
		})
	}
}

func TestMaclawSrvClientRecoveryFallbackRequiresManualReview(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.String())
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/instances/inst_1/runs/run_1" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(RuntimeRun{
			ID:        "run_1",
			SessionID: "sess_1",
			Status:    "failed",
			Error:     "target call failed",
			Metadata:  map[string]string{"report_id": "report_1", "payload": "must-not-leak"},
		})
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	client = client.WithInstanceID("inst_1")

	recovery, err := client.GetEvaluationJobRecovery(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("GetEvaluationJobRecovery: %v", err)
	}
	if recovery.JobID != "run_1" || recovery.RunID != "run_1" || recovery.Action != "manual_review" || recovery.Strategy != "manual_review" {
		t.Fatalf("recovery = %#v", recovery)
	}
	if recovery.CanResume || recovery.CanRetry || !recovery.ManualReviewRequired || recovery.RecoveryIndexed {
		t.Fatalf("recovery flags = %#v", recovery)
	}
	body, _ := json.Marshal(recovery)
	if strings.Contains(string(body), "payload") || strings.Contains(string(body), "must-not-leak") || strings.Contains(string(body), "target call failed") {
		t.Fatalf("recovery leaked run internals: %s", body)
	}
	if strings.Join(seen, "\n") != "GET /api/v1/instances/inst_1/runs/run_1" {
		t.Fatalf("seen requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestMaclawSrvClientRetryAndResumeReturnManualReviewConflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("retry/resume fallback should not call upstream, got %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	client = client.WithInstanceID("inst_1")

	for _, check := range []struct {
		name string
		fn   func(context.Context) (*EvaluationJob, error)
	}{
		{name: "retry", fn: func(ctx context.Context) (*EvaluationJob, error) { return client.RetryEvaluationJob(ctx, "run_1") }},
		{name: "resume", fn: func(ctx context.Context) (*EvaluationJob, error) { return client.ResumeEvaluationJob(ctx, "run_1") }},
	} {
		t.Run(check.name, func(t *testing.T) {
			_, err := check.fn(context.Background())
			var upstream *UpstreamError
			if !errors.As(err, &upstream) || upstream.StatusCode != http.StatusConflict {
				t.Fatalf("error = %#v, want 409 upstream conflict", err)
			}
			if strings.Contains(err.Error(), "payload") || strings.Contains(err.Error(), "secret") {
				t.Fatalf("unsafe error = %v", err)
			}
		})
	}
}

func TestMaclawSrvClientCancelsEvaluationJobThroughInstanceRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/instances/inst_1/runs/run_1/cancel" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(RuntimeRun{ID: "run_1", Status: "cancelled"})
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	client = client.WithInstanceID("inst_1")

	job, err := client.CancelEvaluationJob(context.Background(), "run_1")
	if err != nil {
		t.Fatalf("CancelEvaluationJob: %v", err)
	}
	if job.ID != "run_1" || job.Status != EvaluationJobStatusCanceled {
		t.Fatalf("job = %#v", job)
	}
}

func TestMaclawSrvClientRegistersRedteamMCPBridgeWithoutLeakingSecret(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.String())
		if got := r.Header.Get("Authorization"); got != "Bearer user-token" {
			t.Fatalf("authorization = %q", got)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/mcp/servers":
			if r.URL.Query().Get("limit") != "100" {
				t.Fatalf("list query = %s", r.URL.RawQuery)
			}
			_ = json.NewEncoder(w).Encode(struct {
				Items []MCPServerView `json:"items"`
			}{Items: nil})
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/mcp/servers":
			var in MCPServerInput
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				t.Fatalf("decode mcp input: %v", err)
			}
			if in.Kind != "remote" || in.Name != RedteamMCPBridgeName || in.EndpointURL != "https://platform.internal/api/v1/internal/redteam/mcp" {
				t.Fatalf("mcp input = %#v", in)
			}
			if in.AuthType != "bearer" || in.AuthSecret != "bridge-secret" {
				t.Fatalf("auth input = %#v", in)
			}
			if in.Headers["X-Evaluating-Platform-Bridge"] != "redteam_v1" || in.AutoStart {
				t.Fatalf("headers/autostart = %#v / %v", in.Headers, in.AutoStart)
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(MCPServerView{
				ID:            "mcp_remote_1",
				Kind:          "remote",
				Name:          in.Name,
				EndpointURL:   in.EndpointURL,
				AuthType:      "bearer",
				HasAuthSecret: true,
				HeaderNames:   []string{"X-Evaluating-Platform-Bridge"},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	view, err := client.EnsureRedteamMCPBridge(context.Background(), RedteamMCPBridgeInput{
		EndpointURL: "https://platform.internal/api/v1/internal/redteam/mcp",
		AuthSecret:  "bridge-secret",
	})
	if err != nil {
		t.Fatalf("EnsureRedteamMCPBridge: %v", err)
	}
	if view.ID != "mcp_remote_1" || !view.HasAuthSecret || strings.Contains(view.EndpointURL, "bridge-secret") {
		t.Fatalf("mcp view = %#v", view)
	}
	body, _ := json.Marshal(view)
	if strings.Contains(string(body), "bridge-secret") {
		t.Fatalf("mcp view leaked secret: %s", body)
	}
	if strings.Join(seen, "\n") != "GET /api/v1/mcp/servers?limit=100\nPOST /api/v1/mcp/servers" {
		t.Fatalf("seen requests:\n%s", strings.Join(seen, "\n"))
	}
}

func TestMaclawSrvClientReusesExistingRedteamMCPBridge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/mcp/servers" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(struct {
			Items []MCPServerView `json:"items"`
		}{Items: []MCPServerView{{
			ID:            "mcp_existing",
			Kind:          "remote",
			Name:          RedteamMCPBridgeName,
			EndpointURL:   "https://platform.internal/api/v1/internal/redteam/mcp",
			AuthType:      "bearer",
			HasAuthSecret: true,
			HeaderNames:   []string{"X-Evaluating-Platform-Bridge"},
		}}})
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	view, err := client.EnsureRedteamMCPBridge(context.Background(), RedteamMCPBridgeInput{
		EndpointURL: "https://platform.internal/api/v1/internal/redteam/mcp",
	})
	if err != nil {
		t.Fatalf("EnsureRedteamMCPBridge: %v", err)
	}
	if view.ID != "mcp_existing" {
		t.Fatalf("mcp view = %#v", view)
	}
}

func TestMaclawSrvClientUpdatesRedteamMCPBridgeWhenSecretProvided(t *testing.T) {
	var patched bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/mcp/servers":
			_ = json.NewEncoder(w).Encode(struct {
				Items []MCPServerView `json:"items"`
			}{Items: []MCPServerView{{
				ID:            "mcp_existing",
				Kind:          "remote",
				Name:          RedteamMCPBridgeName,
				EndpointURL:   "https://platform.internal/api/v1/internal/redteam/mcp",
				AuthType:      "bearer",
				HasAuthSecret: true,
				HeaderNames:   []string{"X-Evaluating-Platform-Bridge"},
			}}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/mcp/servers/mcp_existing":
			patched = true
			var raw map[string]any
			if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
				t.Fatalf("decode update: %v", err)
			}
			if raw["auth_secret"] != "rotated-secret" {
				t.Fatalf("update did not include rotated auth secret: %#v", raw)
			}
			_ = json.NewEncoder(w).Encode(MCPServerView{ID: "mcp_existing", Name: RedteamMCPBridgeName, HasAuthSecret: true})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	if _, err := client.EnsureRedteamMCPBridge(context.Background(), RedteamMCPBridgeInput{
		EndpointURL: "https://platform.internal/api/v1/internal/redteam/mcp",
		AuthSecret:  "rotated-secret",
	}); err != nil {
		t.Fatalf("EnsureRedteamMCPBridge: %v", err)
	}
	if !patched {
		t.Fatalf("expected existing redteam mcp bridge to be updated when auth secret is provided")
	}
}

func TestMaclawSrvClientUpdatesStaleRedteamMCPBridge(t *testing.T) {
	var seen []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.String())
		if got := r.Header.Get("Authorization"); got != "Bearer user-token" {
			t.Fatalf("authorization = %q", got)
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/mcp/servers":
			_ = json.NewEncoder(w).Encode(struct {
				Items []MCPServerView `json:"items"`
			}{Items: []MCPServerView{{
				ID:            "mcp_existing",
				Kind:          "remote",
				Name:          RedteamMCPBridgeName,
				EndpointURL:   "http://127.0.0.1:18081/api/v1/internal/maclaw/redteam-mcp",
				AuthType:      "bearer",
				HasAuthSecret: true,
				HeaderNames:   []string{"X-Evaluating-Platform-Bridge"},
			}}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/v1/mcp/servers/mcp_existing":
			var raw map[string]any
			if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
				t.Fatalf("decode mcp update: %v", err)
			}
			if _, ok := raw["kind"]; ok {
				t.Fatalf("mcp update must not include create-only kind field: %#v", raw)
			}
			body, _ := json.Marshal(raw)
			var in MCPServerInput
			if err := json.Unmarshal(body, &in); err != nil {
				t.Fatalf("decode mcp input: %v", err)
			}
			if in.Name != RedteamMCPBridgeName || in.EndpointURL != "http://127.0.0.1:8080/api/v1/internal/maclaw/redteam-mcp" {
				t.Fatalf("mcp update = %#v", in)
			}
			if in.AuthType != "bearer" || in.AuthSecret != "bridge-secret" {
				t.Fatalf("auth update = %#v", in)
			}
			if in.Headers["X-Evaluating-Platform-Bridge"] != "redteam_v1" {
				t.Fatalf("headers = %#v", in.Headers)
			}
			_ = json.NewEncoder(w).Encode(MCPServerView{
				ID:            "mcp_existing",
				Kind:          "remote",
				Name:          in.Name,
				EndpointURL:   in.EndpointURL,
				AuthType:      in.AuthType,
				HasAuthSecret: true,
				HeaderNames:   []string{"X-Evaluating-Platform-Bridge"},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.String())
		}
	}))
	defer server.Close()

	client, err := NewMaclawSrvClient(Config{BaseURL: server.URL, APIToken: "user-token", TimeoutSeconds: 1})
	if err != nil {
		t.Fatalf("NewMaclawSrvClient: %v", err)
	}
	view, err := client.EnsureRedteamMCPBridge(context.Background(), RedteamMCPBridgeInput{
		EndpointURL: "http://127.0.0.1:8080/api/v1/internal/maclaw/redteam-mcp",
		AuthSecret:  "bridge-secret",
	})
	if err != nil {
		t.Fatalf("EnsureRedteamMCPBridge: %v", err)
	}
	if view.ID != "mcp_existing" || view.EndpointURL != "http://127.0.0.1:8080/api/v1/internal/maclaw/redteam-mcp" {
		t.Fatalf("mcp view = %#v", view)
	}
	if strings.Join(seen, "\n") != "GET /api/v1/mcp/servers?limit=100\nPATCH /api/v1/mcp/servers/mcp_existing" {
		t.Fatalf("seen requests:\n%s", strings.Join(seen, "\n"))
	}
}
