import { useEffect, useState } from 'react'
import { Row, Col, Card, Statistic, Table, Tag, Button, Progress, Typography } from 'antd'
import {
  SafetyOutlined,
  ExclamationCircleOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  ArrowUpOutlined,
} from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { assessmentService, type Assessment } from '../../services/assessment'

const { Title, Text } = Typography

const statusConfig: Record<string, { color: string; label: string }> = {
  completed: { color: 'success', label: '已完成' },
  running: { color: 'processing', label: '进行中' },
  failed: { color: 'error', label: '失败' },
  pending: { color: 'default', label: '待执行' },
  canceled: { color: 'warning', label: '已取消' },
}


export function Dashboard() {
  const navigate = useNavigate()
  const [assessments, setAssessments] = useState<Assessment[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    assessmentService.list(100, 0).then(r => {
      setAssessments(r.items || [])
    }).catch(() => {}).finally(() => setLoading(false))
  }, [])

  const total = assessments.length
  const completed = assessments.filter(a => a.status === 'completed').length
  const running = assessments.filter(a => a.status === 'running' || a.status === 'pending').length
  const recent5 = assessments.slice(0, 5)

  const columns = [
    { title: '评估任务', dataIndex: 'name', key: 'name',
      render: (v: string) => <Text style={{ color: 'var(--text-primary)' }}>{v}</Text> },
    { title: '状态', dataIndex: 'status', key: 'status',
      render: (v: string) => <Tag color={statusConfig[v]?.color}>{statusConfig[v]?.label}</Tag> },
    { title: '日期', dataIndex: 'created_at', key: 'date',
      render: (v: string) => <Text style={{ color: 'var(--text-secondary)' }}>{v?.slice(0, 10)}</Text> },
    { title: '操作', key: 'action',
      render: (_: unknown, record: Assessment) => (
        <Button type="link" size="small" style={{ color: '#4d96ff', padding: 0 }}
          disabled={record.status !== 'completed'}
          onClick={() => navigate(`/enterprise/reports/${record.id}`)}>
          查看报告
        </Button>
      )
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 24, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>安全评估概览</Title>
          <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>实时监控 AI 系统安全态势</Text>
        </div>
        <Button type="primary" icon={<SafetyOutlined />}
          onClick={() => navigate('/enterprise/assessments/new')}
          style={{ height: 38 }}>
          发起新评估
        </Button>
      </div>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        {[
          { title: '累计评估', value: total, icon: <SafetyOutlined />, color: '#4d96ff', suffix: '次' },
          { title: '高危发现', value: 0, icon: <ExclamationCircleOutlined />, color: '#ff7a45', suffix: '项' },
          { title: '已完成', value: completed, icon: <CheckCircleOutlined />, color: '#52c41a', suffix: '次' },
          { title: '评估中', value: running, icon: <ClockCircleOutlined />, color: '#ffc53d', suffix: '个' },
        ].map((stat, i) => (
          <Col span={6} key={i}>
            <Card style={{
              background: 'var(--bg-card)',
              border: '1px solid var(--border-color)',
              borderRadius: 8,
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                <Statistic
                  title={<Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>{stat.title}</Text>}
                  value={stat.value}
                  suffix={<Text style={{ color: 'var(--text-muted)', fontSize: 13 }}>{stat.suffix}</Text>}
                  valueStyle={{ color: stat.color, fontSize: 28, fontWeight: 700 }}
                />
                <div style={{
                  width: 40, height: 40,
                  borderRadius: 8,
                  background: `${stat.color}20`,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                }}>
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
            title={<Text style={{ color: 'var(--text-primary)' }}>综合安全评分</Text>}
            style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)', height: '100%' }}
          >
            <div style={{ textAlign: 'center', padding: '12px 0' }}>
              <div style={{ fontSize: 56, fontWeight: 800, color: '#4d96ff', lineHeight: 1 }}>
                {completed > 0 ? Math.round(80 - (0 / completed) * 50) : '—'}
              </div>
              <Text style={{ color: 'var(--text-secondary)' }}>/ 100</Text>
              <div style={{ marginTop: 16, display: 'flex', gap: 8, flexDirection: 'column' }}>
                {[
                  { label: '评估完成率', score: total > 0 ? Math.round((completed / total) * 100) : 0 },
                  { label: '任务成功率', score: total > 0 ? Math.round((completed / total) * 100) : 0 },
                ].map(item => (
                  <div key={item.label}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                      <Text style={{ color: 'var(--text-secondary)', fontSize: 12 }}>{item.label}</Text>
                      <Text style={{ color: 'var(--text-primary)', fontSize: 12 }}>{item.score}%</Text>
                    </div>
                    <Progress
                      percent={item.score}
                      size="small"
                      showInfo={false}
                      strokeColor={item.score < 60 ? '#ff4d4f' : item.score < 80 ? '#ffc53d' : '#52c41a'}
                      trailColor='var(--border-color)'
                    />
                  </div>
                ))}
              </div>
            </div>
          </Card>
        </Col>

        <Col span={16}>
          <Card
            title={<Text style={{ color: 'var(--text-primary)' }}>近期评估记录</Text>}
            extra={<Button type="link" size="small" style={{ color: '#4d96ff' }}
              onClick={() => navigate('/enterprise/assessments')}>
              查看全部 <ArrowUpOutlined style={{ transform: 'rotate(45deg)' }} />
            </Button>}
            style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
          >
            <Table
              dataSource={recent5}
              columns={columns}
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
