import { useEffect, useState } from 'react'
import { Form, Input, Button, Typography, App } from 'antd'
import { useNavigate } from 'react-router-dom'
import { LockOutlined, UserOutlined, SafetyCertificateOutlined } from '@ant-design/icons'
import { useAuth } from '../context/AuthContext'

const { Title, Text } = Typography

const roleRoute: Record<string, string> = {
  enterprise: '/enterprise/dashboard',
  expert: '/expert/samples',
  admin: '/admin/dashboard',
}

export function Login() {
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const { login, user } = useAuth()
  const { message } = App.useApp()

  useEffect(() => {
    if (user) {
      navigate(roleRoute[user.role] || '/enterprise/dashboard', { replace: true })
    }
  }, [navigate, user])

  if (user) {
    return null
  }

  const onFinish = async (values: { email: string; password: string }) => {
    setLoading(true)
    try {
      await login(values.email, values.password)
      message.success('登录成功')
      const stored = JSON.parse(localStorage.getItem('user') || '{}')
      navigate(roleRoute[stored.role] || '/enterprise/dashboard', { replace: true })
    } catch (err: unknown) {
      message.error((err as Error).message || '登录失败，请检查账号密码')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'var(--bg-base)',
      backgroundImage: `
        linear-gradient(rgba(26, 109, 255, 0.04) 1px, transparent 1px),
        linear-gradient(90deg, rgba(26, 109, 255, 0.04) 1px, transparent 1px)
      `,
      backgroundSize: '40px 40px',
    }}>
      <div style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        height: '2px',
        background: 'linear-gradient(90deg, transparent, var(--color-primary), transparent)',
      }} />

      <div style={{
        width: 420,
        padding: '48px 40px',
        background: 'var(--bg-card)',
        border: '1px solid var(--border-color)',
        borderRadius: 12,
        boxShadow: '0 0 60px rgba(26, 109, 255, 0.1), 0 20px 60px rgba(0,0,0,0.8)',
      }}>
        <div style={{ textAlign: 'center', marginBottom: 36 }}>
          <div style={{
            display: 'inline-flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: 56,
            height: 56,
            borderRadius: 14,
            background: 'rgba(26, 109, 255, 0.15)',
            border: '1px solid rgba(26, 109, 255, 0.4)',
            marginBottom: 16,
          }}>
            <SafetyCertificateOutlined style={{ fontSize: 28, color: '#4d96ff' }} />
          </div>
          <Title level={3} style={{ color: 'var(--text-primary)', margin: 0, letterSpacing: 1 }}>
            AI 安全评估平台
          </Title>
          <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
            AI Security Evaluation Platform
          </Text>
        </div>

        <Form name="login" onFinish={onFinish} layout="vertical" size="large">
          <Form.Item
            name="email"
            label={<span style={{ color: 'var(--text-secondary)' }}>邮箱</span>}
            rules={[{ required: true, message: '请输入邮箱' }, { type: 'email', message: '邮箱格式不正确' }]}
          >
            <Input prefix={<UserOutlined style={{ color: 'var(--text-muted)' }} />} placeholder="your@email.com" autoComplete="email" />
          </Form.Item>

          <Form.Item
            name="password"
            label={<span style={{ color: 'var(--text-secondary)' }}>密码</span>}
            rules={[{ required: true, message: '请输入密码' }]}
          >
            <Input.Password prefix={<LockOutlined style={{ color: 'var(--text-muted)' }} />} placeholder="••••••••" autoComplete="current-password" />
          </Form.Item>

          <Form.Item style={{ marginTop: 24, marginBottom: 0 }}>
            <Button type="primary" htmlType="submit" loading={loading} block style={{ height: 44, fontSize: 15, letterSpacing: 2 }}>
              登 录
            </Button>
          </Form.Item>
        </Form>

        <div style={{ textAlign: 'center', marginTop: 20 }}>
          <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>
            还没有账号？ <a href="#" style={{ color: 'var(--text-link)' }}>联系管理员开通</a>
          </Text>
        </div>
      </div>
    </div>
  )
}
