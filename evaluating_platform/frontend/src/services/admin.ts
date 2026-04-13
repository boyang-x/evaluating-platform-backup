import api from './api'
import type { User } from './auth'
import type { Asset } from './asset'

export interface AdminStats {
  total_users: number
  users_by_role: Record<string, number>
  total_assessments: number
  pending_assets: number
  total_revenue: number
  weekly_assessments: number[]
}

export const adminService = {
  async listUsers(params?: {
    limit?: number
    offset?: number
    role?: string
    keyword?: string
  }): Promise<{ items: User[]; total: number }> {
    const res = await api.get('/admin/users', { params })
    return res.data
  },

  async updateUserRole(id: string, role: string): Promise<void> {
    await api.put(`/admin/users/${id}/role`, { role })
  },

  async setUserActive(id: string, active: boolean): Promise<void> {
    await api.put(`/admin/users/${id}/active`, { active })
  },

  async listPendingAssets(limit = 20, offset = 0): Promise<{ items: Asset[]; total: number }> {
    const res = await api.get('/admin/assets/pending', { params: { limit, offset } })
    return res.data
  },

  async approveAsset(id: string): Promise<void> {
    await api.put(`/admin/assets/${id}/approve`)
  },

  async rejectAsset(id: string, reason?: string): Promise<void> {
    await api.put(`/admin/assets/${id}/reject`, { reason })
  },

  async getStats(): Promise<AdminStats> {
    const res = await api.get<AdminStats>('/admin/stats')
    return res.data
  },
}
