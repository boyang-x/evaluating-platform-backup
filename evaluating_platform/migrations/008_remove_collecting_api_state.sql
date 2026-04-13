-- ============================================================
-- 移除 collecting_api 状态
-- 被测 LLM 配置改为从账户设置自动读取，不再需要在聊天中收集 API 信息
-- ============================================================

-- 将残留的 collecting_api 状态回退到 collecting_intent
UPDATE chat_sessions SET state = 'collecting_intent' WHERE state = 'collecting_api';

-- 删除旧约束，添加不含 collecting_api 的新约束
ALTER TABLE chat_sessions DROP CONSTRAINT IF EXISTS chat_sessions_state_check;
ALTER TABLE chat_sessions ADD CONSTRAINT chat_sessions_state_check
    CHECK (state IN ('idle','collecting_intent','confirming_plan','running','completed','failed'));
