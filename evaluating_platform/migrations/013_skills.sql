CREATE TABLE IF NOT EXISTS skills (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    expert_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                 VARCHAR(255) NOT NULL,
    slug                 VARCHAR(255) NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    skill_type           VARCHAR(50) NOT NULL DEFAULT 'generator_skill',
    category             VARCHAR(100) NOT NULL DEFAULT '',
    capability_profile   VARCHAR(100) NOT NULL DEFAULT '',
    status               VARCHAR(20) NOT NULL DEFAULT 'draft'
                         CHECK (status IN ('draft', 'published', 'disabled', 'deprecated')),
    latest_version_id    UUID,
    published_version_id UUID,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_skills_expert_slug UNIQUE (expert_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_skills_expert_id ON skills(expert_id);
CREATE INDEX IF NOT EXISTS idx_skills_status ON skills(status);
CREATE INDEX IF NOT EXISTS idx_skills_skill_type ON skills(skill_type);

CREATE TABLE IF NOT EXISTS skill_versions (
    id                        UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    skill_id                  UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    version                   VARCHAR(50) NOT NULL,
    manifest_version          VARCHAR(50) NOT NULL DEFAULT '1.0',
    display_name              VARCHAR(255) NOT NULL,
    summary                   TEXT NOT NULL DEFAULT '',
    package_object_path       TEXT NOT NULL,
    package_hash              VARCHAR(128) NOT NULL,
    package_size              BIGINT NOT NULL DEFAULT 0,
    prompt_text               TEXT NOT NULL DEFAULT '',
    input_source_mode         VARCHAR(50) NOT NULL DEFAULT 'embedded_dataset_only',
    execution_runtime         VARCHAR(50) NOT NULL DEFAULT 'python3.12',
    execution_entrypoint      TEXT NOT NULL,
    self_test_entrypoint      TEXT NOT NULL,
    permissions               JSONB NOT NULL DEFAULT '{}',
    embedded_dataset_summary  JSONB NOT NULL DEFAULT '{}',
    assessment_types          JSONB NOT NULL DEFAULT '[]',
    metadata                  JSONB NOT NULL DEFAULT '{}',
    validation_report         JSONB NOT NULL DEFAULT '{}',
    examples                  JSONB NOT NULL DEFAULT '{}',
    status                    VARCHAR(30) NOT NULL DEFAULT 'draft'
                              CHECK (status IN ('draft', 'self_test_passed', 'self_test_failed', 'published', 'deprecated')),
    last_self_test_run_id     UUID,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_skill_versions_skill_version UNIQUE (skill_id, version)
);

CREATE INDEX IF NOT EXISTS idx_skill_versions_skill_id ON skill_versions(skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_versions_status ON skill_versions(status);

CREATE TABLE IF NOT EXISTS skill_runs (
    id                     UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    skill_id               UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    skill_version_id       UUID NOT NULL REFERENCES skill_versions(id) ON DELETE CASCADE,
    assessment_id          UUID REFERENCES assessments(id) ON DELETE SET NULL,
    run_type               VARCHAR(30) NOT NULL CHECK (run_type IN ('self_test', 'generate')),
    trigger_source         VARCHAR(30) NOT NULL DEFAULT 'expert_portal'
                           CHECK (trigger_source IN ('expert_portal', 'orchestrator', 'system')),
    status                 VARCHAR(30) NOT NULL DEFAULT 'pending'
                           CHECK (status IN ('pending', 'running', 'completed', 'failed', 'timeout')),
    exit_code              INT,
    stdout_log             TEXT NOT NULL DEFAULT '',
    stderr_log             TEXT NOT NULL DEFAULT '',
    result_payload         JSONB NOT NULL DEFAULT '{}',
    validation_report      JSONB NOT NULL DEFAULT '{}',
    payload_dataset_summary JSONB NOT NULL DEFAULT '{}',
    error_message          TEXT NOT NULL DEFAULT '',
    started_at             TIMESTAMPTZ,
    completed_at           TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_skill_runs_skill_id ON skill_runs(skill_id);
CREATE INDEX IF NOT EXISTS idx_skill_runs_skill_version_id ON skill_runs(skill_version_id);
CREATE INDEX IF NOT EXISTS idx_skill_runs_assessment_id ON skill_runs(assessment_id);
CREATE INDEX IF NOT EXISTS idx_skill_runs_status ON skill_runs(status);
CREATE INDEX IF NOT EXISTS idx_skill_runs_created_at ON skill_runs(created_at DESC);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_skills_latest_version') THEN
        ALTER TABLE skills
            ADD CONSTRAINT fk_skills_latest_version
            FOREIGN KEY (latest_version_id) REFERENCES skill_versions(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_skills_published_version') THEN
        ALTER TABLE skills
            ADD CONSTRAINT fk_skills_published_version
            FOREIGN KEY (published_version_id) REFERENCES skill_versions(id) ON DELETE SET NULL;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_skill_versions_last_self_test_run') THEN
        ALTER TABLE skill_versions
            ADD CONSTRAINT fk_skill_versions_last_self_test_run
            FOREIGN KEY (last_self_test_run_id) REFERENCES skill_runs(id) ON DELETE SET NULL;
    END IF;
END $$;

DROP TRIGGER IF EXISTS trg_skills_updated_at ON skills;
CREATE TRIGGER trg_skills_updated_at
    BEFORE UPDATE ON skills
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS trg_skill_versions_updated_at ON skill_versions;
CREATE TRIGGER trg_skill_versions_updated_at
    BEFORE UPDATE ON skill_versions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS trg_skill_runs_updated_at ON skill_runs;
CREATE TRIGGER trg_skill_runs_updated_at
    BEFORE UPDATE ON skill_runs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
