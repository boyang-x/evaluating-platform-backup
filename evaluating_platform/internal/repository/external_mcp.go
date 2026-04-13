package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"evaluating_platform/internal/model"
)

type ExternalMCPServerRepository struct {
	pool *pgxpool.Pool
}

type ExternalMCPToolRepository struct {
	pool *pgxpool.Pool
}

func NewExternalMCPServerRepository(pool *pgxpool.Pool) *ExternalMCPServerRepository {
	return &ExternalMCPServerRepository{pool: pool}
}

func NewExternalMCPToolRepository(pool *pgxpool.Pool) *ExternalMCPToolRepository {
	return &ExternalMCPToolRepository{pool: pool}
}

const externalServerSelectCols = `
    id, expert_id, name, namespace, description, base_url, transport_type,
    auth_type, auth_key, auth_header, auth_prefix,
    upstream_base_url, upstream_api_key, upstream_model, upstream_timeout_seconds,
    timeout_seconds,
    enabled, skill_prompt, status, last_sync_at, last_error, created_at, updated_at
`

func scanExternalMCPServer(row interface {
	Scan(dest ...interface{}) error
}) (*model.ExternalMCPServer, error) {
	var item model.ExternalMCPServer
	err := row.Scan(
		&item.ID,
		&item.ExpertID,
		&item.Name,
		&item.Namespace,
		&item.Description,
		&item.BaseURL,
		&item.TransportType,
		&item.AuthType,
		&item.AuthKey,
		&item.AuthHeader,
		&item.AuthPrefix,
		&item.UpstreamBaseURL,
		&item.UpstreamAPIKey,
		&item.UpstreamModel,
		&item.UpstreamTimeout,
		&item.TimeoutSeconds,
		&item.Enabled,
		&item.SkillPrompt,
		&item.Status,
		&item.LastSyncAt,
		&item.LastError,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *ExternalMCPServerRepository) Create(ctx context.Context, item *model.ExternalMCPServer) error {
	query := `
        INSERT INTO external_mcp_servers (
            id, expert_id, name, namespace, description, base_url, transport_type,
            auth_type, auth_key, auth_header, auth_prefix,
            upstream_base_url, upstream_api_key, upstream_model, upstream_timeout_seconds,
            timeout_seconds,
            enabled, skill_prompt, status, last_sync_at, last_error
        ) VALUES (
            $1,$2,$3,$4,$5,$6,$7,
            $8,$9,$10,$11,$12,$13,$14,$15,
            $16,
            $17,$18,$19,$20,$21
        )
        RETURNING created_at, updated_at
    `
	return r.pool.QueryRow(ctx, query,
		item.ID,
		item.ExpertID,
		item.Name,
		item.Namespace,
		item.Description,
		item.BaseURL,
		item.TransportType,
		item.AuthType,
		item.AuthKey,
		item.AuthHeader,
		item.AuthPrefix,
		item.UpstreamBaseURL,
		item.UpstreamAPIKey,
		item.UpstreamModel,
		item.UpstreamTimeout,
		item.TimeoutSeconds,
		item.Enabled,
		item.SkillPrompt,
		item.Status,
		item.LastSyncAt,
		item.LastError,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
}

func (r *ExternalMCPServerRepository) Update(ctx context.Context, item *model.ExternalMCPServer) error {
	query := `
        UPDATE external_mcp_servers
        SET name = $1,
            namespace = $2,
            description = $3,
            base_url = $4,
            transport_type = $5,
            auth_type = $6,
            auth_key = $7,
            auth_header = $8,
            auth_prefix = $9,
            upstream_base_url = $10,
            upstream_api_key = $11,
            upstream_model = $12,
            upstream_timeout_seconds = $13,
            timeout_seconds = $14,
            enabled = $15,
            skill_prompt = $16,
            status = $17,
            last_sync_at = $18,
            last_error = $19,
            updated_at = NOW()
        WHERE id = $20 AND expert_id = $21
        RETURNING created_at, updated_at
    `
	err := r.pool.QueryRow(ctx, query,
		item.Name,
		item.Namespace,
		item.Description,
		item.BaseURL,
		item.TransportType,
		item.AuthType,
		item.AuthKey,
		item.AuthHeader,
		item.AuthPrefix,
		item.UpstreamBaseURL,
		item.UpstreamAPIKey,
		item.UpstreamModel,
		item.UpstreamTimeout,
		item.TimeoutSeconds,
		item.Enabled,
		item.SkillPrompt,
		item.Status,
		item.LastSyncAt,
		item.LastError,
		item.ID,
		item.ExpertID,
	).Scan(&item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return fmt.Errorf("external mcp server not found or access denied")
		}
		return fmt.Errorf("update external mcp server: %w", err)
	}
	return nil
}

func (r *ExternalMCPServerRepository) Delete(ctx context.Context, id, expertID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM external_mcp_servers WHERE id = $1 AND expert_id = $2`, id, expertID)
	if err != nil {
		return fmt.Errorf("delete external mcp server: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("external mcp server not found or access denied")
	}
	return nil
}

func (r *ExternalMCPServerRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ExternalMCPServer, error) {
	item, err := scanExternalMCPServer(r.pool.QueryRow(ctx, `SELECT `+externalServerSelectCols+` FROM external_mcp_servers WHERE id = $1`, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query external mcp server: %w", err)
	}
	return item, nil
}

func (r *ExternalMCPServerRepository) GetByIDForExpert(ctx context.Context, id, expertID uuid.UUID) (*model.ExternalMCPServer, error) {
	item, err := scanExternalMCPServer(r.pool.QueryRow(ctx, `SELECT `+externalServerSelectCols+` FROM external_mcp_servers WHERE id = $1 AND expert_id = $2`, id, expertID))
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query external mcp server: %w", err)
	}
	return item, nil
}

func (r *ExternalMCPServerRepository) ListByExpert(ctx context.Context, expertID uuid.UUID) ([]model.ExternalMCPServer, error) {
	query := `
        SELECT ` + externalServerSelectCols + `,
               COALESCE((SELECT COUNT(*) FROM external_mcp_tools t WHERE t.server_id = s.id AND t.enabled = TRUE), 0) AS tool_count
        FROM external_mcp_servers s
        WHERE expert_id = $1
        ORDER BY updated_at DESC
    `
	rows, err := r.pool.Query(ctx, query, expertID)
	if err != nil {
		return nil, fmt.Errorf("list external mcp servers: %w", err)
	}
	defer rows.Close()

	items := make([]model.ExternalMCPServer, 0)
	for rows.Next() {
		var item model.ExternalMCPServer
		err := rows.Scan(
			&item.ID,
			&item.ExpertID,
			&item.Name,
			&item.Namespace,
			&item.Description,
			&item.BaseURL,
			&item.TransportType,
			&item.AuthType,
			&item.AuthKey,
			&item.AuthHeader,
			&item.AuthPrefix,
			&item.UpstreamBaseURL,
			&item.UpstreamAPIKey,
			&item.UpstreamModel,
			&item.UpstreamTimeout,
			&item.TimeoutSeconds,
			&item.Enabled,
			&item.SkillPrompt,
			&item.Status,
			&item.LastSyncAt,
			&item.LastError,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ToolCount,
		)
		if err != nil {
			return nil, fmt.Errorf("scan external mcp server: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ExternalMCPServerRepository) ListEnabled(ctx context.Context) ([]model.ExternalMCPServer, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+externalServerSelectCols+` FROM external_mcp_servers WHERE enabled = TRUE ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list enabled external mcp servers: %w", err)
	}
	defer rows.Close()

	items := make([]model.ExternalMCPServer, 0)
	for rows.Next() {
		item, err := scanExternalMCPServer(rows)
		if err != nil {
			return nil, fmt.Errorf("scan enabled external mcp server: %w", err)
		}
		items = append(items, *item)
	}
	return items, nil
}

