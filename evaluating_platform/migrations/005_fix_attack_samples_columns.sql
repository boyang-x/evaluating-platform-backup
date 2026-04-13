-- ============================================================
-- 修复 attack_samples 表列名：加密存储 → 明文 MinIO 存储
-- 将 encrypted_path 重命名为 storage_path，删除废弃的加密相关列
-- 幂等：仅在旧列名存在时执行
-- ============================================================

DO $$
BEGIN
    -- 仅当 encrypted_path 列存在时才重命名
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'attack_samples' AND column_name = 'encrypted_path'
    ) THEN
        ALTER TABLE attack_samples RENAME COLUMN encrypted_path TO storage_path;
    END IF;
END $$;

ALTER TABLE attack_samples DROP COLUMN IF EXISTS encryption_key_id;
ALTER TABLE attack_samples DROP COLUMN IF EXISTS field_names;
