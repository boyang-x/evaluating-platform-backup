-- 004_app_detector.sql: 应用检测工具表
CREATE TABLE IF NOT EXISTS app_detectors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expert_id UUID NOT NULL,
    name VARCHAR(200) NOT NULL,
    sub_type VARCHAR(50) NOT NULL DEFAULT 'llm_detector',
    description TEXT DEFAULT '',
    target_config JSONB NOT NULL DEFAULT '{}',
    sample_ids UUID[] DEFAULT '{}',
    template_ids UUID[] DEFAULT '{}',
    package_ids UUID[] DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_app_detectors_expert ON app_detectors(expert_id);
CREATE INDEX IF NOT EXISTS idx_app_detectors_sub_type ON app_detectors(sub_type);
