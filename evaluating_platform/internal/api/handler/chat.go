package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	chatpkg "evaluating_platform/internal/chat"
	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
)

// ChatHandler 对话会话处理器
type ChatHandler struct {
	manager  *chatpkg.Manager
	chatRepo *repository.ChatRepository
}

func NewChatHandler(manager *chatpkg.Manager, chatRepo *repository.ChatRepository) *ChatHandler {
	return &ChatHandler{manager: manager, chatRepo: chatRepo}
}

// CreateSession POST /api/v1/chat/sessions
func (h *ChatHandler) CreateSession(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))

	session := &model.ChatSession{
		ID:         uuid.New(),
		UserID:     userID,
		Title:      "新对话",
		State:      model.ChatStateIdle,
		TargetInfo: map[string]any{},
		PlanInfo:   map[string]any{},
	}
	if err := h.chatRepo.CreateSession(c.Request.Context(), session); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建会话失败"})
		return
	}
	c.JSON(http.StatusCreated, session)
}

// ListSessions GET /api/v1/chat/sessions
func (h *ChatHandler) ListSessions(c *gin.Context) {
	userID, _ := uuid.Parse(c.GetString("user_id"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	sessions, total, err := h.chatRepo.ListSessions(c.Request.Context(), userID, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": sessions, "total": total})
}

// GetSession GET /api/v1/chat/sessions/:id
func (h *ChatHandler) GetSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	session, err := h.chatRepo.GetSession(c.Request.Context(), id)
	if err != nil || session.UserID != userID {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}

	messages, _ := h.chatRepo.ListMessages(c.Request.Context(), id)
	c.JSON(http.StatusOK, gin.H{"session": session, "messages": messages})
}

// SendMessage POST /api/v1/chat/sessions/:id/messages
func (h *ChatHandler) SendMessage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var body struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	aiMsg, err := h.manager.SendMessage(c.Request.Context(), id, userID, body.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, aiMsg)
}

// ConfirmPlan POST /api/v1/chat/sessions/:id/confirm
// 用户点击"确认执行"按钮后调用，返回 assessment_id 供前端跳转
func (h *ChatHandler) ConfirmPlan(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	var body struct {
		TestCount  int `json:"test_count"`
		TestRounds int `json:"test_rounds"`
	}
	if rawBody, readErr := io.ReadAll(c.Request.Body); readErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	} else if len(rawBody) > 0 {
		if err := json.Unmarshal(rawBody, &body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
			return
		}
	}

	testCount := body.TestCount
	if testCount <= 0 {
		testCount = body.TestRounds
	}

	session, err := h.manager.ConfirmPlan(c.Request.Context(), id, userID, testCount)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	resp := gin.H{
		"session": session,
		"plan":    session.PlanInfo,
	}
	if session.AssessmentID != nil {
		resp["assessment_id"] = session.AssessmentID.String()
	}
	c.JSON(http.StatusOK, resp)
}

// DeleteSession DELETE /api/v1/chat/sessions/:id
func (h *ChatHandler) DeleteSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	userID, _ := uuid.Parse(c.GetString("user_id"))

	if err := h.chatRepo.DeleteSession(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "会话不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}
