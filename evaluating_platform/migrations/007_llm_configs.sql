-- ============================================================
-- 编排 LLM 配置表（按用户维度持久化）
-- 被测 LLM 配置表（按用户维度持久化）
-- v1.2.0
-- ============================================================

-- 编排 LLM 配置表
CREATE TABLE IF NOT EXISTS orchestration_llm_configs (
    id              UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    base_url        TEXT           NOT NULL DEFAULT '',
    api_key         TEXT           NOT NULL DEFAULT '',
    model           VARCHAR(255)   NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_orchestration_llm_configs_user_id
    ON orchestration_llm_configs(user_id);

-- updated_at 触发器
DROP TRIGGER IF EXISTS trg_orchestration_llm_configs_updated_at ON orchestration_llm_configs;
CREATE TRIGGER trg_orchestration_llm_configs_updated_at
    BEFORE UPDATE ON orchestration_llm_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- 被测 LLM 配置表
CREATE TABLE IF NOT EXISTS target_llm_configs (
    id              UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id         UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    base_url        TEXT           NOT NULL DEFAULT '',
    api_key         TEXT           NOT NULL DEFAULT '',
    model           VARCHAR(255)   NOT NULL DEFAULT '',
    connector_type  VARCHAR(50)    NOT NULL DEFAULT 'openai',
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_target_llm_configs_user_id
    ON target_llm_configs(user_id);

-- updated_at 触发器
DROP TRIGGER IF EXISTS trg_target_llm_configs_updated_at ON target_llm_configs;
CREATE TRIGGER trg_target_llm_configs_updated_at
    BEFORE UPDATE ON target_llm_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
