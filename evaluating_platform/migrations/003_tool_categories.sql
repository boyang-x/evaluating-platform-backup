-- ============================================================
-- 工具三分类改造：攻击样本 + 模版 + 应用调用评测
-- v1.1.0
-- ============================================================

-- ============================================================
-- 攻击样本表（加密存储，DB 只存元数据）
-- ============================================================
CREATE TABLE IF NOT EXISTS attack_samples (
    id              UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    expert_id       UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sub_type        VARCHAR(50)    NOT NULL
                    CHECK (sub_type IN ('direct_injection','malicious_instruction','compliance_detection','malicious_poisoning')),
    name            VARCHAR(255)   NOT NULL,
    description     TEXT           NOT NULL DEFAULT '',
    encrypted_path  TEXT           NOT NULL DEFAULT '',
    file_hash       VARCHAR(128)   NOT NULL DEFAULT '',
    encryption_key_id VARCHAR(64)  NOT NULL DEFAULT '',
    sample_count    INT            NOT NULL DEFAULT 0,
    field_names     TEXT[]         DEFAULT '{}',
    file_size       BIGINT         NOT NULL DEFAULT 0,
    status          VARCHAR(20)    NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','testing','published','deprecated')),
    visibility      VARCHAR(20)    NOT NULL DEFAULT 'private'
                    CHECK (visibility IN ('private','org','public')),
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_attack_samples_expert_id ON attack_samples(expert_id);
CREATE INDEX IF NOT EXISTS idx_attack_samples_sub_type  ON attack_samples(sub_type);
CREATE INDEX IF NOT EXISTS idx_attack_samples_status    ON attack_samples(status);

-- ============================================================
-- 模版表
-- ============================================================
CREATE TABLE IF NOT EXISTS templates (
    id              UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    expert_id       UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sub_type        VARCHAR(50)    NOT NULL
                    CHECK (sub_type IN ('role_play','multilingual','encoding_evasion')),
    name            VARCHAR(255)   NOT NULL,
    description     TEXT           NOT NULL DEFAULT '',
    content         TEXT           NOT NULL DEFAULT '',
    variables       JSONB          NOT NULL DEFAULT '[]',
    status          VARCHAR(20)    NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','testing','published','deprecated')),
    visibility      VARCHAR(20)    NOT NULL DEFAULT 'private'
                    CHECK (visibility IN ('private','org','public')),
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_templates_expert_id ON templates(expert_id);
CREATE INDEX IF NOT EXISTS idx_templates_sub_type  ON templates(sub_type);
CREATE INDEX IF NOT EXISTS idx_templates_status    ON templates(status);

-- ============================================================
-- 评测包表（加密打包的 .espkg）
-- ============================================================
CREATE TABLE IF NOT EXISTS eval_packages (
    id              UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    expert_id       UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(255)   NOT NULL,
    description     TEXT           NOT NULL DEFAULT '',
    encrypted_path  TEXT           NOT NULL DEFAULT '',
    manifest        JSONB          NOT NULL DEFAULT '{}',
    file_hash       VARCHAR(128)   NOT NULL DEFAULT '',
    encryption_key_id VARCHAR(64)  NOT NULL DEFAULT '',
    status          VARCHAR(20)    NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft','testing','published','deprecated')),
    visibility      VARCHAR(20)    NOT NULL DEFAULT 'private'
                    CHECK (visibility IN ('private','org','public')),
    created_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_eval_packages_expert_id ON eval_packages(expert_id);
CREATE INDEX IF NOT EXISTS idx_eval_packages_status    ON eval_packages(status);

-- ============================================================
-- assets 表新增 tool_category 字段
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'assets' AND column_name = 'tool_category'
    ) THEN
        ALTER TABLE assets ADD COLUMN tool_category VARCHAR(30) NOT NULL DEFAULT '';
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_assets_tool_category ON assets(tool_category);

-- ============================================================
-- updated_at 触发器
-- ============================================================
DROP TRIGGER IF EXISTS trg_attack_samples_updated_at ON attack_samples;
CREATE TRIGGER trg_attack_samples_updated_at
    BEFORE UPDATE ON attack_samples
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS trg_templates_updated_at ON templates;
CREATE TRIGGER trg_templates_updated_at
    BEFORE UPDATE ON templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS trg_eval_packages_updated_at ON eval_packages;
CREATE TRIGGER trg_eval_packages_updated_at
    BEFORE UPDATE ON eval_packages
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
