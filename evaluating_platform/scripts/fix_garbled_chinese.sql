-- =============================================================
-- Fix garbled Chinese data caused by MSYS curl encoding bug
-- Root cause: MSYS bash curl -d '中文' passes args through
-- Windows API (UTF-8 → GBK conversion), corrupting Chinese chars.
--
-- Uses U&'\XXXX' Unicode escape syntax (pure ASCII) so this
-- script itself cannot be corrupted when executed.
-- =============================================================

BEGIN;

-- ─── Fix asset names & descriptions ──────────────────────────────────────────

-- workflow assets: LLM安全基础评估套件
--   description:   覆盖提示注入与越狱检测的轻量评估工作流
UPDATE assets SET
  name        = U&'LLM\5B89\5168\57FA\7840\8BC4\4F30\5957\4EF6',
  description = U&'\8986\76D6\63D0\793A\6CE8\5165\4E0E\8D8A\72B1\68C0\6D4B\7684\8F7B\91CF\8BC4\4F30\5DE5\4F5C\6D41'
WHERE type = 'workflow';

-- published tool_config assets: 私有单工具配置
--   description: 仅内部使用
UPDATE assets SET
  name        = U&'\79C1\6709\5355\5DE5\5177\914D\7F6E',
  description = U&'\4EC5\5185\90E8\4F7F\7528'
WHERE type = 'tool_config' AND status = 'published';

-- draft tool_config assets: 升级后创建
UPDATE assets SET
  name = U&'\5347\7EA7\540E\521B\5EFA'
WHERE type = 'tool_config' AND status = 'draft';

-- ─── Fix user display names ───────────────────────────────────────────────────

-- enterprise@test.com  →  企业用户
UPDATE users SET name = U&'\4F01\4E1A\7528\6237'
WHERE email = 'enterprise@test.com';

-- expert_TIMESTAMP@test.com  →  测试专家张三
UPDATE users SET name = U&'\6D4B\8BD5\4E13\5BB6\5F20\4E09'
WHERE email LIKE 'expert\_%@test.com' ESCAPE '\';

-- enterprise_TIMESTAMP@test.com  →  测试企业
UPDATE users SET name = U&'\6D4B\8BD5\4F01\4E1A'
WHERE email LIKE 'enterprise\_%@test.com' ESCAPE '\';

-- promote_TIMESTAMP@test.com  →  待升级用户
UPDATE users SET name = U&'\5F85\5347\7EA7\7528\6237'
WHERE email LIKE 'promote\_%@test.com' ESCAPE '\';

-- enctest@test.com  →  编码测试用户
UPDATE users SET name = U&'\7F16\7801\6D4B\8BD5\7528\6237'
WHERE email = 'enctest@test.com';

-- scripttest@test.com  →  脚本测试用户
UPDATE users SET name = U&'\811A\672C\6D4B\8BD5\7528\6237'
WHERE email = 'scripttest@test.com';

COMMIT;

-- ─── Verify results ───────────────────────────────────────────────────────────
SELECT 'asset' AS tbl, name, type, status FROM assets ORDER BY type, status;
SELECT 'user'  AS tbl, email, name, role  FROM users  ORDER BY role, email;
