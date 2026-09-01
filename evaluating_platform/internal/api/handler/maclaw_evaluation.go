package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"evaluating_platform/internal/maclaw"
)

type MaclawEvaluationHandler struct {
	gateway            maclaw.EvaluationGateway
	resolver           *maclaw.InstanceResolver
	provider           maclaw.GatewayProvider
	resourceProjection *maclaw.ResourceProjectionService
	runtimeMode        string
}

func NewMaclawEvaluationHandler(gateway maclaw.EvaluationGateway) *MaclawEvaluationHandler {
	return NewMaclawEvaluationHandlerWithResolver(gateway, nil)
}

func NewMaclawEvaluationHandlerWithResolver(gateway maclaw.EvaluationGateway, resolver *maclaw.InstanceResolver) *MaclawEvaluationHandler {
	return &MaclawEvaluationHandler{gateway: gateway, resolver: resolver}
}

func NewMaclawEvaluationHandlerWithProvider(provider maclaw.GatewayProvider) *MaclawEvaluationHandler {
	return &MaclawEvaluationHandler{provider: provider}
}

func NewMaclawEvaluationHandlerWithProviderAndProjection(provider maclaw.GatewayProvider, projection *maclaw.ResourceProjectionService) *MaclawEvaluationHandler {
	return &MaclawEvaluationHandler{provider: provider, resourceProjection: projection}
}

func (h *MaclawEvaluationHandler) SetRuntimeMode(mode string) {
	if h != nil {
		h.runtimeMode = strings.TrimSpace(mode)
	}
}

func (h *MaclawEvaluationHandler) ListResources(c *gin.Context) {
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	includeInactive, err := parseOptionalBool(c.Query("include_inactive"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "include_inactive must be a boolean"})
		return
	}
	limit, err := parseOptionalPositiveInt(c.Query("limit"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
		return
	}
	items, err := gateway.SearchEvaluationResources(c.Request.Context(), maclaw.EvaluationResourceQuery{
		Kind:            maclaw.EvaluationResourceKind(strings.TrimSpace(c.Query("kind"))),
		AssessmentTypes: parseAssessmentTypes(c),
		Query:           firstNonEmpty(c.Query("query"), c.Query("q")),
		IncludeInactive: includeInactive,
		Limit:           limit,
	})
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	if h.resourceProjection != nil && isEnterpriseLikeRole(c.GetString("user_role")) {
		items, err = h.resourceProjection.ListEnterpriseCatalog(c.Request.Context(), items, maclaw.EvaluationResourceQuery{
			Kind:            maclaw.EvaluationResourceKind(strings.TrimSpace(c.Query("kind"))),
			AssessmentTypes: parseAssessmentTypes(c),
			Query:           firstNonEmpty(c.Query("query"), c.Query("q")),
			IncludeInactive: includeInactive,
			Limit:           limit,
		})
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw resource catalog unavailable", "code": "maclaw_resource_projection_failed"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *MaclawEvaluationHandler) CreateResource(c *gin.Context) {
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	var in maclaw.EvaluationResourceInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := gateway.SaveEvaluationResource(c.Request.Context(), in)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	if h.resourceProjection != nil && h.provider != nil {
		session, ok := resolveMaclawGatewaySession(c, h.provider)
		if !ok {
			return
		}
		if err := h.resourceProjection.RecordExpertResource(c.Request.Context(), maclawRuntimeIdentity(c), session, out); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw resource publication update failed", "code": "maclaw_resource_projection_failed"})
			return
		}
	}
	c.JSON(http.StatusCreated, out)
}

