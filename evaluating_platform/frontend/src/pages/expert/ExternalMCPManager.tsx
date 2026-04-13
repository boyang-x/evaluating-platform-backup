import { useEffect, useState } from 'react'
import {
  Alert,
  Button,
  Card,
  Col,
  Empty,
  Form,
  Input,
  InputNumber,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Switch,
  Tag,
  Typography,
  message,
} from 'antd'
import {
  ApiOutlined,
  DeleteOutlined,
  EditOutlined,
  LinkOutlined,
  ReloadOutlined,
  SyncOutlined,
} from '@ant-design/icons'
import { expertService } from '../../services/expert'

const { Paragraph, Text, Title } = Typography
const { TextArea } = Input

interface ExternalMCPServer {
  id: string
  name: string
  namespace: string
  description: string
  base_url: string
  transport_type: string
  auth_type: string
  auth_header: string
  auth_prefix: string
  auth_key_configured: boolean
  upstream_base_url: string
  upstream_model: string
  upstream_timeout_seconds: number
  upstream_api_key_configured: boolean
  timeout_seconds: number
  enabled: boolean
  skill_prompt: string
  status: string
  last_sync_at?: string
  last_error?: string
  tool_count: number
  created_at: string
  updated_at: string
}

interface ExternalMCPTool {
  id: string
  remote_tool_name: string
  proxy_tool_name: string
  description: string
  input_schema: unknown
  enabled: boolean
}

