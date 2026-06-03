import { useEffect, useState } from 'react'
import { Button, Select, Space, Table, Tag, Typography } from 'antd'
import { ReloadOutlined } from '@ant-design/icons'
import { adminService, type AdminMaclawJob } from '../../services/admin'

const { Text, Title } = Typography

export function JobGovernance() {
  const [items, setItems] = useState<AdminMaclawJob[]>([])
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState<string>()

  const load = async () => {
    setLoading(true)
    try {
      const data = await adminService.listMaclawJobs({ status, limit: 50 })
      setItems(data.items)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [status])

  return (
    <div>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <div>
          <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>评测与任务</Title>
          <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>按账号查看 maclaw job 安全摘要。</Text>
        </div>
        <Space>
          <Select
            allowClear
            placeholder="状态"
            style={{ width: 160 }}
            value={status}
            onChange={setStatus}
            options={[
              { value: 'pending', label: 'pending' },
              { value: 'running', label: 'running' },
              { value: 'failed', label: 'failed' },
              { value: 'succeeded', label: 'succeeded' },
              { value: 'canceled', label: 'canceled' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
        </Space>
      </Space>
      <Table
        rowKey={(row) => `${row.platform_user_id}:${row.job.id}`}
        loading={loading}
        dataSource={items}
        pagination={{ pageSize: 12 }}
        columns={[
          { title: '账号', dataIndex: 'platform_email' },
          { title: '角色', dataIndex: 'platform_role', render: (value) => <Tag>{value}</Tag> },
          { title: 'tenant', dataIndex: 'maclaw_tenant_id', ellipsis: true },
          { title: 'instance', dataIndex: 'maclaw_instance_id', ellipsis: true },
          { title: 'job id', render: (_, row) => row.job.id },
          { title: '类型', render: (_, row) => row.job.kind },
          { title: '状态', render: (_, row) => <Tag color={row.job.status === 'failed' ? 'red' : row.job.status === 'succeeded' ? 'green' : 'blue'}>{row.job.status}</Tag> },
        ]}
      />
    </div>
  )
}
