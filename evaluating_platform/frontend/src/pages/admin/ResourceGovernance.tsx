import { useEffect, useState } from 'react'
import { Button, Space, Switch, Table, Tag, Typography, message } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { adminService, type AdminMaclawResource } from '../../services/admin'

const { Text, Title } = Typography

export function ResourceGovernance() {
  const [items, setItems] = useState<AdminMaclawResource[]>([])
  const [loading, setLoading] = useState(false)

  const load = async () => {
    setLoading(true)
    try {
      const data = await adminService.listMaclawResources({ include_inactive: true, limit: 300 })
      setItems(data.items)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const updateEnabled = async (row: AdminMaclawResource, enabled: boolean) => {
    try {
      await adminService.updateMaclawResource(row.source_resource_id, { enabled })
      message.success(enabled ? '资源已启用' : '资源已禁用')
      load()
    } catch (error) {
      message.error((error as Error).message || '更新失败')
    }
  }

  const archive = async (row: AdminMaclawResource) => {
    try {
      await adminService.updateMaclawResource(row.source_resource_id, { enabled: false, status: 'archived' })
      message.success('资源已归档')
      load()
    } catch (error) {
      message.error((error as Error).message || '归档失败')
    }
  }

  return (
    <div>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <div>
          <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>资源治理</Title>
          <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>统一查看专家发布资源目录，不展示资源正文。</Text>
        </div>
        <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
      </Space>
      <Table
        rowKey="source_resource_id"
        loading={loading}
        dataSource={items}
        pagination={{ pageSize: 12 }}
        columns={[
          { title: '资源', dataIndex: 'name', render: (value, row) => <Space orientation="vertical" size={0}><Text>{value}</Text><Text type="secondary">{row.summary || row.source_resource_handle}</Text></Space> },
          { title: '专家', dataIndex: 'source_expert_email', render: (value, row) => <Space orientation="vertical" size={0}><Text>{row.source_expert_name || '-'}</Text><Text type="secondary">{value}</Text></Space> },
          { title: '类型', dataIndex: 'kind', render: (value) => <Tag>{value}</Tag> },
          { title: '版本', dataIndex: 'source_version' },
          { title: '状态', dataIndex: 'status', render: (value) => <Tag color={value === 'published' ? 'green' : value === 'archived' ? 'default' : 'blue'}>{value}</Tag> },
          { title: '企业可见', dataIndex: 'enabled', render: (value, row) => <Switch checked={value} onChange={(checked) => updateEnabled(row, checked)} /> },
          { title: '标签', dataIndex: 'tags', render: (tags?: string[]) => <Space wrap>{(tags || []).map((tag) => <Tag key={tag}>{tag}</Tag>)}</Space> },
          { title: '操作', render: (_, row) => <Button size="small" danger disabled={row.status === 'archived'} onClick={() => archive(row)}>归档</Button> },
        ]}
      />
    </div>
  )
}
