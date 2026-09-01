CREATE TABLE IF NOT EXISTS maclaw_redteam_evidence (
    id               TEXT        PRIMARY KEY,
    handle           TEXT        NOT NULL UNIQUE,
    platform_user_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    instance_id      TEXT        NOT NULL DEFAULT '',
    session_id       TEXT        NOT NULL DEFAULT '',
    run_id           TEXT        NOT NULL DEFAULT '',
    kind             TEXT        NOT NULL DEFAULT 'artifact',
    title            TEXT        NOT NULL DEFAULT '',
    summary          TEXT        NOT NULL DEFAULT '',
    metadata         JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_maclaw_redteam_evidence_user_run
    ON maclaw_redteam_evidence(platform_user_id, run_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_maclaw_redteam_evidence_user_instance
    ON maclaw_redteam_evidence(platform_user_id, instance_id, created_at DESC);

DROP TRIGGER IF EXISTS trg_maclaw_redteam_evidence_updated_at ON maclaw_redteam_evidence;
CREATE TRIGGER trg_maclaw_redteam_evidence_updated_at
    BEFORE UPDATE ON maclaw_redteam_evidence
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TABLE IF NOT EXISTS maclaw_redteam_reports (
    id               TEXT        PRIMARY KEY,
    handle           TEXT        NOT NULL UNIQUE,
    platform_user_id UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    instance_id      TEXT        NOT NULL DEFAULT '',
    session_id       TEXT        NOT NULL DEFAULT '',
    run_id           TEXT        NOT NULL DEFAULT '',
    title            TEXT        NOT NULL DEFAULT '',
    summary          TEXT        NOT NULL DEFAULT '',
    risk_level       TEXT        NOT NULL DEFAULT '',
    safety_score     DOUBLE PRECISION,
    findings         JSONB       NOT NULL DEFAULT '[]'::jsonb,
    evidence_handles JSONB       NOT NULL DEFAULT '[]'::jsonb,
    metadata         JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_maclaw_redteam_reports_user_run
    ON maclaw_redteam_reports(platform_user_id, run_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_maclaw_redteam_reports_user_instance
    ON maclaw_redteam_reports(platform_user_id, instance_id, created_at DESC);

DROP TRIGGER IF EXISTS trg_maclaw_redteam_reports_updated_at ON maclaw_redteam_reports;
CREATE TRIGGER trg_maclaw_redteam_reports_updated_at
    BEFORE UPDATE ON maclaw_redteam_reports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
