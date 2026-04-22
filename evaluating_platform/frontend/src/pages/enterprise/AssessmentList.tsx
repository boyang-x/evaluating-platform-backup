import { useEffect, useRef, useState } from 'react'
import {
  Badge,
  Button,
  Drawer,
  Empty,
  Input,
  Popconfirm,
  Select,
  Space,
  Spin,
  Table,
  Tag,
  Typography,
  message,
} from 'antd'
import {
  DeleteOutlined,
  EyeOutlined,
  ReloadOutlined,
  SearchOutlined,
  StopOutlined,
} from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import {
  assessmentService,
  type Assessment,
  type DeleteAllAssessmentsResult,
  type LogEvent,
} from '../../services/assessment'

const { Text } = Typography

const statusBadge: Record<string, 'success' | 'processing' | 'error' | 'default' | 'warning'> = {
  completed: 'success',
  running: 'processing',
  failed: 'error',
  pending: 'default',
  canceled: 'warning',
}

const statusLabel: Record<string, string> = {
  completed: '已完成',
  running: '进行中',
  failed: '失败',
  pending: '等待中',
  canceled: '已取消',
}

const riskColor: Record<string, string> = {
  critical: '#ff4d4f',
  high: '#ff7a45',
  medium: '#ffc53d',
  low: '#52c41a',
}

const riskLabel: Record<string, string> = {
  critical: '严重',
  high: '高危',
  medium: '中危',
  low: '低危',
}