func (h *MaclawEvaluationHandler) ListTargets(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	includeInactive, err := parseOptionalBool(c.Query("include_inactive"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "include_inactive must be a boolean"})
		return
	}
	limit, err := parseOptionalPositiveInt(c.Query("limit"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
		return
	}
	items, err := gateway.SearchEvaluationTargets(c.Request.Context(), maclaw.EvaluationTargetQuery{
		Kind:            maclaw.EvaluationTargetKind(strings.TrimSpace(c.Query("kind"))),
		Provider:        strings.TrimSpace(c.Query("provider")),
		Query:           firstNonEmpty(c.Query("query"), c.Query("q")),
		IncludeInactive: includeInactive,
		Limit:           limit,
	})
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *MaclawEvaluationHandler) CreateTarget(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	var in maclaw.EvaluationTargetInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if shouldRewriteLocalhostForRuntimeMode(h.runtimeMode) {
		in.BaseURL = normalizeLocalhostURLForDockerRuntime(in.BaseURL)
		if in.Metadata != nil {
			if healthURL := strings.TrimSpace(in.Metadata["health_url"]); healthURL != "" {
				in.Metadata["health_url"] = normalizeLocalhostURLForDockerRuntime(healthURL)
			}
		}
	}
	ensureTargetHealthURL(&in)
	out, err := gateway.SaveEvaluationTarget(c.Request.Context(), in)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func ensureTargetHealthURL(in *maclaw.EvaluationTargetInput) {
	if in == nil || in.Kind != maclaw.EvaluationTargetKindLLM {
		return
	}
	if in.Metadata == nil {
		in.Metadata = map[string]string{}
	}
	if strings.TrimSpace(in.Metadata["health_url"]) != "" {
		return
	}
	baseURL := strings.TrimRight(strings.TrimSpace(in.BaseURL), "/")
	if baseURL == "" {
		return
	}
	in.Metadata["health_url"] = baseURL + "/models"
}

func (h *MaclawEvaluationHandler) GetTarget(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	targetID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("target_id"), c.Param("targetId")))
	if targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target id is required"})
		return
	}
	out, err := gateway.GetEvaluationTarget(c.Request.Context(), targetID)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *MaclawEvaluationHandler) ProbeTarget(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	targetID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("target_id"), c.Param("targetId")))
	if targetID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target id is required"})
		return
	}
	out, err := gateway.ProbeEvaluationTarget(c.Request.Context(), targetID)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *MaclawEvaluationHandler) StartRun(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, instanceID, ok := h.evaluationGatewayWithInstance(c)
	if !ok {
		return
	}
	var in maclaw.EvaluationRunInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if h.resourceProjection != nil && h.provider != nil {
		session, ok := resolveMaclawGatewaySession(c, h.provider)
		if !ok {
			return
		}
		handles, err := h.resourceProjection.ResolveEnterpriseRunResourceHandles(c.Request.Context(), maclawRuntimeIdentity(c), session, in.ResourceHandles)
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw resource projection failed", "code": "maclaw_resource_projection_failed"})
			return
		}
		in.ResourceHandles = handles
	}
	in.InstanceID = instanceID
	out, err := gateway.StartEvaluationRun(c.Request.Context(), in)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusCreated, out)
}

func (h *MaclawEvaluationHandler) ListJobs(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	limit, err := parseOptionalPositiveInt(c.Query("limit"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
		return
	}
	items, err := gateway.ListEvaluationJobs(c.Request.Context(), maclaw.EvaluationJobQuery{
		Kind:   maclaw.EvaluationJobKindRun,
		Status: maclaw.EvaluationJobStatus(strings.TrimSpace(c.Query("status"))),
		Limit:  limit,
	})
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *MaclawEvaluationHandler) GetJob(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	jobID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("job_id"), c.Param("jobId")))
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job id is required"})
		return
	}
	out, err := gateway.GetEvaluationJob(c.Request.Context(), jobID)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	if !isEvaluationRunJob(out) {
		c.JSON(http.StatusNotFound, gin.H{"error": "requested maclaw resource was not found", "code": "maclaw_not_found"})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *MaclawEvaluationHandler) CancelJob(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	jobID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("job_id"), c.Param("jobId")))
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job id is required"})
		return
	}
	out, err := gateway.CancelEvaluationJob(c.Request.Context(), jobID)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	if !isEvaluationRunJob(out) {
		c.JSON(http.StatusNotFound, gin.H{"error": "requested maclaw resource was not found", "code": "maclaw_not_found"})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *MaclawEvaluationHandler) RetryJob(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	jobID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("job_id"), c.Param("jobId")))
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job id is required"})
		return
	}
	out, err := gateway.RetryEvaluationJob(c.Request.Context(), jobID)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	if !isEvaluationRunJob(out) {
		c.JSON(http.StatusNotFound, gin.H{"error": "requested maclaw resource was not found", "code": "maclaw_not_found"})
		return
	}
	c.JSON(http.StatusAccepted, out)
}

