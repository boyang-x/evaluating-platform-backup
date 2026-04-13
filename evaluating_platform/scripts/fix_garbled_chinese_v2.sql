-- =============================================================
-- Fix garbled Chinese data (v2)
-- Root cause: pgx connection without client_encoding=UTF8 on
-- Windows causes GBK→UTF8 encoding corruption.
--
-- Uses U&'\XXXX' Unicode escape syntax (pure ASCII) so this
-- script itself cannot be corrupted.
-- =============================================================

BEGIN;

-- ─── Fix user: test@example.com ──────────────────────────────────────────────
-- name: 测试用户  org_name: 测试公司
UPDATE users SET
  name     = U&'\6D4B\8BD5\7528\6237',
  org_name = U&'\6D4B\8BD5\516C\53F8'
WHERE email = 'test@example.com';

-- ─── Fix user: expert@test.com ───────────────────────────────────────────────
-- name: 安全专家  org_name: 安全研究院
UPDATE users SET
  name     = U&'\5B89\5168\4E13\5BB6',
  org_name = U&'\5B89\5168\7814\7A76\9662'
WHERE email = 'expert@test.com';

-- ─── Fix assessments with garbled names ──────────────────────────────────────
-- assessment: 提示词注入评估  goal: 测试目标LLM是否容易受到提示词注入攻击
UPDATE assessments SET
  name = U&'\63D0\793A\8BCD\6CE8\5165\8BC4\4F30',
  goal = U&'\6D4B\8BD5\76EE\6807LLM\662F\5426\5BB9\6613\53D7\5230\63D0\793A\8BCD\6CE8\5165\653B\51FB'
WHERE id = '4b39ca66-bea2-4c60-a900-e87620e0146d';

-- assessment: 取消的评估  goal: 测试取消评估
UPDATE assessments SET
  name = U&'\53D6\6D88\7684\8BC4\4F30',
  goal = U&'\6D4B\8BD5\53D6\6D88\8BC4\4F30'
WHERE id = 'a211cd69-62c5-4595-aded-78e1da85dba6';

-- ─── Fix any remaining garbled assets ────────────────────────────────────────
-- Reapply asset fixes from v1 script
UPDATE assets SET
  name        = U&'LLM\5B89\5168\57FA\7840\8BC4\4F30\5957\4EF6',
  description = U&'\8986\76D6\63D0\793A\6CE8\5165\4E0E\8D8A\72B1\68C0\6D4B\7684\8F7B\91CF\8BC4\4F30\5DE5\4F5C\6D41'
WHERE type = 'workflow';

UPDATE assets SET
  name        = U&'\79C1\6709\5355\5DE5\5177\914D\7F6E',
  description = U&'\4EC5\5185\90E8\4F7F\7528'
WHERE type = 'tool_config' AND status = 'published';

UPDATE assets SET
  name = U&'\5347\7EA7\540E\521B\5EFA'
WHERE type = 'tool_config' AND status = 'draft';

COMMIT;

-- ─── Verify ──────────────────────────────────────────────────────────────────
SELECT 'user' AS tbl, email, name, org_name FROM users ORDER BY role, email;
SELECT 'assessment' AS tbl, id, name, goal FROM assessments;
