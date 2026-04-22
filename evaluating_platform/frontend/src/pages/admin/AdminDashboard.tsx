import { useEffect, useState } from 'react'
import { Row, Col, Card, Statistic, Typography, Button, Table, Tag } from 'antd'
import { TeamOutlined, SafetyOutlined, WalletOutlined, ArrowUpOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { adminService, type AdminStats } from '../../services/admin'
import { assessmentService, type Assessment } from '../../services/assessment'

const { Title, Text } = Typography

export function AdminDashboard() {
  const navigate = useNavigate()
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [recentAssessments, setRecentAssessments] = useState<Assessment[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    Promise.all([adminService.getStats(), assessmentService.list(10, 0)])
      .then(([nextStats, assessments]) => {
        setStats(nextStats)
        setRecentAssessments(assessments.items || [])
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [])

  const statusConfig: Record<string, { color: string; label: string }> = {
    completed: { color: 'success', label: '已完成' },
    running: { color: 'processing', label: '进行中' },
    failed: { color: 'error', label: '失败' },
    pending: { color: 'default', label: '待执行' },
    canceled: { color: 'warning', label: '已取消' },
  }

  const assessmentColumns = [
    {
      title: '任务名',
      dataIndex: 'name',
      render: (value: string) => <Text style={{ color: 'var(--text-primary)' }}>{value}</Text>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (value: string) => <Tag color={statusConfig[value]?.color}>{statusConfig[value]?.label || value}</Tag>,
    },
    {
      title: '时间',
      dataIndex: 'created_at',
      render: (value: string) => (
        <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>{value?.slice(0, 16).replace('T', ' ')}</Text>
      ),
    },
  ]

  const statCards = [
    {
      title: '累计用户',
      value: stats?.total_users ?? 0,
      icon: <TeamOutlined />,
      color: '#8b5cf6',
      suffix: '人',
    },
    {
      title: '累计评估',
      value: stats?.total_assessments ?? 0,
      icon: <SafetyOutlined />,
      color: '#4d96ff',
      suffix: '次',
    },
    {
      title: '平台总收入',
      value: stats?.total_revenue ?? 0,
      icon: <WalletOutlined />,
      color: '#52c41a',
      prefix: '¥',
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>
          平台概览
        </Title>
        <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>AI 安全评估平台运营状态</Text>
      </div>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        {statCards.map((stat) => (
          <Col span={8} key={stat.title}>
            <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)', borderRadius: 8 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <Statistic
                  title={<Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>{stat.title}</Text>}
                  value={stat.value}
                  prefix={stat.prefix}
                  suffix={
                    stat.suffix ? <Text style={{ color: 'var(--text-muted)', fontSize: 13 }}>{stat.suffix}</Text> : undefined
                  }
                  precision={stat.prefix ? 2 : 0}
                  valueStyle={{ color: stat.color, fontSize: 28, fontWeight: 700 }}
                />
                <div
                  style={{
                    width: 40,
                    height: 40,
                    borderRadius: 8,
                    background: `${stat.color}20`,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                  }}
                >
                  <span style={{ color: stat.color, fontSize: 18 }}>{stat.icon}</span>
                </div>
              </div>
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card
            title={<Text style={{ color: 'var(--text-primary)' }}>用户角色分布</Text>}
            style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
          >
            {stats ? (
              Object.entries(stats.users_by_role || {}).map(([role, count]) => (
                <div key={role} style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
                  <Tag color={role === 'admin' ? 'purple' : role === 'expert' ? 'orange' : 'blue'}>
                    {role === 'enterprise' ? '企业客户' : role === 'expert' ? '安全专家' : '管理员'}
                  </Tag>
                  <Text style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{count} 人</Text>
                </div>
              ))
            ) : (
              <Text style={{ color: 'var(--text-muted)' }}>加载中...</Text>
            )}
          </Card>
        </Col>

        <Col span={16}>
          <Card
            title={<Text style={{ color: 'var(--text-primary)' }}>最近评估任务</Text>}
            extra={
              <Button type="link" size="small" style={{ color: '#4d96ff' }} onClick={() => navigate('/enterprise/assessments')}>
                查看全部 <ArrowUpOutlined style={{ transform: 'rotate(45deg)' }} />
              </Button>
            }
            style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
          >
            <Table
              dataSource={recentAssessments}
              columns={assessmentColumns}
              rowKey="id"
              pagination={false}
              size="small"
              loading={loading}
              style={{ background: 'transparent' }}
            />
          </Card>
        </Col>
      </Row>
    </div>
  )
}
