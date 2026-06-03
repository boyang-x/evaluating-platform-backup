import { useEffect, useState } from 'react'
import { Alert, Button, Card, Form, Input, Select, Space, Switch, Table, Tag, Typography, message } from 'antd'
import { CloudSyncOutlined, ReloadOutlined, SaveOutlined } from '@ant-design/icons'
import { adminService, type MaclawHubStatusItem } from '../../services/admin'

const { Text, Title } = Typography

const sourceOptions = [
  { value: 'skillhub', label: 'SkillHub' },
  { value: 'github', label: 'GitHub' },
  { value: 'skillmarket', label: 'SkillMarket' },
  { value: 'clawhub', label: 'ClawHub' },
]

export function MaclawHubManagement() {
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [syncing, setSyncing] = useState(false)
  const [statusItems, setStatusItems] = useState<MaclawHubStatusItem[]>([])
  const [syncSummary, setSyncSummary] = useState<string>('')

  const load = async () => {
    setLoading(true)
    try {
      const [config, status] = await Promise.all([
        adminService.getMaclawHubConfig(),
        adminService.getMaclawHubConfigStatus(),
      ])
      form.setFieldsValue({
        enabled: config.enabled,
        hub_url: config.hub_url || '',
        allowed_sources: config.allowed_sources?.length ? config.allowed_sources : ['skillhub'],
      })
      setStatusItems(status.items || [])
    } catch (error) {
      message.error((error as Error).message || '加载 Hub 配置失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    load()
  }, [])

  const save = async () => {
    const values = await form.validateFields()
    setLoading(true)
    try {
      const out = await adminService.updateMaclawHubConfig({
        enabled: Boolean(values.enabled),
        hub_url: values.hub_url,
        allowed_sources: values.allowed_sources?.length ? values.allowed_sources : ['skillhub'],
      })
      form.setFieldsValue(out.config)
      setSyncSummary(`已同步 ${out.sync.succeeded}/${out.sync.attempted} 个租户，失败 ${out.sync.failed} 个`)
      message.success('Hub 配置已保存')
      await load()
    } catch (error) {
      message.error((error as Error).message || '保存 Hub 配置失败')
    } finally {
      setLoading(false)
    }
  }

  const sync = async () => {
    setSyncing(true)
    try {
      const out = await adminService.syncMaclawHubConfig()
      setSyncSummary(`已同步 ${out.succeeded}/${out.attempted} 个租户，失败 ${out.failed} 个`)
      message.success('Hub 配置同步已触发')
      await load()
    } catch (error) {
      message.error((error as Error).message || '同步 Hub 配置失败')
    } finally {
      setSyncing(false)
    }
  }

  return (
    <div>
      <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 16 }}>
        <div>
          <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>Hub 管理</Title>
          <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
            统一配置 MaClaw 私有 Hub；专家资源仍走能力目录，Skill 可通过 SkillHub 供各租户发现。
          </Text>
        </div>
        <Button icon={<ReloadOutlined />} onClick={load}>刷新</Button>
      </Space>

      <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)', borderRadius: 8, marginBottom: 16 }}>
        <Space orientation="vertical" size={16} style={{ width: '100%' }}>
          <Alert
            type="info"
            showIcon
            message="浏览器只保存 Hub URL、启用状态和来源策略；Hub token、maclaw token、Skill 包正文不会进入管理员页面。"
          />
          {syncSummary && <Alert type="success" showIcon message={syncSummary} />}
          <Form form={form} layout="vertical" disabled={loading} initialValues={{ enabled: false, allowed_sources: ['skillhub'] }}>
            <Form.Item label="启用统一私有 Hub" name="enabled" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item label="Hub URL" name="hub_url" rules={[{ type: 'url', message: '请输入合法 URL' }]}>
              <Input placeholder="https://hub.example.com" />
            </Form.Item>
            <Form.Item label="允许 Skill 来源" name="allowed_sources">
              <Select mode="multiple" options={sourceOptions} />
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<SaveOutlined />} loading={loading} onClick={save}>保存并同步</Button>
            <Button icon={<CloudSyncOutlined />} loading={syncing} onClick={sync}>同步到已 provision 租户</Button>
          </Space>
        </Space>
      </Card>

      <Table
        rowKey="platform_user_id"
        loading={loading}
        dataSource={statusItems}
        pagination={{ pageSize: 10 }}
        columns={[
          { title: '账号', dataIndex: 'platform_email' },
          { title: '角色', dataIndex: 'platform_role', render: (value) => <Tag>{value}</Tag> },
          { title: 'Tenant', dataIndex: 'maclaw_tenant_id' },
          { title: 'Instance', dataIndex: 'maclaw_instance_id' },
          {
            title: 'Provisioning',
            dataIndex: 'provisioning_status',
            render: (value) => <Tag color={value === 'ready' || value === 'succeeded' ? 'green' : 'blue'}>{value || 'unknown'}</Tag>,
          },
        ]}
      />
    </div>
  )
}
