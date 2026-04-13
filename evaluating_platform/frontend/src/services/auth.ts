import api from './api'

export interface User {
  id: string
  email: string
  name: string
  role: 'enterprise' | 'expert' | 'admin'
  org_name: string
  balance: number
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface LoginResponse {
  token: string
  user: User
}

export const authService = {
  async login(email: string, password: string): Promise<LoginResponse> {
    const res = await api.post<LoginResponse>('/auth/login', { email, password })
    const { token, user } = res.data
    localStorage.setItem('token', token)
    localStorage.setItem('user', JSON.stringify(user))
    return res.data
  },

  async register(data: { email: string; password: string; name: string; role: string; org_name?: string }): Promise<void> {
    await api.post('/auth/register', data)
  },

  async me(): Promise<User> {
    const res = await api.get<User>('/auth/me')
    const user = res.data
    localStorage.setItem('user', JSON.stringify(user))
    return user
  },

  logout(): void {
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  },

  getStoredUser(): User | null {
    try {
      const raw = localStorage.getItem('user')
      return raw ? JSON.parse(raw) : null
    } catch {
      return null
    }
  },

  getToken(): string | null {
    return localStorage.getItem('token')
  },
}
