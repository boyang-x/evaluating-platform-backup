import api from './api'

export interface WorkflowNode {
  id: string
  tool_name: string
  label: string
  params?: Record<string, unknown>
  position: { x: number; y: number }
}

export interface WorkflowEdge {
  id: string
  source: string
  target: string
}

export interface WorkflowConfig {
  nodes: WorkflowNode[]
  edges: WorkflowEdge[]
  params?: Record<string, unknown>
}

export interface Asset {
  id: string
  expert_id: string
  name: string
  description: string
  type: 'tool_config' | 'workflow' | 'suite'
  visibility: 'private' | 'org' | 'public'
  status: 'draft' | 'testing' | 'published' | 'deprecated'
  version: string
  config: WorkflowConfig
  price_unit: number
  call_count: number
  created_at: string
  updated_at: string
}

export interface CreateAssetData {
  name: string
  description?: string
  type: string
  visibility?: string
  version?: string
  config?: WorkflowConfig
  price_unit?: number
}

export const assetService = {
  async listPublic(type?: string, limit = 20, offset = 0): Promise<{ items: Asset[]; total: number }> {
    const res = await api.get('/tools', { params: { type, limit, offset } })
    return res.data
  },

  async listMine(limit = 50, offset = 0): Promise<{ items: Asset[]; total: number }> {
    const res = await api.get('/assets', { params: { mine: 1, limit, offset } })
    return res.data
  },

  async get(id: string): Promise<Asset> {
    const res = await api.get<Asset>(`/tools/${id}`)
    return res.data
  },

  async create(data: CreateAssetData): Promise<{ asset_id: string }> {
    const res = await api.post('/assets', data)
    return res.data
  },

  async update(id: string, data: Partial<CreateAssetData>): Promise<void> {
    await api.put(`/tools/${id}`, data)
  },

  async publish(id: string): Promise<void> {
    await api.put(`/tools/${id}/publish`)
  },

  async submitForReview(id: string): Promise<void> {
    await api.put(`/tools/${id}/submit`)
  },

  async deprecate(id: string): Promise<void> {
    await api.put(`/tools/${id}/deprecate`)
  },
}