func (h *MaclawEvaluationHandler) ResumeJob(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	jobID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("job_id"), c.Param("jobId")))
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job id is required"})
		return
	}
	out, err := gateway.ResumeEvaluationJob(c.Request.Context(), jobID)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	if !isEvaluationRunJob(out) {
		c.JSON(http.StatusNotFound, gin.H{"error": "requested maclaw resource was not found", "code": "maclaw_not_found"})
		return
	}
	c.JSON(http.StatusAccepted, out)
}

func (h *MaclawEvaluationHandler) GetJobRecovery(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	jobID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("job_id"), c.Param("jobId")))
	if jobID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "job id is required"})
		return
	}
	out, err := gateway.GetEvaluationJobRecovery(c.Request.Context(), jobID)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *MaclawEvaluationHandler) GetReport(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	reportID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("report_id"), c.Param("reportId")))
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "report id is required"})
		return
	}
	out, err := gateway.GetEvaluationReport(c.Request.Context(), reportID)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *MaclawEvaluationHandler) ExportReport(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	reportID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("report_id"), c.Param("reportId")))
	if reportID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "report id is required"})
		return
	}
	out, err := gateway.ExportEvaluationReport(c.Request.Context(), reportID, strings.TrimSpace(c.Query("format")))
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	contentType := strings.TrimSpace(out.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	if strings.TrimSpace(out.Filename) != "" {
		c.Header("Content-Disposition", `attachment; filename="`+strings.ReplaceAll(out.Filename, `"`, "")+`"`)
	}
	c.Data(http.StatusOK, contentType, out.Content)
}

func (h *MaclawEvaluationHandler) ListEvidence(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, instanceID, ok := h.evaluationGatewayWithOptionalInstance(c)
	if !ok {
		return
	}
	limit, err := parseOptionalPositiveInt(c.Query("limit"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be a positive integer"})
		return
	}
	items, err := gateway.ListEvaluationEvidence(c.Request.Context(), maclaw.EvaluationEvidenceQuery{
		InstanceID: instanceID,
		SessionID:  strings.TrimSpace(c.Query("session_id")),
		RunID:      strings.TrimSpace(c.Query("run_id")),
		Kind:       maclaw.EvaluationEvidenceKind(strings.TrimSpace(c.Query("kind"))),
		Limit:      limit,
	})
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *MaclawEvaluationHandler) GetEvidence(c *gin.Context) {
	if !requireEnterpriseMaclawExecutionRole(c) {
		return
	}
	gateway, instanceID, ok := h.evaluationGatewayWithOptionalInstance(c)
	if !ok {
		return
	}
	evidenceID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("evidence_id"), c.Param("evidenceId")))
	if evidenceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "evidence id is required"})
		return
	}
	out, err := gateway.GetEvaluationEvidence(c.Request.Context(), evidenceID)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	if instanceID != "" && out.InstanceID != "" && out.InstanceID != instanceID {
		c.JSON(http.StatusNotFound, gin.H{"error": "requested maclaw resource was not found", "code": "maclaw_not_found"})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *MaclawEvaluationHandler) PreviewResource(c *gin.Context) {
	gateway, ok := h.evaluationGateway(c)
	if !ok {
		return
	}
	resourceID := strings.TrimSpace(firstNonEmpty(c.Param("id"), c.Param("resource_id"), c.Param("resourceId")))
	if resourceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource id is required"})
		return
	}
	out, err := gateway.PreviewEvaluationResource(c.Request.Context(), resourceID)
	if err != nil {
		writeMaclawError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func requireEnterpriseMaclawExecutionRole(c *gin.Context) bool {
	role := strings.TrimSpace(c.GetString("user_role"))
	if role == "enterprise" || role == "admin" {
		return true
	}
	c.JSON(http.StatusForbidden, gin.H{
		"error": "maclaw evaluation execution is available to enterprise accounts",
		"code":  "forbidden",
	})
	return false
}

func (h *MaclawEvaluationHandler) available(c *gin.Context) bool {
	if h == nil || h.gateway == nil || !h.gateway.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw runtime is not configured"})
		return false
	}
	return true
}

