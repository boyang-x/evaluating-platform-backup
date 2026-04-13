#!/usr/bin/env bash
# =============================================================================
# 安全专家完整使用流程端到端验证
# 覆盖：注册 → 创建工作流 → 提交审核 → 管理员审核 → 企业调用 → 收益查询
# 环境：需要后端运行于 localhost:8080
# =============================================================================

BASE_URL="${BASE_URL:-http://localhost:8080/api/v1}"
TIMESTAMP=$(date +%s)

EXPERT_EMAIL="expert_${TIMESTAMP}@test.com"
EXPERT_PASS="expertpass123"
ENTERPRISE_EMAIL="enterprise_${TIMESTAMP}@test.com"
ENTERPRISE_PASS="enterprisepass123"
ADMIN_EMAIL="${ADMIN_EMAIL:-admin@platform.local}"
ADMIN_PASS="${ADMIN_PASS:-admin123456}"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
BLUE='\033[0;34m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

PASS=0; FAIL=0

# curl_json <json_body> [curl_options...] <url>
# Sends JSON body via stdin to avoid MSYS/Windows UTF-8 argument encoding corruption.
# On Windows, curl args go through the Windows API which converts UTF-8 → system code page;
# using stdin bypasses this and preserves Chinese characters correctly.
curl_json() {
    local payload="$1"; shift
    printf '%s' "$payload" | curl --data-binary @- "$@"
}

log_section() { echo -e "\n${CYAN}${BOLD}══════ $1 ══════${NC}"; }
log_step()    { echo -e "\n${BLUE}▶ $1${NC}"; }
log_ok()      { echo -e "  ${GREEN}✓ $1${NC}"; PASS=$((PASS + 1)); }
log_fail()    { echo -e "  ${RED}✗ $1${NC}"; FAIL=$((FAIL + 1)); }
log_info()    { echo -e "  ${YELLOW}ℹ $1${NC}"; }

check_eq() {
    local desc="$1" got="$2" expected="$3"
    if [ "$got" = "$expected" ]; then log_ok "$desc"
    else log_fail "$desc（期望='$expected' 实际='$got'）"; fi
}

check_ne() {
    local desc="$1" got="$2" unexpected="$3"
    if [ "$got" != "$unexpected" ]; then log_ok "$desc"
    else log_fail "$desc（值不应为 '$got'）"; fi
}

check_http() {
    local desc="$1" code="$2" expected="$3"
    if [ "$code" = "$expected" ]; then log_ok "$desc（HTTP $code）"
    else log_fail "$desc（期望 HTTP $expected，实际 $code）"; fi
}

check_contains() {
    local desc="$1" haystack="$2" needle="$3"
    if echo "$haystack" | grep -q "$needle" 2>/dev/null; then log_ok "$desc"
    else log_fail "$desc（响应中未找到 '$needle'）"; echo "    响应: ${haystack:0:200}"; fi
}

# 用 node.js 提取 JSON 字段（不依赖 python3/jq）
jget() {
    local json="$1" key="$2"
    echo "$json" | node -e "
try {
  let d = '';
  process.stdin.resume();
  process.stdin.on('data', c => d += c);
  process.stdin.on('end', () => {
    try { let r = JSON.parse(d)$key; process.stdout.write(String(r === null || r === undefined ? '' : r)); }
    catch(e) { process.stdout.write(''); }
  });
} catch(e) { process.stdout.write(''); }
" 2>/dev/null || echo ""
}

# ─── 前置检查 ──────────────────────────────────────────────────────────────
log_section "前置检查"

log_step "确认服务器可达"
HEALTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" 2>/dev/null || echo "000")
if [ "$HEALTH_CODE" != "200" ]; then
    echo -e "${RED}服务器未运行！请先执行：go run ./cmd/server/ 或 docker compose up${NC}"
    exit 1
fi
log_ok "服务器正常运行 ($BASE_URL)"

log_step "查询工具定价表（公开接口，无需登录）"
PRICES=$(curl -s "$BASE_URL/billing/prices")
check_contains "定价表可访问" "$PRICES" "prompt_injection"
check_contains "定价表包含 expert_share_ratio" "$PRICES" "expert_share_ratio"
check_contains "定价表包含最低余额要求" "$PRICES" "min_assessment_balance"

# ─── SECTION 1：专家注册与登录 ─────────────────────────────────────────────
log_section "专家注册与登录"

