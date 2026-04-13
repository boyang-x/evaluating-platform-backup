import { Routes, Route, Link, useLocation, useNavigate } from 'react-router-dom'
import { Layout, Menu, Typography, Avatar, Dropdown, Badge } from 'antd'
import { useEffect, useState } from 'react'
import {
  DashboardOutlined, TeamOutlined, AuditOutlined,
  LogoutOutlined, UserOutlined, BellOutlined, SafetyOutlined,
} from '@ant-design/icons'
import { AdminDashboard } from './AdminDashboard'
import { UserManagement } from './UserManagement'
import { AssetReview } from './AssetReview'
import { useAuth } from '../../context/AuthContext'
import { adminService } from '../../services/admin'

const { Sider, Header, Content } = Layout
const { Text } = Typography

export function AdminPortal() {
  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const [pendingCount, setPendingCount] = useState(0)
  const [stats, setStats] = useState<{ total_users?: number; total_assessments?: number } | null>(null)

  const loadStats = () => {
    adminService.getStats().then(s => {
      setPendingCount(s.pending_assets || 0)
      setStats(s)
    }).catch(() => {})
  }

  useEffect(() => {
    loadStats()
    // 每 15s 轮询待审数量
    const timer = setInterval(loadStats, 15000)
    return () => clearInterval(timer)
  }, [])

  const menuItems = [
    { key: '/admin/dashboard', icon: <DashboardOutlined />, label: '平台概览' },
    { key: '/admin/users', icon: <TeamOutlined />, label: '用户管理' },
    {
      key: '/admin/assets/review',
      icon: <AuditOutlined />,
      label: (
        <span>
          资产审核
          {pendingCount > 0 && (
            <Badge count={pendingCount} size="small" style={{ marginLeft: 8 }} />
          )}
        </span>
      ),
    },
  ]

  return (
    <Layout style={{ minHeight: '100vh', background: 'var(--bg-base)' }}>
      <Sider width={220} style={{
        background: 'var(--bg-surface)',
        borderRight: '1px solid var(--border-color)',
        position: 'fixed', height: '100vh',
        left: 0, top: 0, zIndex: 100,
      }}>
        <div style={{
          height: 64, display: 'flex', alignItems: 'center', paddingLeft: 20,
          borderBottom: '1px solid var(--border-color)', gap: 10,
        }}>
          <div style={{
            width: 32, height: 32, borderRadius: 8,
            background: 'rgba(139, 92, 246, 0.15)',
            border: '1px solid rgba(139, 92, 246, 0.4)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
            <SafetyOutlined style={{ color: '#8b5cf6', fontSize: 16 }} />
          </div>
          <div>
            <div style={{ color: 'var(--text-primary)', fontWeight: 700, fontSize: 14, lineHeight: 1.2 }}>
              AI 安全评估
            </div>
            <div style={{ color: 'var(--text-muted)', fontSize: 11 }}>系统管理员控制台</div>
          </div>
        </div>

        <Menu
          theme="dark" mode="inline"
          selectedKeys={[location.pathname]}
          style={{ background: 'transparent', border: 'none', marginTop: 8 }}
          items={menuItems.map(item => ({
            key: item.key,
            icon: item.icon,
            label: <Link to={item.key}>{item.label}</Link>,
          }))}
        />

        {/* 底部统计 */}
        <div style={{
          position: 'absolute', bottom: 20, left: 12, right: 12,
          padding: '12px 16px',
          background: 'var(--bg-card)', border: '1px solid var(--border-color)', borderRadius: 8,
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <div>
              <Text style={{ color: 'var(--text-muted)', fontSize: 11 }}>用户总数</Text>
              <div style={{ color: '#8b5cf6', fontSize: 16, fontWeight: 700 }}>{stats?.total_users ?? '—'}</div>
            </div>
            <div>
              <Text style={{ color: 'var(--text-muted)', fontSize: 11 }}>总评估</Text>
              <div style={{ color: '#4d96ff', fontSize: 16, fontWeight: 700 }}>{stats?.total_assessments ?? '—'}</div>
            </div>
          </div>
        </div>
      </Sider>

      <Layout style={{ marginLeft: 220 }}>
        <Header style={{
          background: 'var(--bg-surface)', borderBottom: '1px solid var(--border-color)',
          padding: '0 24px', display: 'flex', alignItems: 'center', justifyContent: 'space-between',
          position: 'sticky', top: 0, zIndex: 99,
        }}>
          <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>系统管理员后台</Text>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Badge count={pendingCount} size="small">
              <BellOutlined style={{ color: 'var(--text-secondary)', fontSize: 18, cursor: 'pointer' }}
                onClick={() => navigate('/admin/assets/review')} />
            </Badge>
            <Dropdown menu={{
              items: [{
                key: 'logout', icon: <LogoutOutlined />, label: '退出',
                onClick: () => { logout(); navigate('/login') }
              }]
            }} placement="bottomRight">
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
                <Avatar size={32} icon={<UserOutlined />}
                  style={{ background: 'rgba(139,92,246,0.2)', border: '1px solid rgba(139,92,246,0.4)' }}
                />
                <Text style={{ color: 'var(--text-primary)', fontSize: 13 }}>
                  {user?.name || '管理员'}
                </Text>
              </div>
            </Dropdown>
          </div>
        </Header>

        <Content style={{ padding: 24, minHeight: 'calc(100vh - 64px)' }}>
          <Routes>
            <Route path="dashboard" element={<AdminDashboard />} />
            <Route path="users" element={<UserManagement />} />
            <Route path="assets/review" element={<AssetReview />} />
            <Route path="*" element={<AdminDashboard />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  )
}
