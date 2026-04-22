import type { ChatMessage, ChatSession, PlanInfo, WelcomeCapability } from '../../services/chat'

export const DEFAULT_WELCOME_CAPABILITIES: WelcomeCapability[] = [
  { id: 'default-prompt-injection', label: '提示注入检验', prompt: '请帮我测试目标应用的提示词注入攻击风险。', tone: 'attack', source_kind: 'resource', source_type: 'fallback', description: '围绕提示词注入场景设计评估。' },
  { id: 'default-agent-security', label: 'Agent 安全评估', prompt: '请帮我评估这个 Agent 的安全性。', tone: 'tool', source_kind: 'resource', source_type: 'fallback', description: '围绕 Agent 场景进行安全评估。' },
  { id: 'default-jailbreak', label: '越狱风险检测', prompt: '请帮我检测目标应用的越狱攻击风险。', tone: 'engine', source_kind: 'resource', source_type: 'fallback', description: '围绕越狱与绕过风险设计评估。' },
  { id: 'default-compliance', label: '内容合规检查', prompt: '请帮我做一轮内容安全与合规性检查。', tone: 'governance', source_kind: 'resource', source_type: 'fallback', description: '围绕内容安全与合规场景设计评估。' },
]

export const LABELS: Record<string, string> = {
  prompt_injection: '提示词注入',
  jailbreak: '越狱攻击',
  goal_hijacking: '目标劫持',
  tool_poisoning: '工具投毒',
  compliance_check: '合规检查',
}

export const RESOURCE_MODE_LABELS: Record<string, string> = {
  sample_template: '样本+模板组合',
  sample_rewrite: '样本改写后执行',
  skill_generated: '已发布 Skill 生成',
  composed_attack: '已组合攻击',
}

const EMPTY_PLAN_TEXTS = new Set(['<nil>', 'nil', 'null', 'undefined'])

function isMeaningfulDate(value?: string) {
  if (!value) return false
  const parsed = new Date(value)
  return !Number.isNaN(parsed.getTime()) && parsed.getFullYear() > 1
}

export function normalizeSessionTimestamps(session: ChatSession): ChatSession {
  const now = new Date().toISOString()
  const createdAt = isMeaningfulDate(session.created_at) ? session.created_at : now
  const updatedAt = isMeaningfulDate(session.updated_at) ? session.updated_at : createdAt
  return { ...session, created_at: createdAt, updated_at: updatedAt }
}

export function formatSessionDate(value?: string) {
  if (!isMeaningfulDate(value)) return '刚刚'
  return new Date(value ?? '').toLocaleDateString('zh-CN')
}

function isMeaningfulPlanText(value?: string | null) {
  const text = (value ?? '').trim()
  return text !== '' && !EMPTY_PLAN_TEXTS.has(text.toLowerCase())
}

function fallbackPlanGoal(plan: Partial<PlanInfo>) {
  const assessmentText = (plan.assessment_types ?? [])
    .map(item => LABELS[item] || item)
    .filter(Boolean)
    .join('、') || '安全评测'

  let resourceText = '系统将根据当前可用能力自动选择合适资源并执行测试。'
  switch (plan.resource_mode_preference) {
    case 'sample_template':
      resourceText = '系统将使用样本与模板组合生成测试问题。'
      break
    case 'sample_rewrite':
      resourceText = '系统将基于专家样本完成改写后执行测试。'
      break
    case 'skill_generated':
      resourceText = '系统将优先使用已发布 Skill 生成测试载荷。'
      break
    case 'composed_attack':
      resourceText = '系统将优先使用已组合攻击载荷执行测试。'
      break
  }
  return `围绕${assessmentText}场景开展安全评测。${resourceText}`
}

export function normalizePlanForDisplay(plan: PlanInfo): PlanInfo {
  const goal = isMeaningfulPlanText(plan.goal) ? plan.goal.trim() : fallbackPlanGoal(plan)
  const name = isMeaningfulPlanText(plan.name) ? plan.name.trim() : 'AI 安全评测计划'
  const targetType = isMeaningfulPlanText(plan.target_type) ? plan.target_type.trim() : 'openai'
  const targetUrl = isMeaningfulPlanText(plan.target_url) ? plan.target_url.trim() : ''
  const targetModel = isMeaningfulPlanText(plan.target_model) ? plan.target_model.trim() : ''
  return {
    ...plan,
    name,
    goal,
    target_type: targetType,
    target_url: targetUrl,
    target_model: targetModel,
  }
}

function normalizeRiskScore(value?: number) {
  if (typeof value !== 'number' || Number.isNaN(value)) return 0
  return Math.min(100, Math.max(0, value))
}

export function safetyScoreFromRiskScore(value?: number) {
  return Math.round(100 - normalizeRiskScore(value))
}

function sessionPlanInfo(session?: ChatSession | null) {
  if (!session || !session.plan_info || typeof session.plan_info !== 'object') return null
  return session.plan_info as unknown as Partial<PlanInfo>
}

function syncPlanMessagesWithSession(messages: ChatMessage[], session?: ChatSession | null) {
  const plan = sessionPlanInfo(session)
  if (!plan || Object.keys(plan).length === 0) {
    return messages.map(msg => {
      if (msg.metadata?.card_type !== 'plan_confirm' || !msg.metadata?.plan) return msg
      return {
        ...msg,
        metadata: {
          ...msg.metadata,
          plan: normalizePlanForDisplay(msg.metadata.plan as PlanInfo),
        },
      }
    })
  }

  let lastPlanIndex = -1
  messages.forEach((msg, index) => {
    if (msg.metadata?.card_type === 'plan_confirm' && msg.metadata?.plan) {
      lastPlanIndex = index
    }
  })
  if (lastPlanIndex < 0) return messages

  return messages.map((msg, index) => {
    if (index !== lastPlanIndex || msg.metadata?.card_type !== 'plan_confirm' || !msg.metadata?.plan) return msg
    return {
      ...msg,
      metadata: {
        ...msg.metadata,
        plan: normalizePlanForDisplay({
          ...(msg.metadata.plan as PlanInfo),
          ...plan,
        } as PlanInfo),
      },
    }
  })
}

function collapseProgressMessages(messages: ChatMessage[]) {
  const latestProgressIndex = new Map<string, number>()
  const assessmentsWithReport = new Set<string>()

  messages.forEach((msg, index) => {
    const cardType = msg.metadata?.card_type
    const assessmentId = typeof msg.metadata?.assessment_id === 'string' ? msg.metadata.assessment_id : ''
    if (!assessmentId) return
    if (cardType === 'progress') latestProgressIndex.set(assessmentId, index)
    if (cardType === 'report') assessmentsWithReport.add(assessmentId)
  })

  return messages.filter((msg, index) => {
    if (msg.metadata?.card_type !== 'progress') return true
    const assessmentId = typeof msg.metadata?.assessment_id === 'string' ? msg.metadata.assessment_id : ''
    if (!assessmentId) return true
    if (assessmentsWithReport.has(assessmentId)) return false
    return latestProgressIndex.get(assessmentId) === index
  })
}

export function prepareMessagesForDisplay(messages: ChatMessage[], session?: ChatSession | null) {
  return collapseProgressMessages(syncPlanMessagesWithSession(messages, session))
}