function formatDate(value?: string) {
  if (!value) {
    return '未同步'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function statusTag(status: string) {
  const normalized = status || 'unknown'
  switch (normalized) {
    case 'connected':
      return <Tag color="green">已连接</Tag>
    case 'disabled':
      return <Tag>已停用</Tag>
    case 'pending':
      return <Tag color="gold">待同步</Tag>
    case 'error':
      return <Tag color="red">异常</Tag>
    default:
      return <Tag color="blue">{normalized}</Tag>
  }
}

export function ExternalMCPManager() {
  const [items, setItems] = useState<ExternalMCPServer[]>([])
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [showModal, setShowModal] = useState(false)
  const [editing, setEditing] = useState<ExternalMCPServer | null>(null)
  const [showToolsModal, setShowToolsModal] = useState(false)
  const [toolItems, setToolItems] = useState<ExternalMCPTool[]>([])
  const [toolsLoading, setToolsLoading] = useState(false)
  const [selectedServerName, setSelectedServerName] = useState('')
  const [form] = Form.useForm()

  const loadItems = () => {
    setLoading(true)
    expertService.listExternalMCPServers()
      .then((res) => setItems(res.items || []))
      .catch(() => message.error('加载 MCP 服务失败'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    loadItems()
  }, [])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    form.setFieldsValue({
      transport_type: 'sse',
      auth_type: 'none',
      auth_header: 'Authorization',
      auth_prefix: 'Bearer',
      upstream_base_url: '',
      upstream_api_key: '',
      upstream_model: '',
      upstream_timeout_seconds: 60,
      timeout_seconds: 30,
      enabled: true,
    })
    setShowModal(true)
  }

  const openEdit = (item: ExternalMCPServer) => {
    setEditing(item)
    form.setFieldsValue({
      name: item.name,
      namespace: item.namespace,
      description: item.description,
      base_url: item.base_url,
      transport_type: item.transport_type || 'sse',
      auth_type: item.auth_type || 'none',
      auth_key: '',
      auth_header: item.auth_header || 'Authorization',
      auth_prefix: item.auth_prefix || 'Bearer',
      upstream_base_url: item.upstream_base_url || '',
      upstream_api_key: '',
      upstream_model: item.upstream_model || '',
      upstream_timeout_seconds: item.upstream_timeout_seconds || 60,
      timeout_seconds: item.timeout_seconds || 30,
      enabled: item.enabled,
      skill_prompt: item.skill_prompt || '',
    })
    setShowModal(true)
  }

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      const payload = {
        name: values.name,
        namespace: values.namespace || '',
        description: values.description || '',
        base_url: values.base_url,
        transport_type: values.transport_type,
        auth_type: values.auth_type,
        auth_key: values.auth_key || '',
        auth_header: values.auth_header || '',
        auth_prefix: values.auth_prefix || '',
        upstream_base_url: values.upstream_base_url || '',
        upstream_api_key: values.upstream_api_key || '',
        upstream_model: values.upstream_model || '',
        upstream_timeout_seconds: values.upstream_timeout_seconds || 0,
        timeout_seconds: values.timeout_seconds,
        enabled: values.enabled,
        skill_prompt: values.skill_prompt || '',
      }

      const res = editing
        ? await expertService.updateExternalMCPServer(editing.id, payload)
        : await expertService.createExternalMCPServer(payload)

      message.success(editing ? 'MCP 服务已更新' : 'MCP 服务已创建')
      if (res.warning) {
        message.warning(`配置已保存，但同步失败：${res.warning}`)
      }
      setShowModal(false)
      form.resetFields()
      loadItems()
    } catch (error: unknown) {
      message.error((error as Error).message || '保存 MCP 服务失败')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await expertService.deleteExternalMCPServer(id)
      message.success('MCP 服务已删除')
      loadItems()
    } catch (error: unknown) {
      message.error((error as Error).message || '删除 MCP 服务失败')
    }
  }

  const handleTest = async (id: string) => {
    try {
      const res = await expertService.testExternalMCPServer(id)
      const result = res.result
      message.success(`连接成功，发现 ${result?.tool_count ?? 0} 个工具`)
      loadItems()
    } catch (error: unknown) {
      message.error((error as Error).message || '连接测试失败')
    }
  }

  const handleSync = async (id: string) => {
    try {
      const res = await expertService.syncExternalMCPServer(id)
      message.success(`工具已同步，共 ${res.tools?.length || 0} 个`) 
      loadItems()
    } catch (error: unknown) {
      message.error((error as Error).message || '同步工具失败')
    }
  }

  const handleViewTools = async (item: ExternalMCPServer) => {
    try {
      setSelectedServerName(item.name)
      setShowToolsModal(true)
      setToolsLoading(true)
      const res = await expertService.listExternalMCPTools(item.id)
      setToolItems(res.items || [])
    } catch (error: unknown) {
      message.error((error as Error).message || '加载外部工具失败')
      setShowToolsModal(false)
    } finally {
      setToolsLoading(false)
    }
  }

  const authType = Form.useWatch('auth_type', form)

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <div>
          <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>
            MCP 服务
          </Title>
          <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
            在这里登记外部 MCP Server。同步成功后，外部工具会自动代理进平台内部工具空间，供后续编排调用。
          </Text>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadItems}>刷新</Button>
          <Button type="primary" icon={<ApiOutlined />} onClick={openCreate}>新增 MCP 服务</Button>
        </Space>
      </div>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="当前实现方式"
        description="每个外部 MCP Server 都会由后端独立建立连接，并把远端工具注册成内部代理工具。编排 LLM 仍然只看到一套统一工具空间，不会直接面对多个原始 server。"
      />

      {items.length === 0 ? (
        <Card loading={loading}>
          <Empty description="当前还没有配置外部 MCP 服务。新增后即可测试连接并同步远端工具。" />
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {items.map((item) => (
            <Col key={item.id} xs={24} md={12} xl={8}>
              <Card
                loading={loading}
                hoverable
                title={<Text style={{ color: 'var(--text-primary)' }}>{item.name}</Text>}
                extra={statusTag(item.status)}
                actions={[
                  <Button key="test" type="link" icon={<LinkOutlined />} onClick={() => handleTest(item.id)}>
                    测试连接
                  </Button>,
                  <Button key="sync" type="link" icon={<SyncOutlined />} onClick={() => handleSync(item.id)}>
                    同步工具
                  </Button>,
                  <Button key="edit" type="link" icon={<EditOutlined />} onClick={() => openEdit(item)}>
                    编辑
                  </Button>,
                  <Popconfirm key="delete" title={`确认删除 “${item.name}” 吗？`} onConfirm={() => handleDelete(item.id)}>
                    <Button type="link" danger icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>,
                ]}
                style={{ height: '100%', background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
              >
                <Space size={[0, 8]} wrap style={{ marginBottom: 12 }}>
                  <Tag color="blue">{item.transport_type?.toUpperCase() || 'SSE'}</Tag>
                  <Tag color="purple">{item.namespace}</Tag>
                  <Tag>{item.tool_count} 个工具</Tag>
                  {item.enabled ? <Tag color="green">已启用</Tag> : <Tag>已停用</Tag>}
                </Space>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>服务地址</Text>
                  <Paragraph style={{ color: 'var(--text-secondary)', marginBottom: 0 }} ellipsis={{ rows: 2 }}>
                    {item.base_url}
                  </Paragraph>
                </div>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>服务说明</Text>
                  <Paragraph style={{ color: 'var(--text-secondary)', minHeight: 44, marginBottom: 0 }} ellipsis={{ rows: 2 }}>
                    {item.description || '未填写描述。建议说明这个服务擅长生成什么类型的测试资源。'}
                  </Paragraph>
                </div>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>上游模型配置</Text>
                  <div>
                    <Text style={{ color: 'var(--text-secondary)' }}>
                      {item.upstream_model || item.upstream_base_url || item.upstream_api_key_configured
                        ? `模型：${item.upstream_model || '未设置'}`
                        : '未设置'}
                    </Text>
                  </div>
                  {item.upstream_base_url ? (
                    <Paragraph style={{ color: 'var(--text-secondary)', marginBottom: 0 }} ellipsis={{ rows: 2 }}>
                      {item.upstream_base_url}
                    </Paragraph>
                  ) : null}
                </div>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>最近同步</Text>
                  <div>
                    <Text style={{ color: 'var(--text-secondary)' }}>{formatDate(item.last_sync_at)}</Text>
                  </div>
                </div>

                {item.last_error ? (
                  <Alert type="error" showIcon style={{ marginBottom: 12 }} message="最近一次错误" description={item.last_error} />
                ) : null}

                <Button block onClick={() => handleViewTools(item)}>
                  查看已同步工具
                </Button>
              </Card>
            </Col>
          ))}
        </Row>
      )}

      <Modal
        title={editing ? `编辑 MCP 服务：${editing.name}` : '新增 MCP 服务'}
        open={showModal}
        onCancel={() => {
          setShowModal(false)
          setEditing(null)
          form.resetFields()
        }}
        onOk={handleSave}
        okText={editing ? '保存' : '创建并同步'}
        confirmLoading={saving}
        width={720}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="服务名称" rules={[{ required: true, message: '请输入服务名称' }]}> 
            <Input placeholder="如：Papillon 样本生成服务" />
          </Form.Item>
          <Form.Item name="namespace" label="命名空间" extra="用于生成内部代理工具名，例如 ext_papillon_generate_samples。留空时会自动从名称推导。"> 
            <Input placeholder="如：papillon" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={2} placeholder="说明这个外部 MCP Server 的作用、擅长场景或限制" />
          </Form.Item>
          <Form.Item name="base_url" label="SSE 地址" rules={[{ required: true, message: '请输入 SSE 地址' }]}> 
            <Input placeholder="如：http://127.0.0.1:8091/sse" />
          </Form.Item>
              <Space style={{ width: '100%' }} size="middle" align="start">
                <Form.Item name="transport_type" label="传输方式" style={{ width: 160 }}>
                  <Select options={[{ value: 'sse', label: 'SSE' }]} />
                </Form.Item>
                <Form.Item name="timeout_seconds" label="超时（秒）" style={{ width: 160 }}>
                  <InputNumber min={5} max={300} style={{ width: '100%' }} />
                </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch checkedChildren="启用" unCheckedChildren="停用" />
            </Form.Item>
          </Space>

          <Form.Item name="auth_type" label="认证方式">
            <Select
              options={[
                { value: 'none', label: '无认证' },
                { value: 'bearer', label: 'Bearer Token' },
                { value: 'custom_header', label: '自定义请求头' },
              ]}
            />
          </Form.Item>

          {authType !== 'none' ? (
            <>
              <Space style={{ width: '100%' }} size="middle" align="start">
                {authType === 'custom_header' ? (
                  <Form.Item name="auth_header" label="Header 名称" style={{ width: 220 }} rules={[{ required: true, message: '请输入 Header 名称' }]}> 
                    <Input placeholder="如：X-API-Key" />
                  </Form.Item>
                ) : null}
                <Form.Item name="auth_prefix" label="前缀" style={{ width: 220 }}>
                  <Input placeholder={authType === 'bearer' ? 'Bearer' : '可留空'} />
                </Form.Item>
              </Space>
              <Form.Item
                name="auth_key"
                label={editing && editing.auth_key_configured ? '认证密钥（留空表示保持不变）' : '认证密钥'}
                rules={editing && editing.auth_key_configured ? [] : [{ required: true, message: '请输入认证密钥' }]}
              >
                <Input.Password placeholder="输入 API Key / Token" />
              </Form.Item>
            </>
          ) : null}

          <Card
            size="small"
            title="上游模型配置"
            style={{ marginBottom: 16, background: 'rgba(255,255,255,0.03)', border: '1px solid var(--border-color)' }}
          >
            <Text style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
              适用于支持 Header 覆盖配置的外部服务，例如当前的 CC-BOS。保存后，平台会通过请求头把这些参数传给外部 MCP Server。
            </Text>
            <Form.Item name="upstream_base_url" label="上游 Base URL" style={{ marginTop: 12 }}>
              <Input placeholder="如：https://api.deepseek.com/v1" />
            </Form.Item>
            <Space style={{ width: '100%' }} size="middle" align="start">
              <Form.Item name="upstream_model" label="上游模型" style={{ width: '100%' }}>
                <Input placeholder="如：deepseek-chat / gpt-4o-mini" />
              </Form.Item>
              <Form.Item name="upstream_timeout_seconds" label="上游超时（秒）" style={{ width: 180 }}>
                <InputNumber min={5} max={600} style={{ width: '100%' }} />
              </Form.Item>
            </Space>
            <Form.Item
              name="upstream_api_key"
              label={editing && editing.upstream_api_key_configured ? '上游 API Key（留空表示保持不变）' : '上游 API Key'}
            >
              <Input.Password placeholder="输入外部服务内部调用上游模型所需的 API Key" />
            </Form.Item>
          </Card>

          <Form.Item name="skill_prompt" label="Skill / 使用说明" extra="可选。这里可以写给编排 LLM 的使用建议，后续接入更多外部能力时会很有用。">
            <TextArea rows={4} placeholder="例如：当现有专家样本不足时，再调用这个服务生成补充样本；大结果优先返回句柄而不是正文。" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`已同步工具：${selectedServerName}`}
        open={showToolsModal}
        onCancel={() => {
          setShowToolsModal(false)
          setToolItems([])
          setSelectedServerName('')
        }}
        footer={null}
        width={900}
      >
        {toolItems.length === 0 && !toolsLoading ? (
          <Empty description="当前还没有同步到工具。可以先点击“同步工具”。" />
        ) : (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            {toolItems.map((tool) => (
              <Card key={tool.id} loading={toolsLoading} size="small" style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}>
                <Space direction="vertical" style={{ width: '100%' }} size={6}>
                  <Space wrap>
                    <Text style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{tool.proxy_tool_name}</Text>
                    <Tag color="blue">远端：{tool.remote_tool_name}</Tag>
                    {tool.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>}
                  </Space>
                  <Paragraph style={{ color: 'var(--text-secondary)', marginBottom: 0 }}>
                    {tool.description || '无描述'}
                  </Paragraph>
                  <div>
                    <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>输入 Schema</Text>
                    <pre style={{ marginTop: 8, padding: 12, background: 'rgba(255,255,255,0.04)', borderRadius: 8, color: 'var(--text-secondary)', overflowX: 'auto' }}>
                      {JSON.stringify(tool.input_schema, null, 2)}
                    </pre>
                  </div>
                </Space>
              </Card>
            ))}
          </Space>
        )}
      </Modal>
    </div>
  )
}
