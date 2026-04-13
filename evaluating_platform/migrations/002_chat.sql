-- ============================================================
-- Chat 会话功能迁移
-- v1.1.0
-- ============================================================

-- 会话状态枚举说明：
--   idle               初始状态，等待用户输入
--   collecting_intent  大模型正在识别用户意图
--   collecting_api     大模型追问 API 信息
--   confirming_plan    大模型已生成评估计划，等待用户确认
--   running            评估任务执行中
--   completed          评估完成
--   failed             会话异常终止

CREATE TABLE IF NOT EXISTS chat_sessions (
    id            UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title         VARCHAR(255) NOT NULL DEFAULT '新对话',
    state         VARCHAR(30)  NOT NULL DEFAULT 'idle'
                               CHECK (state IN ('idle','collecting_intent','collecting_api','confirming_plan','running','completed','failed')),
    assessment_id UUID         REFERENCES assessments(id),
    -- 收集到的目标系统信息（JSON）
    target_info   JSONB        NOT NULL DEFAULT '{}',
    -- 大模型识别出的评估计划（JSON）
    plan_info     JSONB        NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_id    ON chat_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_created_at ON chat_sessions(created_at DESC);

-- 消息角色：user | assistant | system | tool_event
CREATE TABLE IF NOT EXISTS chat_messages (
    id         UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID         NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
    role       VARCHAR(20)  NOT NULL CHECK (role IN ('user','assistant','system','tool_event')),
    content    TEXT         NOT NULL DEFAULT '',
    -- 结构化元数据：卡片类型、评估进度、工具调用结果等
    metadata   JSONB        NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_session_id ON chat_messages(session_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_created_at ON chat_messages(created_at ASC);

-- updated_at 触发器
DROP TRIGGER IF EXISTS trg_chat_sessions_updated_at ON chat_sessions;
CREATE TRIGGER trg_chat_sessions_updated_at
    BEFORE UPDATE ON chat_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
