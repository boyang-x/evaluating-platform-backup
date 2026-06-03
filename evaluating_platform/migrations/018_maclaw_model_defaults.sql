CREATE TABLE IF NOT EXISTS maclaw_model_defaults (
    id TEXT PRIMARY KEY,
    encrypted_config BYTEA NOT NULL,
    config_key_id TEXT NOT NULL,
    updated_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
