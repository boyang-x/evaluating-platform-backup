ALTER TABLE templates
    ADD COLUMN IF NOT EXISTS upload_batch_id UUID,
    ADD COLUMN IF NOT EXISTS upload_batch_name VARCHAR(255) NOT NULL DEFAULT '';

UPDATE templates
SET upload_batch_id = id
WHERE upload_batch_id IS NULL;

UPDATE templates
SET upload_batch_name = CASE
    WHEN COALESCE(NULLIF(name, ''), '') <> '' THEN name
    ELSE 'default template batch'
END
WHERE upload_batch_name = '';

ALTER TABLE templates
    ALTER COLUMN upload_batch_id SET NOT NULL,
    ALTER COLUMN upload_batch_name SET NOT NULL;

CREATE INDEX IF NOT EXISTS idx_templates_upload_batch_id ON templates(upload_batch_id);
CREATE INDEX IF NOT EXISTS idx_templates_upload_batch_name ON templates(upload_batch_name);