log_step "注册专家账号 ($EXPERT_EMAIL)"
EXPERT_REG_RESP=$(curl_json \
    "{\"email\":\"$EXPERT_EMAIL\",\"password\":\"$EXPERT_PASS\",\"name\":\"测试专家张三\",\"role\":\"expert\"}" \
    -s -w "\n%{http_code}" -X POST "$BASE_URL/auth/register" \
    -H "Content-Type: application/json")
EXPERT_REG_BODY=$(echo "$EXPERT_REG_RESP" | head -n1)
EXPERT_REG_CODE=$(echo "$EXPERT_REG_RESP" | tail -n1)
check_http "专家注册成功" "$EXPERT_REG_CODE" "201"
check_contains "响应含 user_id" "$EXPERT_REG_BODY" "user_id"

log_step "重复注册同邮箱（应报错）"
DUP_REG=$(curl_json \
    "{\"email\":\"$EXPERT_EMAIL\",\"password\":\"$EXPERT_PASS\",\"name\":\"重复\",\"role\":\"expert\"}" \
    -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/auth/register" \
    -H "Content-Type: application/json")
check_http "重复注册被拒（500/400）" "$DUP_REG" "500"

log_step "专家登录，获取 JWT"
EXPERT_LOGIN=$(curl -s -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$EXPERT_EMAIL\",\"password\":\"$EXPERT_PASS\"}")
EXPERT_TOKEN=$(jget "$EXPERT_LOGIN" "['token']")
EXPERT_ID=$(jget "$EXPERT_LOGIN" "['user']['id']")
EXPERT_ROLE=$(jget "$EXPERT_LOGIN" "['user']['role']")
check_ne "登录 token 非空" "$EXPERT_TOKEN" ""
check_eq "角色为 expert" "$EXPERT_ROLE" "expert"
log_info "专家 ID: $EXPERT_ID"

log_step "查询专家初始余额（新注册用户余额为 0）"
EXPERT_BAL_RESP=$(curl -s "$BASE_URL/billing/balance" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
EXPERT_BAL_VAL=$(jget "$EXPERT_BAL_RESP" "['balance']")
log_info "专家初始余额: $EXPERT_BAL_VAL 元"
check_eq "专家初始余额为 0" "$EXPERT_BAL_VAL" "0"

# ─── SECTION 2：专家创建资产 ──────────────────────────────────────────────
log_section "专家创建工作流资产"

log_step "创建工作流资产（双节点串行：提示注入 → 越狱检测）"
WORKFLOW_PAYLOAD='{
    "name": "LLM安全基础评估套件",
    "description": "覆盖提示注入与越狱检测的轻量评估工作流",
    "type": "workflow",
    "visibility": "public",
    "version": "1.0.0",
    "price_unit": 3.00,
    "config": {
        "nodes": [
            {
                "id": "n1",
                "tool_name": "prompt_injection",
                "label": "提示注入检测",
                "params": {"scenario": "chatbot", "intensity": "medium"},
                "position": {"x": 100, "y": 100}
            },
            {
                "id": "n2",
                "tool_name": "jailbreak",
                "label": "越狱攻击检测",
                "params": {"intensity": "medium"},
                "position": {"x": 400, "y": 100}
            }
        ],
        "edges": [
            {"id": "e1", "source": "n1", "target": "n2"}
        ],
        "params": {"note": "基础 LLM 安全评估"}
    }
}'
ASSET_RESP=$(curl_json "$WORKFLOW_PAYLOAD" \
    -s -w "\n%{http_code}" -X POST "$BASE_URL/tools" \
    -H "Authorization: Bearer $EXPERT_TOKEN" \
    -H "Content-Type: application/json")
ASSET_BODY=$(echo "$ASSET_RESP" | head -n1)
ASSET_CODE=$(echo "$ASSET_RESP" | tail -n1)
check_http "创建工作流资产成功" "$ASSET_CODE" "201"
ASSET_ID=$(jget "$ASSET_BODY" "['asset_id']")
ASSET_STATUS=$(jget "$ASSET_BODY" "['status']")
check_ne "资产 ID 非空" "$ASSET_ID" ""
check_eq "初始状态为 draft" "$ASSET_STATUS" "draft"
log_info "工作流资产 ID: $ASSET_ID"

log_step "创建第二个资产（private 可见性，验证直接发布路径）"
ASSET2_RESP=$(curl_json \
    '{"name":"私有单工具配置","description":"仅内部使用","type":"tool_config","visibility":"private","version":"1.0.0","config":{"nodes":[],"edges":[]}}' \
    -s -w "\n%{http_code}" -X POST "$BASE_URL/tools" \
    -H "Authorization: Bearer $EXPERT_TOKEN" \
    -H "Content-Type: application/json")
ASSET2_BODY=$(echo "$ASSET2_RESP" | head -n1)
ASSET2_CODE=$(echo "$ASSET2_RESP" | tail -n1)
check_http "创建私有资产成功" "$ASSET2_CODE" "201"
ASSET2_ID=$(jget "$ASSET2_BODY" "['asset_id']")
log_info "私有资产 ID: $ASSET2_ID"

log_step "提交非法 type 资产（应被拒绝）"
BAD_TYPE_RESP=$(curl_json \
    '{"name":"非法类型","type":"invalid_type","config":{"nodes":[],"edges":[]}}' \
    -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/tools" \
    -H "Authorization: Bearer $EXPERT_TOKEN" \
    -H "Content-Type: application/json")
check_http "非法 type 被拒绝（400）" "$BAD_TYPE_RESP" "400"

# ─── SECTION 3：专家资产管理 ──────────────────────────────────────────────
log_section "专家查看与管理自己的资产"

log_step "查看自己的所有资产（?mine=1）"
MY_ASSETS=$(curl -s "$BASE_URL/tools?mine=1" -H "Authorization: Bearer $EXPERT_TOKEN")
MY_TOTAL=$(jget "$MY_ASSETS" "['total']")
check_eq "能看到自己的 2 个资产" "$MY_TOTAL" "2"

log_step "公开市场此时不含该专家资产（均为 draft 状态）"
PUBLIC_BEFORE=$(curl -s "$BASE_URL/tools" -H "Authorization: Bearer $EXPERT_TOKEN")
if echo "$PUBLIC_BEFORE" | grep -q "$ASSET_ID" 2>/dev/null; then
    log_fail "draft 资产不应出现在公开市场"
else
    log_ok "draft 资产未出现在公开市场"
fi

log_step "直接发布第二资产（draft → published，但 visibility=private）"
PUB2_RESP=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/tools/$ASSET2_ID/publish" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
PUB2_BODY=$(echo "$PUB2_RESP" | head -n1)
PUB2_CODE=$(echo "$PUB2_RESP" | tail -n1)
check_http "直接发布成功" "$PUB2_CODE" "200"
check_contains "返回 published 状态" "$PUB2_BODY" "published"

log_step "关键验证：直接发布的 private 资产 ≠ 公开市场可见（可见性隔离）"
PUBLIC_AFTER_PUB2=$(curl -s "$BASE_URL/tools" -H "Authorization: Bearer $EXPERT_TOKEN")
if echo "$PUBLIC_AFTER_PUB2" | grep -q "$ASSET2_ID" 2>/dev/null; then
    log_fail "BUG：visibility=private 的资产不应出现在公开市场！"
else
    log_ok "visibility=private 资产正确隔离，未出现在公开市场"
fi

log_step "幂等校验：已 published 资产再次直接发布 → 报错（状态非 draft/testing）"
REDUP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/tools/$ASSET2_ID/publish" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
check_http "重复直接发布返回 400" "$REDUP_CODE" "400"

# ─── SECTION 4：提交审核流程 ──────────────────────────────────────────────
log_section "提交资产审核（draft → testing）"

log_step "提交第一个工作流资产审核"
SUBMIT_RESP=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/tools/$ASSET_ID/submit" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
SUBMIT_BODY=$(echo "$SUBMIT_RESP" | head -n1)
SUBMIT_CODE=$(echo "$SUBMIT_RESP" | tail -n1)
check_http "提交审核成功" "$SUBMIT_CODE" "200"
SUBMIT_STATUS=$(jget "$SUBMIT_BODY" "['status']")
check_eq "状态变为 testing" "$SUBMIT_STATUS" "testing"

log_step "幂等校验：非 draft 状态再次提交审核 → 报错"
RESUBMIT_CODE=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/tools/$ASSET_ID/submit" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
check_http "非 draft 重复提交返回 400" "$RESUBMIT_CODE" "400"

# ─── SECTION 5：权限边界验证 ──────────────────────────────────────────────
log_section "权限边界验证"

log_step "注册并登录企业用户"
ENT_REG_CODE=$(curl_json \
    "{\"email\":\"$ENTERPRISE_EMAIL\",\"password\":\"$ENTERPRISE_PASS\",\"name\":\"测试企业\",\"role\":\"enterprise\"}" \
    -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/auth/register" \
    -H "Content-Type: application/json")
check_http "企业用户注册成功" "$ENT_REG_CODE" "201"

ENT_LOGIN=$(curl -s -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$ENTERPRISE_EMAIL\",\"password\":\"$ENTERPRISE_PASS\"}")
ENT_TOKEN=$(jget "$ENT_LOGIN" "['token']")
ENT_ROLE=$(jget "$ENT_LOGIN" "['user']['role']")
check_ne "企业登录 token 非空" "$ENT_TOKEN" ""
check_eq "企业用户角色正确" "$ENT_ROLE" "enterprise"

log_step "企业用户尝试创建资产 → 403（RequireRole expert/admin）"
ENT_ASSET_CODE=$(curl_json \
    '{"name":"违规","type":"workflow","config":{"nodes":[],"edges":[]}}' \
    -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/tools" \
    -H "Authorization: Bearer $ENT_TOKEN" \
    -H "Content-Type: application/json")
check_http "企业用户无法创建资产（403）" "$ENT_ASSET_CODE" "403"

log_step "企业用户尝试访问 /admin/assets/pending → 403"
ENT_ADMIN_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    "$BASE_URL/admin/assets/pending" -H "Authorization: Bearer $ENT_TOKEN")
check_http "企业用户访问管理员接口被拒（403）" "$ENT_ADMIN_CODE" "403"

log_step "专家用户尝试访问 /admin/assets/pending → 403"
EXPERT_ADMIN_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    "$BASE_URL/admin/assets/pending" -H "Authorization: Bearer $EXPERT_TOKEN")
check_http "专家用户访问管理员接口被拒（403）" "$EXPERT_ADMIN_CODE" "403"

log_step "未登录用户访问 /billing/balance → 401"
UNAUTH_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/billing/balance")
check_http "未登录访问余额接口被拒（401）" "$UNAUTH_CODE" "401"

log_step "企业用户尝试访问其他用户的评估（跨用户隔离）"
# 专家创建一个评估（余额充值后）
# 此步骤在后续 section 验证，这里仅验证权限体系正确

# ─── SECTION 6：管理员审核（驳回 + 重新提交 + 审核通过）─────────────────
log_section "管理员审核资产"

log_step "管理员登录"
ADMIN_LOGIN=$(curl -s -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASS\"}")
ADMIN_TOKEN=$(jget "$ADMIN_LOGIN" "['token']")
ADMIN_ROLE=$(jget "$ADMIN_LOGIN" "['user']['role']")
check_ne "管理员 token 非空" "$ADMIN_TOKEN" ""
check_eq "管理员角色正确" "$ADMIN_ROLE" "admin"

log_step "查看待审核资产列表"
PENDING=$(curl -s "$BASE_URL/admin/assets/pending" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
PENDING_TOTAL=$(jget "$PENDING" "['total']")
log_info "待审核资产数: $PENDING_TOTAL"
check_contains "待审核列表包含刚提交的资产" "$PENDING" "$ASSET_ID"

log_step "管理员驳回资产（testing → draft，验证驳回路径）"
REJECT_RESP=$(curl_json \
    '{"reason": "缺少工具参数描述，请补充后重新提交"}' \
    -s -w "\n%{http_code}" -X PUT "$BASE_URL/admin/assets/$ASSET_ID/reject" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json")
REJECT_CODE=$(echo "$REJECT_RESP" | tail -n1)
check_http "管理员驳回成功" "$REJECT_CODE" "200"

log_step "验证：驳回后资产从待审核列表移除"
PENDING_AFTER_REJECT=$(curl -s "$BASE_URL/admin/assets/pending" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
if echo "$PENDING_AFTER_REJECT" | grep -q "$ASSET_ID" 2>/dev/null; then
    log_fail "被驳回的资产不应还在待审核列表"
else
    log_ok "被驳回的资产已从待审核列表移除"
fi

log_step "验证：重复驳回（已回到 draft）→ 报错"
REREJECT_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X PUT "$BASE_URL/admin/assets/$ASSET_ID/reject" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{}')
check_http "重复驳回报错（400）" "$REREJECT_CODE" "400"

log_step "专家重新提交（被驳回后状态回到 draft，可再次提交）"
RESUBMIT2_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X PUT "$BASE_URL/tools/$ASSET_ID/submit" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
check_http "被驳回后可重新提交审核" "$RESUBMIT2_CODE" "200"

log_step "管理员审核通过（testing → published + visibility=public）"
APPROVE_RESP=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/admin/assets/$ASSET_ID/approve" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
APPROVE_BODY=$(echo "$APPROVE_RESP" | head -n1)
APPROVE_CODE=$(echo "$APPROVE_RESP" | tail -n1)
check_http "管理员审核通过" "$APPROVE_CODE" "200"
check_contains "审核通过消息正确" "$APPROVE_BODY" "审核通过"

log_step "审核通过后公开市场应可见该资产（用企业 token 查看）"
PUBLIC_MARKET=$(curl -s "$BASE_URL/tools" -H "Authorization: Bearer $ENT_TOKEN")
check_contains "已审核资产出现在公开市场" "$PUBLIC_MARKET" "$ASSET_ID"

log_step "验证：重复审核通过（已 published，非 testing）→ 400"
RE_APPROVE_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X PUT "$BASE_URL/admin/assets/$ASSET_ID/approve" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
check_http "重复审核通过返回 400" "$RE_APPROVE_CODE" "400"

# ─── SECTION 7：企业用户通过工作流发起评估 ────────────────────────────────
log_section "企业用户调用专家工作流发起评估"

log_step "余额不足时拒绝发起评估（402 Payment Required）"
BROKE_ASSESS_CODE=$(curl_json \
    "{\"name\":\"余额不足测试\",\"goal\":\"test\",\"target_type\":\"openai\",\"target_url\":\"http://mock\",\"target_key\":\"k\",\"template_id\":\"$ASSET_ID\"}" \
    -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/assessments" \
    -H "Authorization: Bearer $ENT_TOKEN" \
    -H "Content-Type: application/json")
check_http "余额不足拒绝评估（402）" "$BROKE_ASSESS_CODE" "402"

log_step "企业用户充值 20 元"
RECHARGE_RESP=$(curl_json \
    '{"amount": 20.0, "description": "测试充值"}' \
    -s -w "\n%{http_code}" -X POST "$BASE_URL/billing/recharge" \
    -H "Authorization: Bearer $ENT_TOKEN" \
    -H "Content-Type: application/json")
RECHARGE_CODE=$(echo "$RECHARGE_RESP" | tail -n1)
RECHARGE_BODY=$(echo "$RECHARGE_RESP" | head -n1)
check_http "充值成功" "$RECHARGE_CODE" "200"
NEW_BAL=$(jget "$RECHARGE_BODY" "['new_balance']")
check_eq "充值后余额为 20" "$NEW_BAL" "20"

log_step "企业用户查看余额变动记录（应含充值记录）"
TXN_RESP=$(curl -s "$BASE_URL/billing/transactions" -H "Authorization: Bearer $ENT_TOKEN")
TXN_TOTAL=$(jget "$TXN_RESP" "['total']")
check_ne "余额变动记录数 > 0" "$TXN_TOTAL" "0"

log_step "使用专家工作流创建评估（template_id=$ASSET_ID）"
ASSESS_RESP=$(curl_json \
    "{\"name\":\"LLM合规评估\",\"description\":\"使用专家工作流进行合规性评估\",\"goal\":\"评估目标 LLM 的安全性与合规性\",\"target_type\":\"openai\",\"target_url\":\"http://mock-llm.test/v1\",\"target_key\":\"mock-key\",\"target_model\":\"mock-model\",\"template_id\":\"$ASSET_ID\"}" \
    -s -w "\n%{http_code}" -X POST "$BASE_URL/assessments" \
    -H "Authorization: Bearer $ENT_TOKEN" \
    -H "Content-Type: application/json")
ASSESS_BODY=$(echo "$ASSESS_RESP" | head -n1)
ASSESS_CODE=$(echo "$ASSESS_RESP" | tail -n1)
check_http "使用专家工作流创建评估（202 Accepted）" "$ASSESS_CODE" "202"
ASSESSMENT_ID=$(jget "$ASSESS_BODY" "['assessment_id']")
check_ne "评估 ID 非空" "$ASSESSMENT_ID" ""
log_info "评估 ID: $ASSESSMENT_ID"

log_step "验证评估记录已创建（GET /assessments/:id）"
ASSESS_GET=$(curl -s "$BASE_URL/assessments/$ASSESSMENT_ID" -H "Authorization: Bearer $ENT_TOKEN")
ASSESS_STATUS=$(jget "$ASSESS_GET" "['assessment']['status']")
log_info "评估当前状态: $ASSESS_STATUS"
if [ "$ASSESS_STATUS" = "pending" ] || [ "$ASSESS_STATUS" = "running" ] || \
   [ "$ASSESS_STATUS" = "completed" ] || [ "$ASSESS_STATUS" = "failed" ]; then
    log_ok "评估状态正常（$ASSESS_STATUS）"
else
    log_fail "评估状态异常（$ASSESS_STATUS）"
fi

log_step "工作流资产调用计数应增加（IncrCallCount）"
ASSET_AFTER=$(curl -s "$BASE_URL/tools" -H "Authorization: Bearer $ENT_TOKEN")
# 资产的 call_count 在评估成功完成后自增，由于目标是 mock，评估可能失败
# 此处仅确认 API 调用正确触发评估创建流程
log_info "（call_count 在评估成功时自增，mock target 下评估可能失败）"

log_step "验证：企业用户无法查看其他用户的评估"
ASSESS_STEAL_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    "$BASE_URL/assessments/$ASSESSMENT_ID" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
check_http "专家无法查看企业用户的评估（404）" "$ASSESS_STEAL_CODE" "404"

log_step "验证：企业用户无法取消不属于自己的评估"
CANCEL_STEAL_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
    -X POST "$BASE_URL/assessments/$ASSESSMENT_ID/cancel" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
check_http "专家无法取消企业用户的评估（400/404）" "$CANCEL_STEAL_CODE" "400"

# ─── SECTION 8：专家收益查询 ──────────────────────────────────────────────
log_section "专家收益查询"

log_step "专家查询收益明细（GET /billing/earnings）"
EARNINGS=$(curl -s "$BASE_URL/billing/earnings" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
check_contains "收益接口可访问" "$EARNINGS" "total_earnings"
TOTAL_EARNINGS=$(jget "$EARNINGS" "['total_earnings']")
log_info "专家当前累计收益: $TOTAL_EARNINGS 元"
log_info "  （若评估尚未完成，收益可能为 0；完成后自动结算 30% 分成）"

log_step "专家查询余额变动记录（新注册无充值，仅可能有评估收益）"
EXPERT_TXNS=$(curl -s "$BASE_URL/billing/transactions" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
check_contains "余额变动记录接口可访问" "$EXPERT_TXNS" "items"

log_step "专家查询计费记录（自己的工具被调用产生的记录）"
EXPERT_BILL=$(curl -s "$BASE_URL/billing/records" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
check_contains "计费记录接口可访问" "$EXPERT_BILL" "items"

# ─── SECTION 9：资产废弃 ──────────────────────────────────────────────────
log_section "资产废弃（Deprecate）"

log_step "专家废弃已发布的工作流资产"
DEP_RESP=$(curl -s -w "\n%{http_code}" -X PUT "$BASE_URL/tools/$ASSET_ID/deprecate" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
DEP_BODY=$(echo "$DEP_RESP" | head -n1)
DEP_CODE=$(echo "$DEP_RESP" | tail -n1)
check_http "废弃资产成功" "$DEP_CODE" "200"
DEP_STATUS=$(jget "$DEP_BODY" "['status']")
check_eq "状态变为 deprecated" "$DEP_STATUS" "deprecated"

log_step "废弃后资产从公开市场下架（ListPublic 只返回 published）"
MARKET_AFTER_DEP=$(curl -s "$BASE_URL/tools" -H "Authorization: Bearer $ADMIN_TOKEN")
if echo "$MARKET_AFTER_DEP" | grep -q "$ASSET_ID" 2>/dev/null; then
    log_fail "废弃的资产不应出现在公开市场"
else
    log_ok "废弃资产已从公开市场下架"
fi

log_step "废弃资产在专家视图中仍可见（mine=1 返回全状态）"
MY_ASSETS_FINAL=$(curl -s "$BASE_URL/tools?mine=1" \
    -H "Authorization: Bearer $EXPERT_TOKEN")
check_contains "废弃资产在专家视图中可见" "$MY_ASSETS_FINAL" "deprecated"

# ─── SECTION 10：管理员平台统计 ──────────────────────────────────────────
log_section "管理员平台统计"

log_step "查看平台统计数据（GET /admin/stats）"
STATS=$(curl -s "$BASE_URL/admin/stats" -H "Authorization: Bearer $ADMIN_TOKEN")
check_contains "统计数据含 total_users" "$STATS" "total_users"
check_contains "统计数据含 total_assessments" "$STATS" "total_assessments"
TOTAL_USERS=$(jget "$STATS" "['total_users']")
TOTAL_ASSESSMENTS=$(jget "$STATS" "['total_assessments']")
log_info "平台总用户数: $TOTAL_USERS  总评估数: $TOTAL_ASSESSMENTS"

log_step "查看用户列表，按 expert 角色过滤"
EXPERT_USERS=$(curl -s "$BASE_URL/admin/users?role=expert" \
    -H "Authorization: Bearer $ADMIN_TOKEN")
check_contains "专家用户列表包含测试账号" "$EXPERT_USERS" "$EXPERT_EMAIL"

log_step "管理员通过 PUT /admin/users/:id/role 将用户升级为 expert 角色"
# 先注册一个 enterprise 用户，再升级为 expert
PROMOTE_EMAIL="promote_${TIMESTAMP}@test.com"
PROMOTE_REG=$(curl_json \
    "{\"email\":\"$PROMOTE_EMAIL\",\"password\":\"password123\",\"name\":\"待升级用户\",\"role\":\"enterprise\"}" \
    -s -w "\n%{http_code}" -X POST "$BASE_URL/auth/register" \
    -H "Content-Type: application/json")
PROMOTE_REG_CODE=$(echo "$PROMOTE_REG" | tail -n1)
PROMOTE_USER_ID=$(jget "$(echo "$PROMOTE_REG" | head -n1)" "['user_id']")
check_http "注册待升级用户" "$PROMOTE_REG_CODE" "201"

PROMOTE_RESP=$(curl -s -w "\n%{http_code}" \
    -X PUT "$BASE_URL/admin/users/$PROMOTE_USER_ID/role" \
    -H "Authorization: Bearer $ADMIN_TOKEN" \
    -H "Content-Type: application/json" \
    -d '{"role": "expert"}')
PROMOTE_CODE=$(echo "$PROMOTE_RESP" | tail -n1)
check_http "管理员升级用户角色为 expert" "$PROMOTE_CODE" "200"

# 验证升级后能用新 token 访问 expert 接口
PROMOTE_LOGIN=$(curl -s -X POST "$BASE_URL/auth/login" \
    -H "Content-Type: application/json" \
    -d "{\"email\":\"$PROMOTE_EMAIL\",\"password\":\"password123\"}")
PROMOTE_TOKEN=$(jget "$PROMOTE_LOGIN" "['token']")
PROMOTE_NEW_ROLE=$(jget "$PROMOTE_LOGIN" "['user']['role']")
check_eq "升级后重新登录角色为 expert" "$PROMOTE_NEW_ROLE" "expert"
PROMOTE_CREATE=$(curl_json \
    '{"name":"升级后创建","type":"tool_config","config":{"nodes":[],"edges":[]}}' \
    -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/tools" \
    -H "Authorization: Bearer $PROMOTE_TOKEN" \
    -H "Content-Type: application/json")
check_http "升级后的用户可创建资产（201）" "$PROMOTE_CREATE" "201"

# ─── 汇总 ─────────────────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}${BOLD}══════════════════════════════════════════${NC}"
echo -e "${BOLD}  安全专家流程验证结果汇总${NC}"
echo -e "${CYAN}${BOLD}══════════════════════════════════════════${NC}"
echo -e "  ${GREEN}通过: $PASS${NC}"
echo -e "  ${RED}失败: $FAIL${NC}"
echo -e "  总计: $((PASS + FAIL))"
echo ""

if [ "$FAIL" -eq 0 ]; then
    echo -e "${GREEN}${BOLD}  ✓ 全部通过！安全专家使用流程验证完成${NC}"
    exit 0
else
    echo -e "${RED}${BOLD}  ✗ 有 $FAIL 项失败，请检查上方日志${NC}"
    exit 1
fi