export function AssessmentList() {
  const navigate = useNavigate()
  const [items, setItems] = useState<Assessment[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [actionKey, setActionKey] = useState('')
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
    assessmentService.list(pageSize, (page - 1) * pageSize)
      .then((res) => {
        setItems(res.items || [])
        setTotal(res.total || 0)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    fetchList()
  }, [page])

  useEffect(() => {
    const hasActive = items.some((item) => item.status === 'pending' || item.status === 'running')
    if (!hasActive) {
      return
    }
    const timer = setInterval(fetchList, 5000)
    return () => clearInterval(timer)
  }, [items])

  useEffect(() => {
    logsEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [logs])

  const openDrawer = (id: string) => {
    setSelectedId(id)
    setLogs([])
    setStreamDone(false)
    setDrawerOpen(true)

    sseCloseRef.current?.()
    const close = assessmentService.streamLogs(
      id,
      (event) => setLogs((prev) => [...prev, event]),
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

  const refreshAfterMutation = () => {
    if (items.length === 1 && page > 1) {
      setPage((current) => Math.max(1, current - 1))
      return
    }
    fetchList()
  }

  const selectedAssessment = items.find((item) => item.id === selectedId)

  const handleCancel = async (assessment: Assessment) => {
    const nextActionKey = `cancel:${assessment.id}`
    setActionKey(nextActionKey)
    try {
      await assessmentService.cancel(assessment.id)
      message.success('评估任务已取消')
      if (selectedId === assessment.id) {
        closeDrawer()
      }
      refreshAfterMutation()
    } catch (error: unknown) {
      message.error((error as Error).message || '取消评估失败')
    } finally {
      setActionKey((current) => (current === nextActionKey ? '' : current))
    }
  }

  const handleDelete = async (assessment: Assessment) => {
    const nextActionKey = `delete:${assessment.id}`
    setActionKey(nextActionKey)
    try {
      await assessmentService.delete(assessment.id)
      message.success('评估记录已删除')
      if (selectedId === assessment.id) {
        closeDrawer()
      }
      refreshAfterMutation()
    } catch (error: unknown) {
      message.error((error as Error).message || '删除评估记录失败')
    } finally {
      setActionKey((current) => (current === nextActionKey ? '' : current))
    }
  }

  const handleDeleteAll = async () => {
    const nextActionKey = 'delete-all'
    setActionKey(nextActionKey)
    try {
      const summary = await assessmentService.deleteAll() as DeleteAllAssessmentsResult

      if (selectedAssessment && selectedAssessment.status !== 'running' && selectedAssessment.status !== 'pending') {
        closeDrawer()
      }

      if (summary.deleted_count > 0 && summary.active_count > 0) {
        message.success(`已删除 ${summary.deleted_count} 条评估记录，另有 ${summary.active_count} 条进行中记录未删除`)
      } else if (summary.deleted_count > 0) {
        message.success(`已删除 ${summary.deleted_count} 条评估记录`)
      } else if (summary.active_count > 0) {
        message.warning(`当前有 ${summary.active_count} 条进行中评估，请先取消后再删除`)
      } else {
        message.info('当前没有可删除的评估记录')
      }

      if (page > 1) {
        setPage(1)
      } else {
        fetchList()
      }
    } catch (error: unknown) {
      message.error((error as Error).message || '批量删除评估记录失败')
    } finally {
      setActionKey((current) => (current === nextActionKey ? '' : current))
    }
  }

  const filtered = items.filter((item) => {
    if (statusFilter !== 'all' && item.status !== statusFilter) {
      return false
    }
    if (keyword && !item.name.toLowerCase().includes(keyword.toLowerCase())) {
      return false
    }
    return true
  })

  const columns = [
    {
      title: '评估任务',
      dataIndex: 'name',
      render: (value: string, record: Assessment) => (
        <div>
          <Text style={{ color: 'var(--text-primary)', display: 'block' }}>{value}</Text>
          <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>{record.id.slice(0, 8)}...</Text>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      render: (value: string) => (
        <Badge
          status={statusBadge[value] || 'default'}
          text={<Text style={{ color: 'var(--text-secondary)' }}>{statusLabel[value] || value}</Text>}
        />
      ),
    },
    {
      title: '风险等级',
      dataIndex: 'risk_level',
      render: (value: string | null) => (
        value
          ? <Tag color={riskColor[value]} style={{ fontWeight: 600 }}>{riskLabel[value] || value}</Tag>
          : <Text style={{ color: 'var(--text-muted)' }}>-</Text>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      render: (value: string) => (
        <Text style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
          {value?.slice(0, 16).replace('T', ' ')}
        </Text>
      ),
    },
    {
      title: '操作',
      render: (_: unknown, record: Assessment) => (
        <Space wrap>
          {(record.status === 'running' || record.status === 'pending') && (
            <Button
              type="link"
              size="small"
              icon={<EyeOutlined />}
              style={{ padding: 0, color: '#ffc53d' }}
              onClick={() => openDrawer(record.id)}
            >
              查看进度
            </Button>
          )}
          {(record.status === 'running' || record.status === 'pending') && (
            <Popconfirm
              title="确认取消当前评估？"
              okText="确认取消"
              cancelText="再想想"
              onConfirm={() => void handleCancel(record)}
            >
              <Button
                type="link"
                size="small"
                icon={<StopOutlined />}
                loading={actionKey === `cancel:${record.id}`}
                style={{ padding: 0, color: '#ff7a45' }}
              >
                取消评估
              </Button>
            </Popconfirm>
          )}
          <Button
            type="link"
            size="small"
            style={{ padding: 0, color: '#4d96ff' }}
            disabled={record.status !== 'completed'}
            onClick={() => navigate(`/enterprise/reports/${record.id}`)}
          >
            查看报告
          </Button>
          {record.status !== 'running' && record.status !== 'pending' && (
            <Popconfirm
              title="确认删除这条评估记录？"
              description="相关日志和报告也会一并移除。"
              okText="确认删除"
              cancelText="取消"
              onConfirm={() => void handleDelete(record)}
            >
              <Button
                type="link"
                size="small"
                icon={<DeleteOutlined />}
                loading={actionKey === `delete:${record.id}`}
                style={{ padding: 0, color: '#ff7875' }}
              >
                删除记录
              </Button>
            </Popconfirm>
          )}
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
            onChange={(event) => setKeyword(event.target.value)}
          />
          <Select
            value={statusFilter}
            style={{ width: 120 }}
            onChange={setStatusFilter}
            options={[
              { value: 'all', label: '全部状态' },
              { value: 'completed', label: '已完成' },
              { value: 'running', label: '进行中' },
              { value: 'failed', label: '失败' },
              { value: 'pending', label: '等待中' },
              { value: 'canceled', label: '已取消' },
            ]}
          />
          <Popconfirm
            title="确认一键删除全部评估记录？"
            description="会删除当前账号下全部已结束的评估记录；进行中或排队中的任务不会删除，请先取消后再操作。"
            okText="确认删除"
            cancelText="取消"
            onConfirm={() => void handleDeleteAll()}
          >
            <Button
              danger
              icon={<DeleteOutlined />}
              loading={actionKey === 'delete-all'}
              disabled={loading || total === 0}
            >
              一键全部删除
            </Button>
          </Popconfirm>
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
          showTotal: (value) => `共 ${value} 条`,
        }}
        style={{ background: 'transparent' }}
      />

      <Drawer
        title={(
          <div>
            <div style={{ color: 'var(--text-primary)', fontWeight: 600 }}>实时执行日志</div>
            <div style={{ color: 'var(--text-muted)', fontSize: 12, fontWeight: 400 }}>
              {selectedAssessment?.name}
            </div>
          </div>
        )}
        width={600}
        open={drawerOpen}
        onClose={closeDrawer}
        styles={{
          body: { background: 'var(--bg-base)', padding: 16 },
          header: { background: 'var(--bg-surface)', borderBottom: '1px solid var(--border-color)' },
        }}
        footer={streamDone && selectedId ? (
          <Button
            type="primary"
            onClick={() => {
              closeDrawer()
              navigate(`/enterprise/reports/${selectedId}`)
            }}
          >
            查看报告
          </Button>
        ) : null}
      >
        {logs.length === 0 && !streamDone && (
          <div style={{ textAlign: 'center', padding: '60px 0' }}>
            <Spin size="large" />
            <div style={{ color: 'var(--text-muted)', marginTop: 16 }}>等待任务日志...</div>
          </div>
        )}
        {logs.length === 0 && streamDone && <Empty description="暂无日志" />}

        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          {logs.map((log, index) => (
            <div
              key={index}
              style={{
                padding: '10px 14px',
                background: 'var(--bg-card)',
                border: '1px solid var(--border-color)',
                borderRadius: 6,
                fontSize: 13,
              }}
            >
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
                <Text style={{ color: 'var(--text-muted)', fontSize: 11 }}>{log.duration_ms}ms</Text>
              </div>
              <Text style={{ color: 'var(--text-secondary)', fontSize: 12, wordBreak: 'break-all' }}>
                {log.output?.slice(0, 200)}
                {log.output?.length > 200 ? '...' : ''}
              </Text>
            </div>
          ))}
          <div ref={logsEndRef} />
        </div>

        {streamDone && (
          <div
            style={{
              marginTop: 12,
              padding: '8px 14px',
              background: 'rgba(82,196,26,0.08)',
              border: '1px solid rgba(82,196,26,0.3)',
              borderRadius: 6,
              color: '#52c41a',
              fontSize: 13,
            }}
          >
            评估执行完成
          </div>
        )}
      </Drawer>
    </div>
  )
}
