ALTER TABLE skills DROP CONSTRAINT IF EXISTS skills_status_check;

ALTER TABLE skills
    ADD CONSTRAINT skills_status_check
    CHECK (status IN ('draft', 'published', 'disabled', 'deprecated'));
