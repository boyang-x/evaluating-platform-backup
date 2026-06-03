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

export interface PlanInfo {
  name: string
  goal: string
  target_type: string
  target_url: string
  target_key: string
  target_model: string
  target_id?: string
  assessment_types: string[]
  resource_mode_preference?: string
  resource_handles?: string[]
  selected_skills?: string[]
  selected_capability_refs?: string[]
  selection_reasons?: string[]
  selection_strategy?: string
  test_count?: number
}

export interface WelcomeCapability {
  id: string
  label: string
  title?: string
  prompt: string
  tone?: 'attack' | 'governance' | 'engine' | 'tool'
  source_kind?: 'skill' | 'mcp' | 'resource'
  source_type?: string
  description?: string
  tags?: string[]
}
