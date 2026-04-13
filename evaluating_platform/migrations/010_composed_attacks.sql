CREATE TABLE IF NOT EXISTS composed_attacks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    expert_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sub_type VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    storage_path TEXT NOT NULL,
    file_hash VARCHAR(128) NOT NULL DEFAULT '',
    sample_count INT NOT NULL DEFAULT 0,
    file_size BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'published',
    visibility VARCHAR(20) NOT NULL DEFAULT 'public',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_composed_attacks_expert_id ON composed_attacks(expert_id);
CREATE INDEX IF NOT EXISTS idx_composed_attacks_sub_type ON composed_attacks(sub_type);
CREATE INDEX IF NOT EXISTS idx_composed_attacks_created_at ON composed_attacks(created_at DESC);

DROP TRIGGER IF EXISTS trg_composed_attacks_updated_at ON composed_attacks;
CREATE TRIGGER trg_composed_attacks_updated_at
    BEFORE UPDATE ON composed_attacks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
