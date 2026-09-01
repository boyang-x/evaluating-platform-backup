CREATE TABLE IF NOT EXISTS maclaw_skill_publications (
    source_expert_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_maclaw_tenant_id TEXT NOT NULL DEFAULT '',
    source_skill_name TEXT NOT NULL,
    source_version TEXT NOT NULL DEFAULT 'default',
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    enabled BOOLEAN NOT NULL DEFAULT true,
    triggers JSONB NOT NULL DEFAULT '[]'::jsonb,
    tags JSONB NOT NULL DEFAULT '[]'::jsonb,
    assessment_types JSONB NOT NULL DEFAULT '[]'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (source_expert_user_id, source_skill_name, source_version)
);

CREATE INDEX IF NOT EXISTS idx_maclaw_skill_publications_visible
    ON maclaw_skill_publications(enabled, status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_maclaw_skill_publications_expert
    ON maclaw_skill_publications(source_expert_user_id);

CREATE TABLE IF NOT EXISTS maclaw_skill_shadows (
    enterprise_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    enterprise_maclaw_tenant_id TEXT NOT NULL DEFAULT '',
    source_expert_user_id UUID NOT NULL,
    source_skill_name TEXT NOT NULL,
    source_version TEXT NOT NULL DEFAULT 'default',
    shadow_skill_name TEXT NOT NULL,
    sync_status TEXT NOT NULL DEFAULT 'ready',
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (enterprise_user_id, source_expert_user_id, source_skill_name, source_version),
    FOREIGN KEY (source_expert_user_id, source_skill_name, source_version)
        REFERENCES maclaw_skill_publications(source_expert_user_id, source_skill_name, source_version)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_maclaw_skill_shadows_source
    ON maclaw_skill_shadows(source_expert_user_id, source_skill_name, source_version);
