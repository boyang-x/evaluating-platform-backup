import api from './api'
import { authService } from './auth'

export interface Assessment {
  id: string
  user_id: string
  name: string
  description: string
  goal: string
  target_id: string
  template_id?: string
  status: 'pending' | 'running' | 'completed' | 'failed' | 'canceled'
  plan: string[]
  error_msg: string
  created_at: string
  updated_at: string
  completed_at?: string
}

export interface LogEvent {
  assessment_id: string
  iteration: number
  tool_name: string
  output: string
  severity?: string
  tokens_used: number
  duration_ms: number
  timestamp: string
  type?: string
}

export interface CreateAssessmentData {
  name: string
  goal: string
  target_type: string
  target_url: string
  target_key?: string
  target_model?: string
  template_id?: string
  description?: string
}

export const assessmentService = {
  async create(data: CreateAssessmentData): Promise<{ assessment_id: string; status: string }> {
    const res = await api.post('/assessments', data)
    return res.data
  },

  async list(limit = 20, offset = 0): Promise<{ items: Assessment[]; total: number }> {
    const res = await api.get('/assessments', { params: { limit, offset } })
    return res.data
  },

  async get(id: string): Promise<{ assessment: Assessment; report?: unknown }> {
    const res = await api.get(`/assessments/${id}`)
    return res.data
  },

  async cancel(id: string): Promise<void> {
    await api.post(`/assessments/${id}/cancel`)
  },

  streamLogs(
    assessmentID: string,
    onLog: (event: LogEvent) => void,
    onDone: () => void,
  ): () => void {
    const token = authService.getToken()
    const base = (import.meta as unknown as { env: { VITE_API_BASE?: string } }).env.VITE_API_BASE || 'http://localhost:8080/api/v1'
    const url = `${base}/assessments/${assessmentID}/stream?token=${encodeURIComponent(token || '')}`
    const es = new EventSource(url)

    es.onmessage = (e) => {
      try {
        const data: LogEvent = JSON.parse(e.data)
        if (data.type === 'done') {
          onDone()
          es.close()
        } else {
          onLog(data)
        }
      } catch {
        // ignore malformed
      }
    }
    es.onerror = () => {
      onDone()
      es.close()
    }

    return () => es.close()
  },
}
