import { useEffect, useState } from 'react'
import {
  Table, Tag, Typography, Button, Modal, Input, Space, message
} from 'antd'
import { CheckOutlined, CloseOutlined } from '@ant-design/icons'
import { adminService } from '../../services/admin'
import type { Asset } from '../../services/asset'

const { Title, Text } = Typography
const { TextArea } = Input

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

export function AssetReview() {
  const [assets, setAssets] = useState<Asset[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [rejectModal, setRejectModal] = useState(false)
  const [rejectId, setRejectId] = useState<string | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [page, setPage] = useState(1)

  const pageSize = 20

  const loadAssets = () => {
    setLoading(true)
    adminService.listPendingAssets(pageSize, (page - 1) * pageSize).then(r => {
      setAssets(r.items || [])
      setTotal(r.total || 0)
    }).catch(() => {}).finally(() => setLoading(false))
  }

  useEffect(() => { loadAssets() }, [page])

  const handleApprove = async (id: string) => {
    try {
      await adminService.approveAsset(id)
      message.success('已审核通过，资产已发布')
      setAssets(prev => prev.filter(a => a.id !== id))
      setTotal(prev => prev - 1)
    } catch (err: unknown) {
      message.error((err as Error).message || '操作失败')
    }
  }

  const handleReject = async () => {
    if (!rejectId) return
    try {
      await adminService.rejectAsset(rejectId, rejectReason)
      message.success('已驳回，资产退回草稿状态')
      setAssets(prev => prev.filter(a => a.id !== rejectId))
      setTotal(prev => prev - 1)
      setRejectModal(false)
      setRejectId(null)
      setRejectReason('')
    } catch (err: unknown) {
      message.error((err as Error).message || '操作失败')
    }
  }

  const columns = [
    {
      title: '资产名称',
      render: (_: unknown, r: Asset) => (
        <div>
          <Text style={{ color: 'var(--text-primary)', fontWeight: 500 }}>{r.name}</Text>
          <Text style={{ color: 'var(--text-muted)', display: 'block', fontSize: 12 }}>{r.description?.slice(0, 60)}</Text>
        </div>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      render: (v: string) => <Tag color={typeColor[v]}>{typeLabel[v] || v}</Tag>,
    },
    {
      title: '版本',
      dataIndex: 'version',
      render: (v: string) => <Text style={{ color: 'var(--text-secondary)' }}>v{v}</Text>,
    },
    {
      title: '工具节点',
      render: (_: unknown, r: Asset) => (
        <Text style={{ color: 'var(--text-secondary)' }}>
          {r.config?.nodes?.length || 0} 个节点
        </Text>
      ),
    },
    {
      title: '提交时间',
      dataIndex: 'updated_at',
      render: (v: string) => <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>{v?.slice(0, 16).replace('T', ' ')}</Text>,
    },
    {
      title: '操作',
      render: (_: unknown, r: Asset) => (
        <Space>
          <Button
            type="primary"
            size="small"
            icon={<CheckOutlined />}
            style={{ background: '#52c41a', borderColor: '#52c41a' }}
            onClick={() => handleApprove(r.id)}
          >
            通过
          </Button>
          <Button
            danger
            size="small"
            icon={<CloseOutlined />}
            onClick={() => { setRejectId(r.id); setRejectModal(true) }}
          >
            驳回
          </Button>
        </Space>
      ),
    },
  ]

  const expandedRowRender = (record: Asset) => (
    <div style={{ padding: '8px 16px' }}>
      <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>工具节点列表：</Text>
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6, marginTop: 6 }}>
        {(record.config?.nodes || []).map(n => (
          <Tag key={n.id}>{n.label || n.tool_name}</Tag>
        ))}
        {(!record.config?.nodes || record.config.nodes.length === 0) && (
          <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>无工具节点</Text>
        )}
      </div>
    </div>
  )

  return (
    <div>
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>
          资产审核
          {total > 0 && <Tag color="orange" style={{ marginLeft: 12, fontSize: 13 }}>{total} 待审核</Tag>}
        </Title>
        <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
          审核专家提交的资产，通过后自动发布至资产市场
        </Text>
      </div>

      <Table
        dataSource={assets}
        columns={columns}
        rowKey="id"
        loading={loading}
        expandable={{ expandedRowRender }}
        pagination={{
          current: page,
          pageSize,
          total,
          onChange: setPage,
          showTotal: t => `共 ${t} 条待审`,
        }}
        locale={{ emptyText: '暂无待审核资产' }}
        style={{ background: 'transparent' }}
      />

      <Modal
        title="驳回资产"
        open={rejectModal}
        onCancel={() => { setRejectModal(false); setRejectReason('') }}
        onOk={handleReject}
        okText="确认驳回"
        okButtonProps={{ danger: true }}
      >
        <div style={{ padding: '8px 0' }}>
          <Text style={{ color: 'var(--text-secondary)' }}>请填写驳回原因（将通知专家）：</Text>
          <TextArea
            value={rejectReason}
            onChange={e => setRejectReason(e.target.value)}
            rows={4}
            placeholder="如：工具描述不清晰，请补充详细说明..."
            style={{ marginTop: 8 }}
          />
        </div>
      </Modal>
    </div>
  )
}
