import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const ts = require('typescript')

function loadTSModule(relativePath) {
  const filename = path.resolve(relativePath)
  const source = fs.readFileSync(filename, 'utf8')
  const compiled = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022,
      esModuleInterop: true,
    },
    fileName: filename,
  }).outputText
  const module = { exports: {} }
  const sandbox = { module, exports: module.exports, require, console }
  vm.runInNewContext(compiled, sandbox, { filename })
  return module.exports
}

const { parseRuntimePlan, runtimeContentHasPlanConfirm } = loadTSModule('src/services/maclawRuntimePlan.ts')
const { resolveReportSafetyScore } = loadTSModule('src/pages/enterprise/chatDisplay.ts')

const fencedPlan = `\`\`\`json
{
  "response_source": "plan_confirm",
  "target_summary": {
    "target_type": "企业门户中已配置连接的客服 LLM",
    "assessment_scope": "客服场景下的 jailbreak 与 prompt injection 风险评测"
  },
  "risk_types": [
    "jailbreak",
    { "risk": "prompt_injection", "description": "提示注入" }
  ],
  "selected_capability_refs": [
    { "source_type": "skill", "ref": "skillhub:ccbos-classical-chinese-skill" }
  ],
  "selected_skills": [
    { "skill_name": "ccbos-classical-chinese-skill" }
  ],
  "test_count": 3,
  "selection_strategy": "random",
  "requires_confirmation": true
}
\`\`\`

请确认是否执行该计划。`

assert.equal(runtimeContentHasPlanConfirm(fencedPlan), true)
const plan = parseRuntimePlan(fencedPlan)
assert.equal(plan?.target_type, '企业门户中已配置连接的客服 LLM')
assert.equal(plan?.test_count, 3)
assert.equal(plan?.selection_strategy, 'random')
assert.equal(JSON.stringify(plan?.assessment_types), JSON.stringify(['jailbreak', 'prompt_injection']))
assert.equal(plan?.resource_mode_preference, 'skill_generated')
assert.equal(JSON.stringify(plan?.resource_handles), JSON.stringify(['skillhub:ccbos-classical-chinese-skill']))
assert.match(plan?.goal || '', /客服场景/)

const nonPlan = '你好，我是红队测评智能体。'
assert.equal(runtimeContentHasPlanConfirm(nonPlan), false)
assert.equal(parseRuntimePlan(nonPlan), undefined)

const proseWrappedPlan = `信息已齐备。以下是评估计划：

\`\`\`json
{
  "response_source": "plan_confirm",
  "target_summary": "当前被测模型",
  "risk_types": ["jailbreak"],
  "selected_capability_refs": ["skillhub:ccbos-classical-chinese-skill"],
  "selected_skills": ["CCBOS Classical Chinese Skill"],
  "test_count": 5,
  "selection_strategy": "random",
  "requires_confirmation": true
}
\`\`\``
assert.equal(runtimeContentHasPlanConfirm(proseWrappedPlan), true)
const prosePlan = parseRuntimePlan(proseWrappedPlan)
assert.equal(prosePlan?.test_count, 5)
assert.equal(prosePlan?.resource_mode_preference, 'skill_generated')
assert.equal(JSON.stringify(prosePlan?.selected_skills), JSON.stringify(['CCBOS Classical Chinese Skill']))

assert.equal(resolveReportSafetyScore({
  cardType: 'report',
  directSafetyScore: 62.4,
  successCount: 3,
  failureCount: 2,
  executedCount: 5,
}), 62)
assert.equal(resolveReportSafetyScore({
  cardType: 'report',
  successCount: 3,
  failureCount: 2,
  executedCount: 5,
}), undefined)
assert.equal(resolveReportSafetyScore({
  cardType: 'progress',
  riskScore: 20,
}), 80)

console.log('maclaw runtime adapter tests passed')
