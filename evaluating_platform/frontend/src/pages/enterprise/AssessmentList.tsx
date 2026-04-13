import { useEffect, useState, useRef } from 'react'
import {
  Table, Tag, Button, Input, Select, Space, Typography, Badge,
  Drawer, Spin, Empty
} from 'antd'
import { SearchOutlined, ReloadOutlined, EyeOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { assessmentService, type Assessment, type LogEvent } from '../../services/assessment'

const { Text } = Typography

const statusBadge: Record<string, React.ComponentProps<typeof Badge>['status']> = {
  completed: 'success',
  running: 'processing',
  failed: 'error',
  pending: 'default',
  canceled: 'warning',
}

const statusLabel: Record<string, string> = {
  completed: '已完成', running: '进行中', failed: '失败', pending: '等待中', canceled: '已取消',
}

const riskColor: Record<string, string> = {
  critical: '#ff4d4f', high: '#ff7a45', medium: '#ffc53d', low: '#52c41a',
}

const riskLabel: Record<string, string> = {
  critical: '严重', high: '高危', medium: '中危', low: '低危',
}

export function AssessmentList() {
  const navigate = useNavigate()
  const [items, setItems] = useState<Assessment[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [statusFilter, setStatusFilter] = useState('all')
  const [keyword, setKeyword] = useState('')
  const [drawerOpen, setDrawerOpen] = useState(false)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [logs, setLogs] = useState<LogEvent[]>([])
  const [streamDone, setStreamDone] = useState(false)
  const sseCloseRef = useRef<(() => void) | null>(null)
  const logsEndRef = useRef<HTMLDivElement>(null)

  const pageSize = 20

  const fetchList = () => {
    setLoading(true)
    assessmentService.list(pageSize, (page - 1) * pageSize).then(r => {
      setItems(r.items || [])
      setTotal(r.total || 0)
    }).catch(() => {}).finally(() => setLoading(false))
  }

  useEffect(() => { fetchList() }, [page])

  // 自动轮询：有 pending/running 任务时每 5s 刷新
  useEffect(() => {
    const hasActive = items.some(a => a.status === 'pending' || a.status === 'running')
    if (!hasActive) return
    const timer = setInterval(fetchList, 5000)
    return () => clearInterval(timer)
  }, [items])

  // 当 logs 变化时自动滚动到底部
  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  const openDrawer = (id: string) => {
    setSelectedId(id)
    setLogs([])
    setStreamDone(false)
    setDrawerOpen(true)

    // 关闭旧的 SSE 连接
    sseCloseRef.current?.()

    const close = assessmentService.streamLogs(
      id,
      (event) => setLogs(prev => [...prev, event]),
      () => setStreamDone(true),
    )
    sseCloseRef.current = close
  }

  const closeDrawer = () => {
    sseCloseRef.current?.()
    sseCloseRef.current = null
    setDrawerOpen(false)
    setSelectedId(null)
    setLogs([])
  }

  const filtered = items.filter(a => {
    if (statusFilter !== 'all' && a.status !== statusFilter) return false
    if (keyword && !a.name.toLowerCase().includes(keyword.toLowerCase())) return false
    return true
  })

  const selectedAssessment = items.find(a => a.id === selectedId)

  const columns = [
    {
      title: '评估任务',
      dataIndex: 'name',
      render: (v: string, r: Assessment) => (
        <div>
          <Text style={{ color: 'var(--text-primary)', display: 'block' }}>{v}</Text>
          <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>{r.id.slice(0, 8)}...</Text>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (v: string) => (
        <Badge status={statusBadge[v]} text={
          <Text style={{ color: 'var(--text-secondary)' }}>{statusLabel[v]}</Text>
        } />
      ),
    },
    {
      title: '风险等级',
      dataIndex: 'risk_level',
      render: (v: string | null) => v
        ? <Tag color={riskColor[v]} style={{ fontWeight: 600 }}>{riskLabel[v]}</Tag>
        : <Text style={{ color: 'var(--text-muted)' }}>—</Text>,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      render: (v: string) => <Text style={{ color: 'var(--text-secondary)', fontSize: 12 }}>{v?.slice(0, 16).replace('T', ' ')}</Text>,
    },
    {
      title: '操作',
      render: (_: unknown, r: Assessment) => (
        <Space>
          {(r.status === 'running' || r.status === 'pending') && (
            <Button type="link" size="small" icon={<EyeOutlined />}
              style={{ padding: 0, color: '#ffc53d' }}
              onClick={() => openDrawer(r.id)}>
              查看进度
            </Button>
          )}
          <Button type="link" size="small" style={{ padding: 0, color: '#4d96ff' }}
            disabled={r.status !== 'completed'}
            onClick={() => navigate(`/enterprise/reports/${r.id}`)}>
            查看报告
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <Text style={{ color: 'var(--text-primary)', fontSize: 18, fontWeight: 600 }}>评估记录</Text>
        <Space>
          <Input
            placeholder="搜索评估任务..."
            prefix={<SearchOutlined style={{ color: 'var(--text-muted)' }} />}
            style={{ width: 220 }}
            value={keyword}
            onChange={e => setKeyword(e.target.value)}
          />
          <Select value={statusFilter} style={{ width: 120 }} onChange={setStatusFilter} options={[
            { value: 'all', label: '全部状态' },
            { value: 'completed', label: '已完成' },
            { value: 'running', label: '进行中' },
            { value: 'failed', label: '失败' },
            { value: 'pending', label: '等待中' },
          ]} />
          <Button icon={<ReloadOutlined />} onClick={fetchList} style={{ color: 'var(--text-secondary)' }} />
        </Space>
      </div>

      <Table
        dataSource={filtered}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: setPage,
          showTotal: t => `共 ${t} 条`,
        }}
        style={{ background: 'transparent' }}
      />

      {/* SSE 实时日志 Drawer */}
      <Drawer
        title={
          <div>
            <div style={{ color: 'var(--text-primary)', fontWeight: 600 }}>
              实时执行日志
            </div>
            <div style={{ color: 'var(--text-muted)', fontSize: 12, fontWeight: 400 }}>
              {selectedAssessment?.name}
            </div>
          </div>
        }
        width={600}
        open={drawerOpen}
        onClose={closeDrawer}
        styles={{ body: { background: 'var(--bg-base)', padding: 16 }, header: { background: 'var(--bg-surface)', borderBottom: '1px solid var(--border-color)' } }}
        footer={
          streamDone && selectedId ? (
            <Button type="primary" onClick={() => {
              closeDrawer()
              navigate(`/enterprise/reports/${selectedId}`)
            }}>
              查看报告
            </Button>
          ) : null
        }
      >
        {logs.length === 0 && !streamDone && (
          <div style={{ textAlign: 'center', padding: '60px 0' }}>
            <Spin size="large" />
            <div style={{ color: 'var(--text-muted)', marginTop: 16 }}>等待任务日志...</div>
          </div>
        )}
        {logs.length === 0 && streamDone && (
          <Empty description="暂无日志" />
        )}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {logs.map((log, i) => (
            <div key={i} style={{
              padding: '10px 14px',
              background: 'var(--bg-card)',
              border: '1px solid var(--border-color)',
              borderRadius: 6,
              fontSize: 13,
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                  <Tag color="#4d96ff" style={{ fontSize: 11 }}>#{log.iteration}</Tag>
                  <Text style={{ color: 'var(--text-primary)', fontWeight: 500 }}>{log.tool_name}</Text>
                  {log.severity && (
                    <Tag color={riskColor[log.severity] || '#8890a4'} style={{ fontSize: 11 }}>
                      {riskLabel[log.severity] || log.severity}
                    </Tag>
                  )}
                </div>
                <Text style={{ color: 'var(--text-muted)', fontSize: 11 }}>
                  {log.duration_ms}ms
                </Text>
              </div>
              <Text style={{ color: 'var(--text-secondary)', fontSize: 12, wordBreak: 'break-all' }}>
                {log.output?.slice(0, 200)}{log.output?.length > 200 ? '...' : ''}
              </Text>
            </div>
          ))}
          <div ref={logsEndRef} />
        </div>
        {streamDone && (
          <div style={{
            marginTop: 12, padding: '8px 14px',
            background: 'rgba(82,196,26,0.08)',
            border: '1px solid rgba(82,196,26,0.3)',
            borderRadius: 6,
            color: '#52c41a',
            fontSize: 13,
          }}>
            评估执行完成
          </div>
        )}
      </Drawer>
    </div>
  )
}