func (r *ExternalMCPServerRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status, lastError string, lastSyncAt *time.Time) error {
	_, err := r.pool.Exec(ctx, `
        UPDATE external_mcp_servers
        SET status = $1,
            last_error = $2,
            last_sync_at = $3,
            updated_at = NOW()
        WHERE id = $4
    `, status, lastError, lastSyncAt, id)
	if err != nil {
		return fmt.Errorf("update external mcp server status: %w", err)
	}
	return nil
}

func (r *ExternalMCPToolRepository) ReplaceByServer(ctx context.Context, serverID uuid.UUID, tools []model.ExternalMCPTool) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replace external mcp tools: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	_, err = tx.Exec(ctx, `DELETE FROM external_mcp_tools WHERE server_id = $1`, serverID)
	if err != nil {
		return fmt.Errorf("clear external mcp tools: %w", err)
	}

	query := `
        INSERT INTO external_mcp_tools (
            id, server_id, remote_tool_name, proxy_tool_name, description,
            input_schema, capability_metadata, skill_text, enabled
        ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
    `
	for _, tool := range tools {
		if _, err = tx.Exec(ctx, query,
			tool.ID,
			serverID,
			tool.RemoteToolName,
			tool.ProxyToolName,
			tool.Description,
			tool.InputSchema,
			tool.CapabilityMetadata,
			tool.SkillText,
			tool.Enabled,
		); err != nil {
			return fmt.Errorf("insert external mcp tool: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replace external mcp tools: %w", err)
	}
	return nil
}

