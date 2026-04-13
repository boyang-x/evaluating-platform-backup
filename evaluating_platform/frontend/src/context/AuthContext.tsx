import { createContext, useContext, useState, useEffect, type ReactNode } from 'react'
import { authService, type User } from '../services/auth'

interface AuthContextValue {
  user: User | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => void
}

export const AuthContext = createContext<AuthContextValue>({
  user: null,
  loading: true,
  login: async () => {},
  logout: () => {},
})

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // 从 localStorage 恢复登录状态，然后从服务器刷新用户信息
    // 这样可以确保显示名称等字段始终是数据库中的最新值，
    // 避免因历史数据损坏（如乱码）导致界面显示错误。
    const stored = authService.getStoredUser()
    setUser(stored)

    if (stored && authService.getToken()) {
      authService.me()
        .then(fresh => setUser(fresh))
        .catch(() => {
          // token 失效，清理本地状态
          authService.logout()
          setUser(null)
        })
        .finally(() => setLoading(false))
    } else {
      setLoading(false)
    }
  }, [])

  const login = async (email: string, password: string) => {
    const res = await authService.login(email, password)
    setUser(res.user)
  }

  const logout = () => {
    authService.logout()
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  return useContext(AuthContext)
}
