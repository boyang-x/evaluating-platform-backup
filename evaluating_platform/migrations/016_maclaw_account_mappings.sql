CREATE TABLE IF NOT EXISTS maclaw_account_mappings (
    platform_user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    platform_role TEXT NOT NULL,
    platform_email TEXT NOT NULL,
    platform_org_name TEXT NOT NULL DEFAULT '',
    maclaw_tenant_id TEXT NOT NULL,
    maclaw_user_id TEXT NOT NULL,
    maclaw_credential_id TEXT NOT NULL,
    maclaw_instance_id TEXT NOT NULL,
    encrypted_api_key BYTEA NOT NULL,
    encrypted_api_secret BYTEA NOT NULL,
    credential_key_id TEXT NOT NULL,
    encrypted_access_token BYTEA,
    access_token_key_id TEXT,
    access_token_expires_at TIMESTAMPTZ,
    provisioning_status TEXT NOT NULL DEFAULT 'ready',
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_maclaw_account_mappings_platform_user
    ON maclaw_account_mappings(platform_user_id);

CREATE INDEX IF NOT EXISTS idx_maclaw_account_mappings_maclaw_tenant
    ON maclaw_account_mappings(maclaw_tenant_id);
