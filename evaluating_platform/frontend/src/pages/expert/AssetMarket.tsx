import { useEffect, useState } from 'react'
import {
  Row, Col, Card, Input, Select, Tag, Typography, Spin, Empty, Space, Modal
} from 'antd'
import { SearchOutlined, ShopOutlined } from '@ant-design/icons'
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

export function AssetMarket() {
  const [assets, setAssets] = useState<Asset[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [keyword, setKeyword] = useState('')
  const [typeFilter, setTypeFilter] = useState('')
  const [selected, setSelected] = useState<Asset | null>(null)

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
          浏览平台公开发布的评估资产，共 {total} 个
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
                  borderRadius: 8, height: '100%', cursor: 'pointer',
                }}
                hoverable
                onClick={() => setSelected(asset)}
              >
                <div style={{ marginBottom: 12 }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <Tag color={typeColor[asset.type]} style={{ fontSize: 11 }}>
                      {typeLabel[asset.type]}
                    </Tag>
                    <Text style={{ color: '#4d96ff', fontWeight: 600 }}>¥{asset.price_unit}/次</Text>
                  </div>
                  <Text style={{ color: 'var(--text-primary)', fontWeight: 600, fontSize: 15, display: 'block', marginTop: 8 }}>
                    {asset.name}
                  </Text>
                  <Text style={{ color: 'var(--text-secondary)', fontSize: 12, display: 'block', marginTop: 4 }}>
                    {asset.description || '暂无描述'}
                  </Text>
                </div>
                <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>
                  <ShopOutlined style={{ marginRight: 4 }} />
                  {asset.call_count} 次调用 · v{asset.version}
                </Text>
              </Card>
            </Col>
          ))}
        </Row>
      )}

      {/* 资产详情弹窗 */}
      <Modal
        title={selected?.name}
        open={!!selected}
        onCancel={() => setSelected(null)}
        footer={null}
        width={560}
      >
        {selected && (
          <div>
            <div style={{ marginBottom: 12 }}>
              <Tag color={typeColor[selected.type]}>{typeLabel[selected.type]}</Tag>
              <Tag color="blue" style={{ marginLeft: 8 }}>v{selected.version}</Tag>
            </div>
            <Text style={{ color: 'var(--text-secondary)', display: 'block', marginBottom: 16 }}>
              {selected.description || '暂无描述'}
            </Text>
            {selected.config?.nodes?.length > 0 && (
              <div>
                <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>包含工具节点：</Text>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 6 }}>
                  {selected.config.nodes.map(n => (
                    <Tag key={n.id}>{n.label || n.tool_name}</Tag>
                  ))}
                </div>
              </div>
            )}
            <div style={{ marginTop: 16 }}>
              <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>
                调用次数：{selected.call_count} · 单价：¥{selected.price_unit}/次
              </Text>
            </div>
          </div>
        )}
      </Modal>
    </div>
  )
}
