import { useEffect, useMemo, useState } from 'react'
import {
  Badge,
  Button,
  Card,
  Col,
  Empty,
  Popconfirm,
  Row,
  Space,
  Statistic,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from 'antd'
import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  EditOutlined,
  GlobalOutlined,
  LockOutlined,
  RiseOutlined,
  StopOutlined,
} from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '../../context/AuthContext'
import { assetService, type Asset } from '../../services/asset'
import { billingService } from '../../services/billing'

const { Text, Title } = Typography

const typeLabel: Record<string, string> = {
  tool_config: '工具配置',
  workflow: '工作流',
  suite: '评估套件',
}

const typeColor: Record<string, string> = {
  tool_config: '#4d96ff',
  workflow: '#ff7a45',
  suite: '#52c41a',
}

const statusBadge: Record<string, React.ComponentProps<typeof Badge>['status']> = {
  published: 'success',
  testing: 'processing',
  draft: 'default',
  deprecated: 'error',
}

const statusLabel: Record<string, string> = {
  published: '已发布',
  testing: '审核中',
  draft: '草稿',
  deprecated: '已废弃',
}

const EXPERT_SHARE = 0.3

export function MyAssets() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [assets, setAssets] = useState<Asset[]>([])
  const [loading, setLoading] = useState(true)
  const [totalEarnings, setTotalEarnings] = useState(0)
  const [activeTab, setActiveTab] = useState('all')

  const load = () => {
    if (!user) return
    setLoading(true)
    Promise.all([assetService.listMine(100, 0), billingService.getEarnings(1, 0)])
      .then(([assetRes, earningRes]) => {
        setAssets(assetRes.items || [])
        setTotalEarnings(earningRes.total_earnings || 0)
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load()
  }, [user])

  const handleSubmitReview = async (id: string) => {
    try {
      await assetService.submitForReview(id)
      message.success('已提交审核')
      load()
    } catch (err: unknown) {
      message.error((err as Error).message || '提交失败')
    }
  }

  const handlePublish = async (id: string) => {
    try {
      await assetService.publish(id)
      message.success('资产已发布')
      load()
    } catch (err: unknown) {
      message.error((err as Error).message || '发布失败')
    }
  }

  const handleDeprecate = async (id: string) => {
    try {
      await assetService.deprecate(id)
      message.success('资产已废弃')
      load()
    } catch (err: unknown) {
      message.error((err as Error).message || '操作失败')
    }
  }

  const published = assets.filter((asset) => asset.status === 'published').length
  const totalCalls = assets.reduce((sum, asset) => sum + (asset.call_count || 0), 0)
  const estimatedEarnings = assets.reduce((sum, asset) => sum + (asset.call_count || 0) * (asset.price_unit || 0) * EXPERT_SHARE, 0)

  const filteredAssets = useMemo(() => {
    if (activeTab === 'all') {
      return assets
    }
    return assets.filter((asset) => asset.status === activeTab)
  }, [activeTab, assets])

  const columns = [
    {
      title: '资产名称',
      render: (_: unknown, asset: Asset) => (
        <div>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <Text style={{ color: 'var(--text-primary)', fontWeight: 500 }}>{asset.name}</Text>
            <Tag color={typeColor[asset.type]} style={{ fontSize: 11 }}>
              {typeLabel[asset.type]}
            </Tag>
            {asset.visibility === 'public' ? (
              <GlobalOutlined style={{ color: '#4d96ff', fontSize: 12 }} title="公开" />
            ) : (
              <LockOutlined style={{ color: 'var(--text-muted)', fontSize: 12 }} title="私有" />
            )}
          </div>
          <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>
            {asset.version}
            {asset.description ? ` · ${asset.description.slice(0, 40)}` : ''}
          </Text>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 120,
      render: (value: string) => (
        <Badge
          status={statusBadge[value] || 'default'}
          text={<Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>{statusLabel[value] || value}</Text>}
        />
      ),
    },
    {
      title: '调用 / 单价',
      width: 140,
      render: (_: unknown, asset: Asset) => (
        <div>
          <Text style={{ color: '#4d96ff' }}>{asset.call_count ?? 0} 次</Text>
          <Text style={{ color: 'var(--text-muted)', display: 'block', fontSize: 12 }}>
            ¥{(asset.price_unit ?? 0).toFixed(2)} / 次
          </Text>
        </div>
      ),
    },
    {
      title: '专家收益',
      width: 120,
      render: (_: unknown, asset: Asset) => {
        const earning = (asset.call_count ?? 0) * (asset.price_unit ?? 0) * EXPERT_SHARE
        return (
          <Text style={{ color: earning > 0 ? '#52c41a' : 'var(--text-muted)', fontWeight: earning > 0 ? 600 : 400 }}>
            ¥ {earning.toFixed(2)}
          </Text>
        )
      },
    },
    {
      title: '操作',
      width: 220,
      render: (_: unknown, asset: Asset) => (
        <Space size={4}>
          <Button
            type="link"
            size="small"
            style={{ padding: 0, color: '#4d96ff' }}
            icon={<EditOutlined />}
            onClick={() => navigate(asset.type === 'workflow' ? '/expert/workflow' : '/expert/engine')}
          >
            编辑
          </Button>
          {asset.status === 'draft' ? (
            <Button
              type="link"
              size="small"
              style={{ padding: 0, color: '#ffc53d' }}
              icon={<ClockCircleOutlined />}
              onClick={() => handleSubmitReview(asset.id)}
            >
              提交审核
            </Button>
          ) : null}
          {(asset.status === 'draft' || asset.status === 'testing') ? (
            <Button
              type="link"
              size="small"
              style={{ padding: 0, color: '#52c41a' }}
              icon={<CheckCircleOutlined />}
              onClick={() => handlePublish(asset.id)}
            >
              直接发布
            </Button>
          ) : null}
          {asset.status === 'published' ? (
            <Popconfirm
              title="废弃后企业将无法继续使用该资产，确认吗？"
              onConfirm={() => handleDeprecate(asset.id)}
              okText="废弃"
              cancelText="取消"
              okButtonProps={{ danger: true }}
            >
              <Button type="link" size="small" danger style={{ padding: 0 }} icon={<StopOutlined />}>
                废弃
              </Button>
            </Popconfirm>
          ) : null}
        </Space>
      ),
    },
  ]

  return (
    <div>
      <div style={{ marginBottom: 20 }}>
        <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>
          我的资产
        </Title>
        <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>资产生命周期管理与收益追踪</Text>
      </div>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }} size="small">
            <Statistic title={<Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>资产总数</Text>} value={assets.length} valueStyle={{ color: 'var(--text-primary)', fontSize: 22 }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }} size="small">
            <Statistic
              title={<Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>已发布</Text>}
              value={published}
              valueStyle={{ color: '#52c41a', fontSize: 22 }}
              suffix={<Text style={{ color: 'var(--text-muted)', fontSize: 13 }}>/ {assets.length}</Text>}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }} size="small">
            <Statistic
              title={<Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>累计被调用</Text>}
              value={totalCalls}
              valueStyle={{ color: '#4d96ff', fontSize: 22 }}
              suffix={<Text style={{ color: 'var(--text-muted)', fontSize: 13 }}>次</Text>}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }} size="small">
            <Statistic
              title={<Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>累计收益</Text>}
              value={totalEarnings}
              precision={2}
              prefix="¥"
              valueStyle={{ color: '#ff7a45', fontSize: 22 }}
              suffix={estimatedEarnings > totalEarnings ? <Text style={{ color: 'var(--text-muted)', fontSize: 11 }}>{`(预估 ¥${estimatedEarnings.toFixed(2)})`}</Text> : null}
            />
            <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginTop: 4 }}>
              <RiseOutlined style={{ color: '#ff7a45', fontSize: 11 }} />
              <Text style={{ color: 'var(--text-muted)', fontSize: 11 }}>调用费用 × 30% 分成</Text>
            </div>
          </Card>
        </Col>
      </Row>

      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        style={{ marginBottom: 16 }}
        items={[
          { key: 'all', label: `全部 (${assets.length})` },
          { key: 'published', label: `已发布 (${assets.filter((asset) => asset.status === 'published').length})` },
          { key: 'testing', label: `审核中 (${assets.filter((asset) => asset.status === 'testing').length})` },
          { key: 'draft', label: `草稿 (${assets.filter((asset) => asset.status === 'draft').length})` },
          { key: 'deprecated', label: `已废弃 (${assets.filter((asset) => asset.status === 'deprecated').length})` },
        ]}
      />

      <Table
        dataSource={filteredAssets}
        columns={columns}
        rowKey="id"
        loading={loading}
        style={{ background: 'transparent' }}
        locale={{
          emptyText: (
            <Empty
              description={
                <Text style={{ color: 'var(--text-muted)' }}>
                  {activeTab === 'all' ? '暂无资产，前往引擎管理或工作流编排创建。' : `暂无“${statusLabel[activeTab]}”状态的资产`}
                </Text>
              }
            />
          ),
        }}
      />
    </div>
  )
}
