import { Routes, Route, Link, useLocation, useNavigate } from 'react-router-dom'
import { Layout, Menu, Typography, Button, Badge, Avatar, Dropdown, Segmented } from 'antd'
import { useEffect, useState } from 'react'
import {
  DashboardOutlined,
  SafetyOutlined,
  FileTextOutlined,
  SettingOutlined,
  LogoutOutlined,
  UserOutlined,
  BellOutlined,
  ShopOutlined,
  WalletOutlined,
  RobotOutlined,
  AppstoreOutlined,
} from '@ant-design/icons'
import { Dashboard } from './Dashboard'
import { NewAssessment } from './NewAssessment'
import { AssessmentList } from './AssessmentList'
import { ReportView } from './ReportView'
import { BillingPage } from './BillingPage'
import { MarketPlace } from './MarketPlace'
import { Settings } from './Settings'
import { ChatPage } from './ChatPage'
import { EnterpriseSkillOpen } from './EnterpriseSkillOpen'
import { useAuth } from '../../context/AuthContext'
import { billingService } from '../../services/billing'
import appLogoUrl from '../../assets/qianxin-ai-security-logo.png'

const { Sider, Header, Content } = Layout
const { Text } = Typography

const menuItems = [
  { key: '/enterprise/dashboard', icon: <DashboardOutlined />, label: '概览' },
  { key: '/enterprise/assessments/new', icon: <SafetyOutlined />, label: '发起评估' },
  { key: '/enterprise/assessments', icon: <FileTextOutlined />, label: '评估记录' },
  { key: '/enterprise/market', icon: <ShopOutlined />, label: '资产市场' },
  { key: '/enterprise/billing', icon: <WalletOutlined />, label: '计费管理' },
  { key: '/enterprise/settings', icon: <SettingOutlined />, label: '账户设置' },
]

type Mode = 'classic' | 'ai'

export function EnterprisePortal() {
  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout } = useAuth()
  const [balance, setBalance] = useState<number | null>(null)
  const [mode, setMode] = useState<Mode>('ai')

  useEffect(() => {
    billingService.getBalance().then(r => setBalance(r.balance)).catch(() => {})
  }, [])

  const skillOpenMatch = location.pathname.match(/^\/enterprise\/skills\/([^/]+)\/open$/)
  if (skillOpenMatch) {
    return <EnterpriseSkillOpen skillId={skillOpenMatch[1]} />
  }

  const userMenu = [
    {
      key: 'logout',
      icon: <LogoutOutlined />,
      label: '退出登录',
      onClick: () => {
        logout()
        navigate('/login')
      },
    },
  ]

  return (
    <Layout style={{ minHeight: '100vh', background: 'var(--bg-base)' }}>
      <Sider
        width={220}
        style={{
          background: 'var(--bg-surface)',
          borderRight: '1px solid var(--border-color)',
          position: 'fixed',
          height: '100vh',
          left: 0, top: 0,
          zIndex: 100,
        }}
      >
        <div style={{
          height: 64,
          display: 'flex',
          alignItems: 'center',
          paddingLeft: 20,
          borderBottom: '1px solid var(--border-color)',
          gap: 10,
        }}>
          <div style={{
            width: 42, height: 42,
            borderRadius: '50%',
            background: 'transparent',
            overflow: 'hidden',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
            <img
              src={appLogoUrl}
              alt="Qianxin China-ASEAN AI Security Research Institute"
              style={{ width: '100%', height: '100%', objectFit: 'contain' }}
            />
          </div>
          <div>
            <div style={{ color: 'var(--text-primary)', fontWeight: 700, fontSize: 14, lineHeight: 1.2 }}>
              AI 安全评估
            </div>
            <div style={{ color: 'var(--text-muted)', fontSize: 11 }}>企业客户门户</div>
          </div>
        </div>

        {/* 模式切换 */}
        <div style={{ padding: '12px 12px 4px' }}>
          <Segmented
            block
            value={mode}
            onChange={v => setMode(v as Mode)}
            options={[
              { value: 'ai', icon: <RobotOutlined />, label: 'AI 服务' },
              { value: 'classic', icon: <AppstoreOutlined />, label: '传统服务' },
            ]}
            style={{ fontSize: 12 }}
          />
        </div>

        {mode === 'classic' && (
          <Menu
            theme="dark"
            mode="inline"
            selectedKeys={[location.pathname]}
            style={{ background: 'transparent', border: 'none', marginTop: 4 }}
            items={menuItems.map(item => ({
              key: item.key,
              icon: item.icon,
              label: <Link to={item.key}>{item.label}</Link>,
            }))}
          />
        )}

        {/* 余额显示 */}
        <div style={{
          position: 'absolute',
          bottom: 20,
          left: 12,
          right: 12,
          padding: '12px 16px',
          background: 'var(--bg-card)',
          border: '1px solid var(--border-color)',
          borderRadius: 8,
        }}>
          <Text style={{ color: 'var(--text-muted)', fontSize: 11 }}>账户余额</Text>
          <div style={{ color: '#4d96ff', fontSize: 18, fontWeight: 700, fontVariantNumeric: 'tabular-nums' }}>
            {balance !== null ? `¥ ${balance.toFixed(2)}` : '—'}
          </div>
          <Button type="primary" size="small" block style={{ marginTop: 8, fontSize: 12 }}
            onClick={() => navigate('/enterprise/billing')}>
            充值
          </Button>
        </div>
      </Sider>

      <Layout style={{ marginLeft: 220 }}>
        <Header style={{
          background: 'var(--bg-surface)',
          borderBottom: '1px solid var(--border-color)',
          padding: '0 24px',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          position: 'sticky',
          top: 0,
          zIndex: 99,
        }}>
          <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
            {new Date().toLocaleDateString('zh-CN', { year: 'numeric', month: 'long', day: 'numeric' })}
          </Text>

          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <Badge count={0} size="small">
              <BellOutlined style={{ color: 'var(--text-secondary)', fontSize: 18, cursor: 'pointer' }} />
            </Badge>
            <Dropdown menu={{ items: userMenu }} placement="bottomRight">
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer' }}>
                <Avatar size={32} icon={<UserOutlined />}
                  style={{ background: 'rgba(26, 109, 255, 0.2)', border: '1px solid rgba(26, 109, 255, 0.4)' }}
                />
                <Text style={{ color: 'var(--text-primary)', fontSize: 13 }}>
                  {user?.name || user?.email || '用户'}
                </Text>
              </div>
            </Dropdown>
          </div>
        </Header>

        <Content style={{ padding: mode === 'ai' ? 0 : 24, minHeight: 'calc(100vh - 64px)' }}>
          {mode === 'ai' ? (
            <ChatPage />
          ) : (
            <Routes>
              <Route path="dashboard" element={<Dashboard />} />
              <Route path="assessments/new" element={<NewAssessment />} />
              <Route path="assessments" element={<AssessmentList />} />
              <Route path="reports/:id" element={<ReportView />} />
              <Route path="market" element={<MarketPlace />} />
              <Route path="billing" element={<BillingPage />} />
              <Route path="settings" element={<Settings />} />
              <Route path="*" element={<Dashboard />} />
            </Routes>
          )}
        </Content>
      </Layout>
    </Layout>
  )
}
