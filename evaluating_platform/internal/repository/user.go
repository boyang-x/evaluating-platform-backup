package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"evaluating_platform/internal/model"
)

// UserRepository 用户数据访问层
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository 创建用户仓库
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create 创建用户
func (r *UserRepository) Create(ctx context.Context, user *model.User, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	query := `
		INSERT INTO users (id, email, password_hash, name, role, org_name, balance)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = r.pool.Exec(ctx, query,
		user.ID, user.Email, string(hash), user.Name, user.Role, user.OrgName, user.Balance)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetByEmail 根据邮箱查询用户
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, org_name, balance, is_active, created_at, updated_at
		FROM users WHERE email = $1 AND is_active = true
	`
	var user model.User
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role,
		&user.OrgName, &user.Balance, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &user, nil
}

// GetByID 根据 ID 查询用户
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, email, password_hash, name, role, org_name, balance, is_active, created_at, updated_at
		FROM users WHERE id = $1
	`
	var user model.User
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role,
		&user.OrgName, &user.Balance, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("query user: %w", err)
	}
	return &user, nil
}

// VerifyPassword 验证密码
func (r *UserRepository) VerifyPassword(ctx context.Context, email, password string) (*model.User, error) {
	user, err := r.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid password")
	}
	return user, nil
}

// UpdateBalance 更新余额（事务）
func (r *UserRepository) UpdateBalance(ctx context.Context, userID uuid.UUID, delta float64, description string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 查询当前余额
	var balanceBefore float64
	err = tx.QueryRow(ctx, "SELECT balance FROM users WHERE id = $1 FOR UPDATE", userID).Scan(&balanceBefore)
	if err != nil {
		return fmt.Errorf("query balance: %w", err)
	}

	balanceAfter := balanceBefore + delta
	if balanceAfter < 0 {
		return fmt.Errorf("insufficient balance")
	}

	// 更新余额
	_, err = tx.Exec(ctx, "UPDATE users SET balance = $1 WHERE id = $2", balanceAfter, userID)
	if err != nil {
		return fmt.Errorf("update balance: %w", err)
	}

	// 记录变动
	transType := "deduct"
	if delta > 0 {
		transType = "recharge"
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO balance_transactions (user_id, type, amount, balance_before, balance_after, description)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, userID, transType, delta, balanceBefore, balanceAfter, description)
	if err != nil {
		return fmt.Errorf("insert transaction: %w", err)
	}

	return tx.Commit(ctx)
}

// ─── Admin 专用方法 ───────────────────────────────────────────────────────────

// ListAll 分页查询所有用户，支持按 role 和关键词过滤
func (r *UserRepository) ListAll(ctx context.Context, limit, offset int, role, keyword string) ([]*model.User, int, error) {
	conds := []string{"email NOT LIKE 'deleted-%@deleted.local'"}
	args := []interface{}{}

	if role != "" {
		args = append(args, role)
		conds = append(conds, fmt.Sprintf("role = $%d", len(args)))
	}
	if keyword != "" {
		args = append(args, "%"+keyword+"%")
		conds = append(conds, fmt.Sprintf("(email ILIKE $%d OR name ILIKE $%d)", len(args), len(args)))
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM users "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	limitN := len(args) + 1
	offsetN := len(args) + 2
	listArgs := append(args, limit, offset)

	query := fmt.Sprintf(`
		SELECT id, email, password_hash, name, role, org_name, balance, is_active, created_at, updated_at
		FROM users %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, limitN, offsetN)

	rows, err := r.pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	users := make([]*model.User, 0)
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.PasswordHash, &u.Name, &u.Role,
			&u.OrgName, &u.Balance, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, &u)
	}
	return users, total, nil
}

// UpdateRole 更新用户角色
func (r *UserRepository) UpdateRole(ctx context.Context, id uuid.UUID, role model.UserRole) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2",
		role, id,
	)
	if err != nil {
		return fmt.Errorf("update role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// SetActive 启用或禁用用户
func (r *UserRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	tag, err := r.pool.Exec(ctx,
		"UPDATE users SET is_active = $1, updated_at = NOW() WHERE id = $2",
		active, id,
	)
	if err != nil {
		return fmt.Errorf("set active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

// Delete removes a platform user from the active management surface. Most
// platform-owned data cascades from users(id); legacy billing and audit rows
// are cleaned first because those historical tables intentionally predate
// cascade semantics.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete user: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, "DELETE FROM billing_records WHERE user_id = $1 OR expert_id = $1", id); err != nil {
		return fmt.Errorf("delete user billing records: %w", err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM balance_transactions WHERE user_id = $1", id); err != nil {
		return fmt.Errorf("delete user balance transactions: %w", err)
	}
	if _, err := tx.Exec(ctx, "DELETE FROM audit_logs WHERE user_id = $1", id); err != nil {
		return fmt.Errorf("delete user audit logs: %w", err)
	}
	tag, err := tx.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return tx.Commit(ctx)
}

// CountByRole counts users grouped by role.
func (r *UserRepository) CountByRole(ctx context.Context) (map[string]int, error) {
	rows, err := r.pool.Query(ctx, "SELECT role, COUNT(*) FROM users WHERE email NOT LIKE 'deleted-%@deleted.local' GROUP BY role")
	if err != nil {
		return nil, fmt.Errorf("count by role: %w", err)
	}
	defer rows.Close()

	result := map[string]int{}
	for rows.Next() {
		var role string
		var count int
		if err := rows.Scan(&role, &count); err != nil {
			return nil, err
		}
		result[role] = count
	}
	return result, nil
}
