DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'templates'
    ) THEN
        ALTER TABLE templates DROP CONSTRAINT IF EXISTS templates_sub_type_check;
    END IF;
END $$;
