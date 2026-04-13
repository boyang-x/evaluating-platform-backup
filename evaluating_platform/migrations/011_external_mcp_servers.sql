CREATE TABLE IF NOT EXISTS external_mcp_servers (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    expert_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name              VARCHAR(255) NOT NULL,
    namespace         VARCHAR(120) NOT NULL,
    description       TEXT NOT NULL DEFAULT '',
    base_url          TEXT NOT NULL,
    transport_type    VARCHAR(50) NOT NULL DEFAULT 'sse',
    auth_type         VARCHAR(50) NOT NULL DEFAULT 'none',
    auth_key          TEXT NOT NULL DEFAULT '',
    auth_header       VARCHAR(255) NOT NULL DEFAULT 'Authorization',
    auth_prefix       VARCHAR(120) NOT NULL DEFAULT 'Bearer',
    timeout_seconds   INTEGER NOT NULL DEFAULT 30,
    enabled           BOOLEAN NOT NULL DEFAULT TRUE,
    skill_prompt      TEXT NOT NULL DEFAULT '',
    status            VARCHAR(50) NOT NULL DEFAULT 'disconnected',
    last_sync_at      TIMESTAMPTZ,
    last_error        TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_external_mcp_servers_namespace UNIQUE (namespace)
);

CREATE INDEX IF NOT EXISTS idx_external_mcp_servers_expert_id
    ON external_mcp_servers(expert_id);

DROP TRIGGER IF EXISTS trg_external_mcp_servers_updated_at ON external_mcp_servers;
CREATE TRIGGER trg_external_mcp_servers_updated_at
    BEFORE UPDATE ON external_mcp_servers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE IF NOT EXISTS external_mcp_tools (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id            UUID NOT NULL REFERENCES external_mcp_servers(id) ON DELETE CASCADE,
    remote_tool_name     VARCHAR(255) NOT NULL,
    proxy_tool_name      VARCHAR(255) NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    input_schema         JSONB NOT NULL DEFAULT '{}'::jsonb,
    capability_metadata  JSONB NOT NULL DEFAULT '{}'::jsonb,
    skill_text           TEXT NOT NULL DEFAULT '',
    enabled              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_external_mcp_tools_server_remote UNIQUE (server_id, remote_tool_name),
    CONSTRAINT uq_external_mcp_tools_proxy UNIQUE (proxy_tool_name)
);

CREATE INDEX IF NOT EXISTS idx_external_mcp_tools_server_id
    ON external_mcp_tools(server_id);

DROP TRIGGER IF EXISTS trg_external_mcp_tools_updated_at ON external_mcp_tools;
CREATE TRIGGER trg_external_mcp_tools_updated_at
    BEFORE UPDATE ON external_mcp_tools
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
