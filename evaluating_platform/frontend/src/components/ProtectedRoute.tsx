import { Navigate } from 'react-router-dom'
import { useAuth } from '../context/AuthContext'

const roleDefaultRoute: Record<string, string> = {
  enterprise: '/enterprise/dashboard',
  expert: '/expert/samples',
  admin: '/admin/dashboard',
}

interface ProtectedRouteProps {
  allowedRoles: string[]
  children: React.ReactNode
}

export function ProtectedRoute({ allowedRoles, children }: ProtectedRouteProps) {
  const { user, loading } = useAuth()

  if (loading) return null

  if (!user) {
    return <Navigate to="/login" replace />
  }

  if (!allowedRoles.includes(user.role)) {
    return <Navigate to={roleDefaultRoute[user.role] || '/login'} replace />
  }

  return <>{children}</>
}
