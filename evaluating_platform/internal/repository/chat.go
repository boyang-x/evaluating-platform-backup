package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

// ChatRepository 对话数据访问层
type ChatRepository struct {
	pool *pgxpool.Pool
}

func NewChatRepository(pool *pgxpool.Pool) *ChatRepository {
	return &ChatRepository{pool: pool}
}

// ─── Session ─────────────────────────────────────────────────────────────────

func (r *ChatRepository) CreateSession(ctx context.Context, s *model.ChatSession) error {
	targetJSON, _ := json.Marshal(s.TargetInfo)
	planJSON, _ := json.Marshal(s.PlanInfo)
	err := r.pool.QueryRow(ctx, `
		INSERT INTO chat_sessions (id, user_id, title, state, target_info, plan_info)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`,
		s.ID, s.UserID, s.Title, s.State, targetJSON, planJSON).
		Scan(&s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert chat session: %w", err)
	}
	return nil
}

func (r *ChatRepository) GetSession(ctx context.Context, id uuid.UUID) (*model.ChatSession, error) {
	var s model.ChatSession
	var targetJSON, planJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, user_id, title, state, assessment_id, target_info, plan_info, created_at, updated_at
		FROM chat_sessions WHERE id = $1`, id).Scan(
		&s.ID, &s.UserID, &s.Title, &s.State, &s.AssessmentID,
		&targetJSON, &planJSON, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get chat session: %w", err)
	}
	json.Unmarshal(targetJSON, &s.TargetInfo)
	json.Unmarshal(planJSON, &s.PlanInfo)
	return &s, nil
}

func (r *ChatRepository) ListSessions(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*model.ChatSession, int, error) {
	var total int
	r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM chat_sessions WHERE user_id = $1", userID).Scan(&total)

	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, title, state, assessment_id, target_info, plan_info, created_at, updated_at
		FROM chat_sessions WHERE user_id = $1
		ORDER BY updated_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list chat sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*model.ChatSession
	for rows.Next() {
		var s model.ChatSession
		var targetJSON, planJSON []byte
		if err := rows.Scan(&s.ID, &s.UserID, &s.Title, &s.State, &s.AssessmentID,
			&targetJSON, &planJSON, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, err
		}
		json.Unmarshal(targetJSON, &s.TargetInfo)
		json.Unmarshal(planJSON, &s.PlanInfo)
		sessions = append(sessions, &s)
	}
	return sessions, total, nil
}

func (r *ChatRepository) UpdateSession(ctx context.Context, s *model.ChatSession) error {
	targetJSON, _ := json.Marshal(s.TargetInfo)
	planJSON, _ := json.Marshal(s.PlanInfo)
	_, err := r.pool.Exec(ctx, `
		UPDATE chat_sessions
		SET title=$1, state=$2, assessment_id=$3, target_info=$4, plan_info=$5, updated_at=NOW()
		WHERE id=$6`,
		s.Title, s.State, s.AssessmentID, targetJSON, planJSON, s.ID)
	if err != nil {
		return fmt.Errorf("update chat session: %w", err)
	}
	return nil
}

func (r *ChatRepository) DeleteSession(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		"DELETE FROM chat_sessions WHERE id=$1 AND user_id=$2", id, userID)
	if err != nil {
		return fmt.Errorf("delete chat session: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("session not found")
	}
	return nil
}

// ─── Message ─────────────────────────────────────────────────────────────────

func (r *ChatRepository) CreateMessage(ctx context.Context, m *model.ChatMessage) error {
	metaJSON, _ := json.Marshal(m.Metadata)
	err := r.pool.QueryRow(ctx, `
		INSERT INTO chat_messages (id, session_id, role, content, metadata)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at`,
		m.ID, m.SessionID, m.Role, m.Content, metaJSON).
		Scan(&m.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert chat message: %w", err)
	}
	return nil
}

func (r *ChatRepository) ListMessages(ctx context.Context, sessionID uuid.UUID) ([]*model.ChatMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, session_id, role, content, metadata, created_at
		FROM chat_messages WHERE session_id = $1
		ORDER BY created_at ASC`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}
	defer rows.Close()

	var msgs []*model.ChatMessage
	for rows.Next() {
		var m model.ChatMessage
		var metaJSON []byte
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &metaJSON, &m.CreatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal(metaJSON, &m.Metadata)
		msgs = append(msgs, &m)
	}
	return msgs, nil
}
