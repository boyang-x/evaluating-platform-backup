-- ============================================================
-- AI 安全评估平台 - 数据库初始化脚本
-- v1.0.0
-- ============================================================

-- 启用 UUID 扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================
-- 用户表
-- ============================================================
CREATE TABLE IF NOT EXISTS users (
    id            UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         VARCHAR(255)   NOT NULL UNIQUE,
    password_hash VARCHAR(255)   NOT NULL,
    name          VARCHAR(100)   NOT NULL,
    role          VARCHAR(20)    NOT NULL CHECK (role IN ('enterprise', 'expert', 'admin')),
    org_name      VARCHAR(255)   NOT NULL DEFAULT '',
    balance       DECIMAL(12,4)  NOT NULL DEFAULT 0.00,
    is_active     BOOLEAN        NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_role  ON users(role);

-- ============================================================
-- 目标系统表（被评估的 LLM/Agent 系统）
-- ============================================================
CREATE TABLE IF NOT EXISTS target_systems (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(255) NOT NULL,
    type       VARCHAR(20)  NOT NULL CHECK (type IN ('openai', 'agent', 'custom')),
    endpoint   TEXT         NOT NULL,
    api_key    TEXT         NOT NULL DEFAULT '',
    config     JSONB        NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_target_systems_user_id ON target_systems(user_id);

-- ============================================================
-- 专家资产表（先建，assessment 会引用它）
-- ============================================================
CREATE TABLE IF NOT EXISTS assets (
    id          UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    expert_id   UUID           NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255)   NOT NULL,
    description TEXT           NOT NULL DEFAULT '',
    type        VARCHAR(20)    NOT NULL CHECK (type IN ('tool_config', 'workflow', 'suite')),
    visibility  VARCHAR(20)    NOT NULL DEFAULT 'private'
                               CHECK (visibility IN ('private', 'org', 'public')),
    status      VARCHAR(20)    NOT NULL DEFAULT 'draft'
                               CHECK (status IN ('draft', 'testing', 'published', 'deprecated')),
    version     VARCHAR(20)    NOT NULL DEFAULT '1.0.0',
    config      JSONB          NOT NULL DEFAULT '{}',
    price_unit  DECIMAL(10,4)  NOT NULL DEFAULT 0.00,
    call_count  BIGINT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assets_expert_id  ON assets(expert_id);
CREATE INDEX IF NOT EXISTS idx_assets_type       ON assets(type);
CREATE INDEX IF NOT EXISTS idx_assets_visibility ON assets(visibility);
CREATE INDEX IF NOT EXISTS idx_assets_status     ON assets(status);

-- ============================================================
-- 评估任务表
-- ============================================================
CREATE TABLE IF NOT EXISTS assessments (
    id           UUID             PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id      UUID             NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR(255)     NOT NULL,
    description  TEXT             NOT NULL DEFAULT '',
    goal         TEXT             NOT NULL,
    target_id    UUID             NOT NULL REFERENCES target_systems(id),
    template_id  UUID             REFERENCES assets(id),
    status       VARCHAR(20)      NOT NULL DEFAULT 'pending'
                                  CHECK (status IN ('pending', 'running', 'completed', 'failed', 'canceled')),
    plan         JSONB            NOT NULL DEFAULT '[]',
    error_msg    TEXT             NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_assessments_user_id    ON assessments(user_id);
CREATE INDEX IF NOT EXISTS idx_assessments_status     ON assessments(status);
CREATE INDEX IF NOT EXISTS idx_assessments_created_at ON assessments(created_at DESC);

-- ============================================================
-- 评估执行日志表（每次工具调用记录）
-- ============================================================
CREATE TABLE IF NOT EXISTS assessment_logs (
    id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    assessment_id UUID         NOT NULL REFERENCES assessments(id) ON DELETE CASCADE,
    iteration     INT          NOT NULL DEFAULT 0,
    tool_name     VARCHAR(100) NOT NULL,
    input         JSONB        NOT NULL DEFAULT '{}',
    output        JSONB        NOT NULL DEFAULT '{}',
    thinking      TEXT         NOT NULL DEFAULT '',
    tokens_used   INT          NOT NULL DEFAULT 0,
    duration_ms   BIGINT       NOT NULL DEFAULT 0,
    severity      VARCHAR(20)  NOT NULL DEFAULT 'info'
                               CHECK (severity IN ('critical', 'high', 'medium', 'low', 'info')),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_assessment_logs_assessment_id ON assessment_logs(assessment_id);
CREATE INDEX IF NOT EXISTS idx_assessment_logs_tool_name     ON assessment_logs(tool_name);

-- ============================================================
-- 评估报告表
-- ============================================================
CREATE TABLE IF NOT EXISTS reports (
    id            UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    assessment_id UUID        NOT NULL UNIQUE REFERENCES assessments(id) ON DELETE CASCADE,
    title         VARCHAR(255) NOT NULL,
    summary       TEXT         NOT NULL DEFAULT '',
    risk_level    VARCHAR(20)  NOT NULL DEFAULT 'info'
                               CHECK (risk_level IN ('critical', 'high', 'medium', 'low', 'info')),
    findings      JSONB        NOT NULL DEFAULT '[]',
    metrics       JSONB        NOT NULL DEFAULT '{}',
    raw_content   TEXT         NOT NULL DEFAULT '',
    pdf_url       TEXT         NOT NULL DEFAULT '',
    html_url      TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reports_assessment_id ON reports(assessment_id);

-- ============================================================
-- 计费记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS billing_records (
    id            UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID           NOT NULL REFERENCES users(id),
    assessment_id UUID           REFERENCES assessments(id),
    tool_name     VARCHAR(100)   NOT NULL DEFAULT '',
    tool_category VARCHAR(50)    NOT NULL DEFAULT '',
    call_count    INT            NOT NULL DEFAULT 1,
    tokens_used   INT            NOT NULL DEFAULT 0,
    amount        DECIMAL(12,6)  NOT NULL DEFAULT 0.00,
    asset_id      UUID           REFERENCES assets(id),
    expert_id     UUID           REFERENCES users(id),
    expert_share  DECIMAL(12,6)  NOT NULL DEFAULT 0.00,
    created_at    TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_billing_records_user_id       ON billing_records(user_id);
CREATE INDEX IF NOT EXISTS idx_billing_records_assessment_id ON billing_records(assessment_id);
CREATE INDEX IF NOT EXISTS idx_billing_records_created_at    ON billing_records(created_at DESC);

-- ============================================================
-- 余额变动记录表
-- ============================================================
CREATE TABLE IF NOT EXISTS balance_transactions (
    id             UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id        UUID           NOT NULL REFERENCES users(id),
    type           VARCHAR(20)    NOT NULL CHECK (type IN ('recharge', 'deduct', 'refund')),
    amount         DECIMAL(12,4)  NOT NULL,
    balance_before DECIMAL(12,4)  NOT NULL,
    balance_after  DECIMAL(12,4)  NOT NULL,
    description    TEXT           NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_balance_transactions_user_id ON balance_transactions(user_id);

-- ============================================================
-- 审计日志表
-- ============================================================
CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID         REFERENCES users(id),
    action      VARCHAR(100) NOT NULL,
    resource    VARCHAR(100) NOT NULL DEFAULT '',
    resource_id UUID,
    detail      JSONB        NOT NULL DEFAULT '{}',
    ip          VARCHAR(50)  NOT NULL DEFAULT '',
    user_agent  TEXT         NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id    ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action     ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- ============================================================
-- updated_at 自动更新触发器
-- ============================================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS trg_assessments_updated_at ON assessments;
CREATE TRIGGER trg_assessments_updated_at
    BEFORE UPDATE ON assessments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS trg_assets_updated_at ON assets;
CREATE TRIGGER trg_assets_updated_at
    BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- ============================================================
-- 初始管理员账户
-- 密码: admin123456（生产环境请务必修改）
-- bcrypt hash of "admin123456"
-- ============================================================
INSERT INTO users (email, password_hash, name, role, org_name)
VALUES (
    'admin@platform.local',
    '$2a$10$CXiVG3vSaCNDOvIh0Ixn1u9CsQUAw/wiQsiU2NuneMT.WtwPEjUBK',
    '平台管理员',
    'admin',
    'AI安全评估平台'
) ON CONFLICT (email) DO NOTHING;
