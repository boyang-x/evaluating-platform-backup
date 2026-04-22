import api from './api'

export interface AssetNode {
  id: string
  tool_name: string
  label: string
  params?: Record<string, unknown>
  position: { x: number; y: number }
}

export interface AssetEdge {
  id: string
  source: string
  target: string
}

export interface AssetConfig {
  nodes: AssetNode[]
  edges: AssetEdge[]
  params?: Record<string, unknown>
}

export interface Asset {
  id: string
  expert_id: string
  name: string
  description: string
  type: 'tool_config' | 'suite'
  visibility: 'private' | 'org' | 'public'
  status: 'draft' | 'testing' | 'published' | 'deprecated'
  version: string
  config: AssetConfig
  price_unit: number
  call_count: number
  created_at: string
  updated_at: string
}

export interface CreateAssetData {
  name: string
  description?: string
  type: 'tool_config' | 'suite'
  visibility?: string
  version?: string
  config?: AssetConfig
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

  async create(data: CreateAssetData): Promise<{ asset_id: string }> {
    const res = await api.post('/assets', data)
    return res.data
  },

  async deprecate(id: string): Promise<void> {
    await api.put(`/tools/${id}/deprecate`)
  },
}