func (h *MaclawEvaluationHandler) evaluationGateway(c *gin.Context) (maclaw.EvaluationGateway, bool) {
	if h != nil && h.provider != nil {
		session, ok := resolveMaclawGatewaySession(c, h.provider)
		if !ok {
			return nil, false
		}
		return session.Client, true
	}
	if !h.available(c) {
		return nil, false
	}
	return h.gateway, true
}

func (h *MaclawEvaluationHandler) evaluationGatewayWithInstance(c *gin.Context) (maclaw.EvaluationGateway, string, bool) {
	if h != nil && h.provider != nil {
		session, instanceID, ok := h.evaluationSessionWithInstance(c)
		if !ok {
			return nil, "", false
		}
		return session.Client, instanceID, true
	}
	if !h.available(c) {
		return nil, "", false
	}
	instanceID, ok := h.resolveInstance(c)
	if !ok {
		return nil, "", false
	}
	return h.gateway, instanceID, true
}

func (h *MaclawEvaluationHandler) evaluationSessionWithInstance(c *gin.Context) (*maclaw.GatewaySession, string, bool) {
	if h != nil && h.provider != nil {
		session, ok := resolveMaclawGatewaySession(c, h.provider)
		if !ok {
			return nil, "", false
		}
		if strings.TrimSpace(session.InstanceID) == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw instance is not configured for current user"})
			return nil, "", false
		}
		return session, session.InstanceID, true
	}
	if !h.available(c) {
		return nil, "", false
	}
	instanceID, ok := h.resolveInstance(c)
	if !ok {
		return nil, "", false
	}
	return &maclaw.GatewaySession{InstanceID: instanceID}, instanceID, true
}

func (h *MaclawEvaluationHandler) evaluationGatewayWithOptionalInstance(c *gin.Context) (maclaw.EvaluationGateway, string, bool) {
	if h != nil && h.provider != nil {
		session, ok := resolveMaclawGatewaySession(c, h.provider)
		if !ok {
			return nil, "", false
		}
		return session.Client, session.InstanceID, true
	}
	if !h.available(c) {
		return nil, "", false
	}
	instanceID := strings.TrimSpace(c.Query("instance_id"))
	if h.resolver != nil {
		resolved, ok := h.resolveInstance(c)
		if !ok {
			return nil, "", false
		}
		instanceID = resolved
	}
	return h.gateway, instanceID, true
}

func (h *MaclawEvaluationHandler) resolveInstance(c *gin.Context) (string, bool) {
	if h.resolver == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw instance mapping is not configured"})
		return "", false
	}
	instanceID, ok := h.resolver.Resolve(maclaw.RuntimeIdentity{
		UserID: c.GetString("user_id"),
		Role:   c.GetString("user_role"),
	})
	if !ok || strings.TrimSpace(instanceID) == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "maclaw instance is not configured for current user"})
		return "", false
	}
	return instanceID, true
}

func parseOptionalBool(raw string) (bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, nil
	}
	return strconv.ParseBool(raw)
}

func parseOptionalPositiveInt(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return 0, strconv.ErrSyntax
	}
	return v, nil
}

func parseAssessmentTypes(c *gin.Context) []string {
	values := append([]string{}, c.QueryArray("assessment_type")...)
	values = append(values, c.QueryArray("assessment_types")...)
	out := make([]string, 0, len(values))
	for _, raw := range values {
		for _, item := range strings.Split(raw, ",") {
			item = strings.TrimSpace(item)
			if item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}

func isEnterpriseLikeRole(role string) bool {
	role = strings.TrimSpace(role)
	return role == "enterprise" || role == "admin"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isEvaluationRunJob(job *maclaw.EvaluationJob) bool {
	return job != nil && job.Kind == maclaw.EvaluationJobKindRun
}
