package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	jwtSecret string
	userRepo  *repository.UserRepository
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(jwtSecret string, userRepo *repository.UserRepository) *AuthHandler {
	return &AuthHandler{
		jwtSecret: jwtSecret,
		userRepo:  userRepo,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Email    string         `json:"email" binding:"required,email"`
	Password string         `json:"password" binding:"required,min=8"`
	Name     string         `json:"name" binding:"required"`
	Role     model.UserRole `json:"role" binding:"required"`
	OrgName  string         `json:"org_name"`
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证用户密码
	user, err := h.userRepo.VerifyPassword(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "邮箱或密码错误"})
		return
	}

	// 生成 JWT
	token, err := h.generateToken(user.ID.String(), string(user.Role))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token generation failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"id":       user.ID,
			"email":    user.Email,
			"name":     user.Name,
			"role":     user.Role,
			"org_name": user.OrgName,
			"balance":  user.Balance,
		},
	})
}

// Register 用户注册
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建用户
	user := &model.User{
		ID:      uuid.New(),
		Email:   req.Email,
		Name:    req.Name,
		Role:    req.Role,
		OrgName: req.OrgName,
		Balance: 0.0,
	}
	if err := h.userRepo.Create(c.Request.Context(), user, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "注册失败，邮箱可能已存在"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "注册成功",
		"user_id": user.ID,
	})
}

// Me 返回当前登录用户的最新信息
func (h *AuthHandler) Me(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	user, err := h.userRepo.GetByID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"email":    user.Email,
		"name":     user.Name,
		"role":     user.Role,
		"org_name": user.OrgName,
		"balance":  user.Balance,
	})
}

// generateToken 生成 JWT
func (h *AuthHandler) generateToken(userID, role string) (string, error) {
	claims := jwt.MapClaims{
		"sub":  userID,
		"role": role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}
