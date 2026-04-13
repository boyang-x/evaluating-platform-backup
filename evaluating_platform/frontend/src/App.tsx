import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthProvider } from './context/AuthContext'
import { ProtectedRoute } from './components/ProtectedRoute'
import { Login } from './pages/Login'
import { EnterprisePortal } from './pages/enterprise/EnterprisePortal'
import { ExpertPortal } from './pages/expert/ExpertPortal'
import { AdminPortal } from './pages/admin/AdminPortal'

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/" element={<Navigate to="/login" replace />} />
          <Route path="/login" element={<Login />} />

          <Route
            path="/enterprise/*"
            element={
              <ProtectedRoute allowedRoles={['enterprise', 'admin']}>
                <EnterprisePortal />
              </ProtectedRoute>
            }
          />

          <Route
            path="/expert/*"
            element={
              <ProtectedRoute allowedRoles={['expert', 'admin']}>
                <ExpertPortal />
              </ProtectedRoute>
            }
          />

          <Route
            path="/admin/*"
            element={
              <ProtectedRoute allowedRoles={['admin']}>
                <AdminPortal />
              </ProtectedRoute>
            }
          />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}

export default App
