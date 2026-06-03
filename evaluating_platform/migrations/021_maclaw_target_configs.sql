CREATE TABLE IF NOT EXISTS maclaw_target_configs (
    id               UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    encrypted_config BYTEA       NOT NULL,
    config_key_id    TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_maclaw_target_configs_user_id
    ON maclaw_target_configs(user_id);

DROP TRIGGER IF EXISTS trg_maclaw_target_configs_updated_at ON maclaw_target_configs;
CREATE TRIGGER trg_maclaw_target_configs_updated_at
    BEFORE UPDATE ON maclaw_target_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
