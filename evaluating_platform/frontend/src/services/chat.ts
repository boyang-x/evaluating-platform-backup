import api from './api'

export interface ChatSession {
  id: string
  user_id: string
  title: string
  state: 'idle' | 'collecting_intent' | 'confirming_plan' | 'running' | 'completed' | 'failed'
  assessment_id?: string
  target_info: Record<string, unknown>
  plan_info: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface ChatMessage {
  id: string
  session_id: string
  role: 'user' | 'assistant' | 'system' | 'tool_event'
  content: string
  metadata: {
    card_type?: 'text' | 'plan_confirm' | 'progress' | 'report' | 'skill_launch'
    plan?: PlanInfo
    assessment_id?: string
    risk_level?: string
    status_text?: string
    phase?: string
    executed_count?: number
    planned_count?: number
    pdf_url?: string
    skill_id?: string
    skill_name?: string
    open_url?: string
    [key: string]: unknown
  }
  created_at: string
}

export interface SkillLaunchDocument {
  skill_id: string
  skill_name: string
  skill_slug: string
  version_id: string
  version: string
  document_title: string
  html: string
  launch_mode: string
}

export interface PlanInfo {
  name: string
  goal: string
  target_type: string
  target_url: string
  target_key: string
  target_model: string
  assessment_types: string[]
  resource_mode_preference?: string
  test_count?: number
}

export interface WelcomeCapability {
  id: string
  label: string
  prompt: string
  tone: 'attack' | 'governance' | 'engine' | 'tool'
  source_kind: 'skill' | 'mcp' | 'resource'
  source_type?: string
  description?: string
}

export const chatService = {
  async createSession(): Promise<ChatSession> {
    const res = await api.post('/chat/sessions')
    return res.data
  },

  async listSessions(limit = 50, offset = 0): Promise<{ items: ChatSession[]; total: number }> {
    const res = await api.get('/chat/sessions', { params: { limit, offset } })
    return res.data
  },

  async getSession(id: string): Promise<{ session: ChatSession; messages: ChatMessage[] }> {
    const res = await api.get(`/chat/sessions/${id}`)
    return res.data
  },

  async sendMessage(sessionId: string, content: string): Promise<ChatMessage> {
    const res = await api.post(`/chat/sessions/${sessionId}/messages`, { content })
    return res.data
  },

  async confirmPlan(sessionId: string, testCount?: number): Promise<{ session: ChatSession; plan: PlanInfo }> {
    const res = await api.post(`/chat/sessions/${sessionId}/confirm`, testCount ? { test_count: testCount } : {})
    return res.data
  },

  async deleteSession(id: string): Promise<void> {
    await api.delete(`/chat/sessions/${id}`)
  },

  async getSkillLaunchDocument(id: string): Promise<SkillLaunchDocument> {
    const res = await api.get(`/enterprise/skills/${id}/launch`)
    return res.data.item
  },

  async getWelcomeCapabilities(limit = 6): Promise<WelcomeCapability[]> {
    const res = await api.get('/enterprise/welcome-capabilities', { params: { limit } })
    return res.data.items || []
  },
}
