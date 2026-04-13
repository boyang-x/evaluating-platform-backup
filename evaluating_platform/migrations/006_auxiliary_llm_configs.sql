-- ============================================================
-- 辅助 LLM 配置表（按用户维度持久化）
-- v1.1.0
-- ============================================================
CREATE TABLE IF NOT EXISTS auxiliary_llm_configs (
    id              UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    base_url        TEXT           NOT NULL DEFAULT '',
    api_key         TEXT           NOT NULL DEFAULT '',
    model           VARCHAR(255)   NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_auxiliary_llm_configs_user_id ON auxiliary_llm_configs(user_id);

-- updated_at 触发器
DROP TRIGGER IF EXISTS trg_auxiliary_llm_configs_updated_at ON auxiliary_llm_configs;
CREATE TRIGGER trg_auxiliary_llm_configs_updated_at
    BEFORE UPDATE ON auxiliary_llm_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
