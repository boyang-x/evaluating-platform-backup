package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
)

// AdminHandler exposes administrator-only user management and platform stats APIs.
type AdminHandler struct {
	userRepo    *repository.UserRepository
	billingRepo *repository.BillingRepository
}

func NewAdminHandler(
	userRepo *repository.UserRepository,
	billingRepo *repository.BillingRepository,
) *AdminHandler {
	return &AdminHandler{
		userRepo:    userRepo,
		billingRepo: billingRepo,
	}
}

// ListUsers GET /admin/users?limit&offset&role&keyword
func (h *AdminHandler) ListUsers(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	role := c.Query("role")
	keyword := c.Query("keyword")
	if limit > 100 {
		limit = 100
	}

	users, total, err := h.userRepo.ListAll(c.Request.Context(), limit, offset, role, keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  users,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// UpdateUserRole PUT /admin/users/:id/role
func (h *AdminHandler) UpdateUserRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var body struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	role := model.UserRole(body.Role)
	if role != model.RoleEnterprise && role != model.RoleExpert && role != model.RoleAdmin {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的角色"})
		return
	}

	if err := h.userRepo.UpdateRole(c.Request.Context(), id, role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新角色失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "角色已更新"})
}

// SetUserActive PUT /admin/users/:id/active
func (h *AdminHandler) SetUserActive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.userRepo.SetActive(c.Request.Context(), id, body.Active); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "操作失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "用户状态已更新"})
}

// DeleteUser DELETE /admin/users/:id
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	if currentID, err := uuid.Parse(c.GetString("user_id")); err == nil && currentID == id {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete current admin user"})
		return
	}
	if err := h.userRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete user failed: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "user retired"})
}

// GetStats GET /admin/stats
func (h *AdminHandler) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	roleCounts, err := h.userRepo.CountByRole(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "统计失败"})
		return
	}

	totalRevenue, err := h.billingRepo.SumTotalRevenue(ctx)
	if err != nil {
		totalRevenue = 0
	}

	totalUsers := 0
	for _, n := range roleCounts {
		totalUsers += n
	}

	c.JSON(http.StatusOK, gin.H{
		"total_users":        totalUsers,
		"users_by_role":      roleCounts,
		"total_assessments":  0,
		"total_revenue":      totalRevenue,
		"weekly_assessments": []int{},
	})
}
