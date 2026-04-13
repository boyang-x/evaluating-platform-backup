package model

import (
	"time"

	"github.com/google/uuid"
)

// UserRole 用户角色
type UserRole string

const (
	RoleEnterprise UserRole = "enterprise" // 企业/政府客户
	RoleExpert     UserRole = "expert"     // 安全专家
	RoleAdmin      UserRole = "admin"      // 平台管理员
)

// User 用户
type User struct {
	ID           uuid.UUID `json:"id" db:"id"`
	Email        string    `json:"email" db:"email"`
	PasswordHash string    `json:"-" db:"password_hash"`
	Name         string    `json:"name" db:"name"`
	Role         UserRole  `json:"role" db:"role"`
	OrgName      string    `json:"org_name" db:"org_name"`
	Balance      float64   `json:"balance" db:"balance"` // 预付费余额
	IsActive     bool      `json:"is_active" db:"is_active"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}
