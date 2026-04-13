import { useEffect, useState } from 'react'
import {
  Row, Col, Card, Input, Select, Tag, Typography, Button, Spin, Empty, Space
} from 'antd'
import { SearchOutlined, ShopOutlined, PlayCircleOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { assetService, type Asset } from '../../services/asset'

const { Title, Text } = Typography

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

export function MarketPlace() {
  const navigate = useNavigate()
  const [assets, setAssets] = useState<Asset[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [keyword, setKeyword] = useState('')
  const [typeFilter, setTypeFilter] = useState('')

  useEffect(() => {
    setLoading(true)
    assetService.listPublic(typeFilter || undefined, 50, 0).then(r => {
      setAssets(r.items || [])
      setTotal(r.total || 0)
    }).catch(() => {}).finally(() => setLoading(false))
  }, [typeFilter])

  const filtered = assets.filter(a =>
    !keyword || a.name.toLowerCase().includes(keyword.toLowerCase()) ||
    a.description.toLowerCase().includes(keyword.toLowerCase())
  )

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>资产市场</Title>
        <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
          发现安全专家提供的评估工具和工作流，共 {total} 个资产
        </Text>
      </div>

      <Space style={{ marginBottom: 20 }}>
        <Input
          placeholder="搜索资产..."
          prefix={<SearchOutlined style={{ color: 'var(--text-muted)' }} />}
          style={{ width: 260 }}
          value={keyword}
          onChange={e => setKeyword(e.target.value)}
        />
        <Select
          value={typeFilter || 'all'}
          style={{ width: 140 }}
          onChange={v => setTypeFilter(v === 'all' ? '' : v)}
          options={[
            { value: 'all', label: '全部类型' },
            { value: 'tool_config', label: '工具配置' },
            { value: 'workflow', label: '工作流' },
            { value: 'suite', label: '评估套件' },
          ]}
        />
      </Space>

      {loading ? (
        <div style={{ textAlign: 'center', padding: 60 }}><Spin size="large" /></div>
      ) : filtered.length === 0 ? (
        <Empty description="暂无可用资产" />
      ) : (
        <Row gutter={[16, 16]}>
          {filtered.map(asset => (
            <Col span={8} key={asset.id}>
              <Card
                style={{
                  background: 'var(--bg-card)',
                  border: '1px solid var(--border-color)',
                  borderRadius: 8,
                  height: '100%',
                }}
                hoverable
              >
                <div style={{ marginBottom: 12 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <Tag color={typeColor[asset.type]} style={{ fontSize: 11 }}>
                      {typeLabel[asset.type]}
                    </Tag>
                    <Text style={{ color: '#4d96ff', fontWeight: 600 }}>
                      ¥{asset.price_unit}/次
                    </Text>
                  </div>
                  <Text style={{ color: 'var(--text-primary)', fontWeight: 600, fontSize: 15, display: 'block', marginTop: 8 }}>
                    {asset.name}
                  </Text>
                  <Text style={{ color: 'var(--text-secondary)', fontSize: 12, display: 'block', marginTop: 4 }}>
                    {asset.description || '暂无描述'}
                  </Text>
                </div>

                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>
                    <ShopOutlined style={{ marginRight: 4 }} />
                    {asset.call_count} 次调用
                  </Text>
                  <Button
                    type="primary"
                    size="small"
                    icon={<PlayCircleOutlined />}
                    onClick={() => navigate('/enterprise/assessments/new', {
                      state: { template_id: asset.id, template_name: asset.name }
                    })}
                  >
                    发起评估
                  </Button>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      )}
    </div>
  )
}
