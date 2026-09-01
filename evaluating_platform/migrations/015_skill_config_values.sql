CREATE TABLE IF NOT EXISTS skill_config_values (
    skill_id           UUID        NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    field_key          VARCHAR(120) NOT NULL,
    value_ciphertext   BYTEA       NOT NULL,
    key_id             VARCHAR(64) NOT NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (skill_id, field_key)
);

CREATE INDEX IF NOT EXISTS idx_skill_config_values_skill_id
    ON skill_config_values(skill_id);

DROP TRIGGER IF EXISTS trg_skill_config_values_updated_at ON skill_config_values;
CREATE TRIGGER trg_skill_config_values_updated_at
    BEFORE UPDATE ON skill_config_values
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
