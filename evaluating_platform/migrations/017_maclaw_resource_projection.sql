CREATE TABLE IF NOT EXISTS maclaw_resource_publications (
    source_resource_id TEXT PRIMARY KEY,
    source_expert_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_maclaw_tenant_id TEXT NOT NULL,
    source_resource_handle TEXT NOT NULL UNIQUE,
    source_version TEXT NOT NULL DEFAULT 'default',
    name TEXT NOT NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT false,
    summary TEXT NOT NULL DEFAULT '',
    assessment_types JSONB NOT NULL DEFAULT '[]'::jsonb,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_maclaw_resource_publications_expert
    ON maclaw_resource_publications(source_expert_user_id);

CREATE INDEX IF NOT EXISTS idx_maclaw_resource_publications_visible
    ON maclaw_resource_publications(enabled, status, kind);

CREATE TABLE IF NOT EXISTS maclaw_resource_shadows (
    enterprise_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    enterprise_maclaw_tenant_id TEXT NOT NULL,
    source_expert_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_resource_id TEXT NOT NULL REFERENCES maclaw_resource_publications(source_resource_id) ON DELETE CASCADE,
    source_version TEXT NOT NULL DEFAULT 'default',
    shadow_resource_id TEXT NOT NULL,
    shadow_resource_handle TEXT NOT NULL,
    sync_status TEXT NOT NULL DEFAULT 'ready',
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_user_id, source_resource_id, source_version)
);

CREATE INDEX IF NOT EXISTS idx_maclaw_resource_shadows_source
    ON maclaw_resource_shadows(source_resource_id, source_version);