func (r *ExternalMCPToolRepository) ListByServer(ctx context.Context, serverID uuid.UUID) ([]model.ExternalMCPTool, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, server_id, remote_tool_name, proxy_tool_name, description,
               input_schema, capability_metadata, skill_text, enabled, created_at, updated_at
        FROM external_mcp_tools
        WHERE server_id = $1
        ORDER BY proxy_tool_name ASC
    `, serverID)
	if err != nil {
		return nil, fmt.Errorf("list external mcp tools: %w", err)
	}
	defer rows.Close()

	items := make([]model.ExternalMCPTool, 0)
	for rows.Next() {
		var item model.ExternalMCPTool
		err := rows.Scan(
			&item.ID,
			&item.ServerID,
			&item.RemoteToolName,
			&item.ProxyToolName,
			&item.Description,
			&item.InputSchema,
			&item.CapabilityMetadata,
			&item.SkillText,
			&item.Enabled,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan external mcp tool: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ExternalMCPToolRepository) ListEnabledByServer(ctx context.Context, serverID uuid.UUID) ([]model.ExternalMCPTool, error) {
	rows, err := r.pool.Query(ctx, `
        SELECT id, server_id, remote_tool_name, proxy_tool_name, description,
               input_schema, capability_metadata, skill_text, enabled, created_at, updated_at
        FROM external_mcp_tools
        WHERE server_id = $1 AND enabled = TRUE
        ORDER BY proxy_tool_name ASC
    `, serverID)
	if err != nil {
		return nil, fmt.Errorf("list enabled external mcp tools: %w", err)
	}
	defer rows.Close()

	items := make([]model.ExternalMCPTool, 0)
	for rows.Next() {
		var item model.ExternalMCPTool
		err := rows.Scan(
			&item.ID,
			&item.ServerID,
			&item.RemoteToolName,
			&item.ProxyToolName,
			&item.Description,
			&item.InputSchema,
			&item.CapabilityMetadata,
			&item.SkillText,
			&item.Enabled,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan enabled external mcp tool: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *ExternalMCPToolRepository) DeleteByServer(ctx context.Context, serverID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM external_mcp_tools WHERE server_id = $1`, serverID)
	if err != nil {
		return fmt.Errorf("delete external mcp tools by server: %w", err)
	}
	return nil
}
