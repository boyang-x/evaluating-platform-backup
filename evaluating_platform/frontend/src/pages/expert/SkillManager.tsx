import { useEffect, useState } from 'react'
import {
  Alert,
  Button,
  Card,
  Col,
  Drawer,
  Empty,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Row,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
  Typography,
  Upload,
  message,
} from 'antd'
import {
  CloudUploadOutlined,
  DeleteOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  ReloadOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import { expertService } from '../../services/expert'

const { Paragraph, Text, Title } = Typography

interface SkillVersionSummary {
  id: string
  version: string
  manifest_version: string
  display_name: string
  summary: string
  input_source_mode: string
  execution_runtime: string
  assessment_types: string[]
  status: string
  last_self_test_run_id?: string
  created_at: string
  updated_at: string
}

interface SkillVersionDetail extends SkillVersionSummary {
  permissions?: unknown
  embedded_dataset_summary?: unknown
  validation_report?: unknown
}

interface SkillRunSummary {
  id: string
  skill_version_id: string
  skill_version?: string
  run_type: string
  trigger_source: string
  status: string
  exit_code?: number
  error_message: string
  validation_report?: unknown
  payload_dataset_summary?: unknown
  log_excerpt?: string
  started_at?: string
  completed_at?: string
  created_at: string
  updated_at: string
}

interface SkillSummary {
  id: string
  name: string
  slug: string
  description: string
  skill_type: string
  category: string
  capability_profile: string
  input_source_mode: string
  status: string
  latest_version_id?: string
  published_version_id?: string
  version_count: number
  run_count: number
  latest_version?: SkillVersionSummary
  published_version?: SkillVersionSummary
  created_at: string
  updated_at: string
}

interface SkillDetail extends SkillSummary {
  assessment_types: string[]
  permissions?: unknown
  embedded_dataset_summary?: unknown
  versions: SkillVersionDetail[]
  runs: SkillRunSummary[]
}

interface SkillConfigVersion {
  id: string
  version: string
  display_name: string
  status: string
}

interface SkillConfigOption {
  label: string
  value: string
}

interface SkillConfigField {
  key: string
  label: string
  type: string
  required: boolean
  secret: boolean
  description: string
  placeholder?: string
  default?: unknown
  env_name: string
  options?: SkillConfigOption[]
  configured: boolean
  value?: unknown
}

interface SkillConfigView {
  skill_id: string
  selected_version?: SkillConfigVersion
  config_complete: boolean
  missing_required_keys: string[]
  fields: SkillConfigField[]
}

interface PublishedConfigState {
  version_id: string
  config_complete: boolean
  missing_required_keys: string[]
}

function formatDate(value?: string) {
  if (!value) {
    return '未记录'
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
    second: '2-digit',
  })
}

function prettyJSON(value: unknown) {
  if (typeof value === 'string') {
    return value
  }
  return JSON.stringify(value ?? {}, null, 2)
}

function hasStructuredContent(value: unknown) {
  if (value === null || value === undefined) {
    return false
  }
  if (typeof value === 'string') {
    const trimmed = value.trim()
    return trimmed !== '' && trimmed !== '{}' && trimmed !== '[]'
  }
  if (Array.isArray(value)) {
    return value.length > 0
  }
  if (typeof value === 'object') {
    return Object.keys(value as Record<string, unknown>).length > 0
  }
  return true
}

function renderSkillStatus(status: string) {
  switch (status) {
    case 'published':
      return <Tag color="green">已启用</Tag>
    case 'disabled':
      return <Tag color="orange">已停用</Tag>
    case 'deprecated':
      return <Tag>历史废弃</Tag>
    case 'draft':
      return <Tag color="gold">草稿</Tag>
    default:
      return <Tag color="blue">{status || 'unknown'}</Tag>
  }
}

function renderVersionStatus(status: string) {
  switch (status) {
    case 'published':
      return <Tag color="green">已发布</Tag>
    case 'self_test_passed':
      return <Tag color="cyan">自测通过</Tag>
    case 'self_test_failed':
      return <Tag color="red">自测失败</Tag>
    case 'deprecated':
      return <Tag>历史废弃</Tag>
    case 'draft':
      return <Tag color="gold">草稿</Tag>
    default:
      return <Tag color="blue">{status || 'unknown'}</Tag>
  }
}

function renderRunStatus(status: string) {
  switch (status) {
    case 'completed':
      return <Tag color="green">完成</Tag>
    case 'failed':
      return <Tag color="red">失败</Tag>
    case 'timeout':
      return <Tag color="volcano">超时</Tag>
    case 'running':
      return <Tag color="processing">运行中</Tag>
    case 'pending':
      return <Tag color="gold">排队中</Tag>
    default:
      return <Tag color="blue">{status || 'unknown'}</Tag>
  }
}

function canPublishVersion(status: string) {
  return status === 'self_test_passed' || status === 'published'
}

function InfoBlock({ title, value, emptyText = '暂无' }: { title: string; value: unknown; emptyText?: string }) {
  return (
    <div style={{ marginBottom: 16 }}>
      <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>{title}</Text>
      {hasStructuredContent(value) ? (
        <pre
          style={{
            marginTop: 8,
            marginBottom: 0,
            padding: 12,
            background: 'rgba(255,255,255,0.04)',
            borderRadius: 8,
            color: 'var(--text-secondary)',
            overflowX: 'auto',
            whiteSpace: 'pre-wrap',
            wordBreak: 'break-word',
          }}
        >
          {prettyJSON(value)}
        </pre>
      ) : (
        <Paragraph style={{ color: 'var(--text-secondary)', marginTop: 8, marginBottom: 0 }}>
          {emptyText}
        </Paragraph>
      )}
    </div>
  )
}

function isSecretField(field: SkillConfigField) {
  return field.secret || field.type === 'password'
}

function getConfigInitialValues(config: SkillConfigView | null) {
  const values: Record<string, unknown> = {}
  if (!config) {
    return values
  }
  config.fields.forEach((field) => {
    if (isSecretField(field)) {
      return
    }
    if (field.value !== undefined) {
      values[field.key] = field.value
    }
  })
  return values
}

export function SkillManager() {
  const [items, setItems] = useState<SkillSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [showImportDrawer, setShowImportDrawer] = useState(false)
  const [importing, setImporting] = useState(false)
  const [selectedSkillID, setSelectedSkillID] = useState<string | null>(null)
  const [selectedSkill, setSelectedSkill] = useState<SkillDetail | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [runRefreshing, setRunRefreshing] = useState(false)
  const [actionKey, setActionKey] = useState('')
  const [configLoading, setConfigLoading] = useState(false)
  const [configSaving, setConfigSaving] = useState(false)
  const [selectedConfig, setSelectedConfig] = useState<SkillConfigView | null>(null)
  const [publishedConfigState, setPublishedConfigState] = useState<PublishedConfigState | null>(null)
  const [pendingConfigClears, setPendingConfigClears] = useState<Record<string, boolean>>({})
  const [form] = Form.useForm()
  const [configForm] = Form.useForm()

  const syncConfigForm = (config: SkillConfigView | null) => {
    configForm.resetFields()
    configForm.setFieldsValue(getConfigInitialValues(config))
    setPendingConfigClears({})
  }

  const loadItems = async () => {
    setLoading(true)
    try {
      const res = await expertService.listSkills()
      setItems((res.items || []) as SkillSummary[])
    } catch (error: unknown) {
      message.error((error as Error).message || '加载 Skill 列表失败')
    } finally {
      setLoading(false)
    }
  }

  const loadSkillDetail = async (skillID: string, silent = false) => {
    if (!silent) {
      setDetailLoading(true)
    }
    try {
      const res = await expertService.getSkill(skillID)
      const detail = (res.item || null) as SkillDetail | null
      setSelectedSkill(detail)
      return detail
    } catch (error: unknown) {
      message.error((error as Error).message || '加载 Skill 详情失败')
      return null
    } finally {
      if (!silent) {
        setDetailLoading(false)
      }
    }
  }

  const loadSkillConfig = async (skillID: string, versionID?: string, silent = false) => {
    if (!silent) {
      setConfigLoading(true)
    }
    try {
      const res = await expertService.getSkillConfig(skillID, versionID)
      const config = (res.item || null) as SkillConfigView | null
      setSelectedConfig(config)
      syncConfigForm(config)
      return config
    } catch (error: unknown) {
      message.error((error as Error).message || '加载 Skill 配置失败')
      setSelectedConfig(null)
      syncConfigForm(null)
      return null
    } finally {
      if (!silent) {
        setConfigLoading(false)
      }
    }
  }

  const loadPublishedConfigState = async (skillID: string, publishedVersionID?: string) => {
    if (!publishedVersionID) {
      setPublishedConfigState(null)
      return
    }
    try {
      const res = await expertService.getSkillConfig(skillID, publishedVersionID)
      const config = (res.item || null) as SkillConfigView | null
      if (!config?.selected_version) {
        setPublishedConfigState(null)
        return
      }
      setPublishedConfigState({
        version_id: config.selected_version.id,
        config_complete: config.config_complete,
        missing_required_keys: config.missing_required_keys || [],
      })
    } catch {
      setPublishedConfigState(null)
    }
  }

  const loadSkillRuns = async (skillID: string) => {
    setRunRefreshing(true)
    try {
      const res = await expertService.listSkillRuns(skillID)
      setSelectedSkill((current) => (current ? { ...current, runs: (res.items || []) as SkillRunSummary[] } : current))
    } catch (error: unknown) {
      message.error((error as Error).message || '加载运行记录失败')
    } finally {
      setRunRefreshing(false)
    }
  }

  useEffect(() => {
    void loadItems()
  }, [])

  const openDetail = async (skillID: string) => {
    setSelectedSkillID(skillID)
    setSelectedSkill(null)
    setSelectedConfig(null)
    setPublishedConfigState(null)
    setDetailOpen(true)
    const detail = await loadSkillDetail(skillID)
    await Promise.all([
      loadSkillConfig(skillID),
      loadPublishedConfigState(skillID, detail?.published_version?.id),
    ])
  }

  const refreshAfterMutation = async (skillID?: string | null) => {
    await loadItems()
    if (skillID) {
      const detail = await loadSkillDetail(skillID, true)
      await Promise.all([
        loadSkillConfig(skillID, selectedConfig?.selected_version?.id, true),
        loadPublishedConfigState(skillID, detail?.published_version?.id),
      ])
    }
  }

  const handleImport = async () => {
    try {
      const values = await form.validateFields()
      const formData = new FormData()
      formData.append('file', values.file.file.originFileObj || values.file.file)

      setImporting(true)
      const res = await expertService.importSkill(formData)
      const importedSkill = res.item as SkillSummary | undefined
      message.success('Skill ZIP 导入成功')
      setShowImportDrawer(false)
      form.resetFields()
      await loadItems()
      if (importedSkill?.id) {
        await openDetail(importedSkill.id)
      }
    } catch (error: unknown) {
      message.error((error as Error).message || '导入 Skill 失败')
    } finally {
      setImporting(false)
    }
  }

  const handleSelfTest = async (version: SkillVersionDetail) => {
    if (!selectedSkillID) {
      return
    }
    try {
      setActionKey(`self-test:${version.id}`)
      await expertService.selfTestSkill(selectedSkillID, version.id)
      message.success(`版本 ${version.version} 已执行自测`)
      await Promise.all([refreshAfterMutation(selectedSkillID), loadSkillRuns(selectedSkillID)])
    } catch (error: unknown) {
      message.error((error as Error).message || '触发自测失败')
    } finally {
      setActionKey('')
    }
  }

  const handlePublish = async (version: SkillVersionDetail) => {
    if (!selectedSkillID) {
      return
    }
    try {
      setActionKey(`publish:${version.id}`)
      await expertService.publishSkill(selectedSkillID, version.id)
      message.success(`版本 ${version.version} 已发布`)
      await refreshAfterMutation(selectedSkillID)
    } catch (error: unknown) {
      message.error((error as Error).message || '发布 Skill 失败')
    } finally {
      setActionKey('')
    }
  }

  const handleDisable = async () => {
    if (!selectedSkillID) {
      return
    }
    try {
      setActionKey('disable')
      await expertService.disableSkill(selectedSkillID)
      message.success('Skill 已停用')
      await refreshAfterMutation(selectedSkillID)
    } catch (error: unknown) {
      message.error((error as Error).message || '停用 Skill 失败')
    } finally {
      setActionKey('')
    }
  }

  const handleEnable = async () => {
    if (!selectedSkillID) {
      return
    }
    try {
      setActionKey('enable')
      await expertService.enableSkill(selectedSkillID)
      message.success('Skill 已重新启用')
      await refreshAfterMutation(selectedSkillID)
    } catch (error: unknown) {
      message.error((error as Error).message || '启用 Skill 失败')
    } finally {
      setActionKey('')
    }
  }

  const handleDelete = async () => {
    if (!selectedSkillID) {
      return
    }
    try {
      setActionKey('delete')
      await expertService.deleteSkill(selectedSkillID)
      message.success('Skill 已删除')
      setDetailOpen(false)
      setSelectedSkillID(null)
      setSelectedSkill(null)
      setSelectedConfig(null)
      setPublishedConfigState(null)
      syncConfigForm(null)
      await loadItems()
    } catch (error: unknown) {
      message.error((error as Error).message || '删除 Skill 失败')
    } finally {
      setActionKey('')
    }
  }

  const handleVersionConfigChange = async (versionID: string) => {
    if (!selectedSkillID) {
      return
    }
    await loadSkillConfig(selectedSkillID, versionID)
  }

  const handleConfigValuesChange = (changedValues: Record<string, unknown>) => {
    const changedKeys = Object.keys(changedValues || {})
    if (changedKeys.length === 0) {
      return
    }
    setPendingConfigClears((current) => {
      const next = { ...current }
      changedKeys.forEach((key) => {
        delete next[key]
      })
      return next
    })
  }

  const markConfigClearPending = (fieldKey: string) => {
    setPendingConfigClears((current) => ({ ...current, [fieldKey]: true }))
  }

  const cancelConfigClearPending = (fieldKey: string) => {
    setPendingConfigClears((current) => {
      const next = { ...current }
      delete next[fieldKey]
      return next
    })
  }

  const handleSaveConfig = async () => {
    if (!selectedSkillID || !selectedConfig) {
      return
    }

    const currentValues = configForm.getFieldsValue(true) as Record<string, unknown>
    const updates: Array<{ key: string; value?: unknown; clear?: boolean }> = []
    selectedConfig.fields.forEach((field) => {
      if (pendingConfigClears[field.key]) {
        updates.push({ key: field.key, clear: true })
        return
      }
      if (!configForm.isFieldTouched(field.key)) {
        return
      }

      const currentValue = currentValues[field.key]
      if (isSecretField(field)) {
        if (typeof currentValue !== 'string' || currentValue.trim() === '') {
          return
        }
        updates.push({ key: field.key, value: currentValue })
        return
      }
      if ((field.type === 'text' || field.type === 'textarea' || field.type === 'select') && (typeof currentValue !== 'string' || currentValue.trim() === '')) {
        return
      }
      if (field.type === 'number' && (currentValue === undefined || currentValue === null || currentValue === '')) {
        return
      }
      if (field.type === 'boolean' && typeof currentValue !== 'boolean') {
        return
      }
      updates.push({ key: field.key, value: currentValue })
    })

    if (updates.length === 0) {
      message.info('当前没有需要保存的配置变更')
      return
    }

    try {
      setConfigSaving(true)
      const res = await expertService.updateSkillConfig(selectedSkillID, {
        version_id: selectedConfig.selected_version?.id,
        values: updates,
      })
      const config = (res.item || null) as SkillConfigView | null
      setSelectedConfig(config)
      syncConfigForm(config)
      message.success('Skill 配置已保存')
      const detail = await loadSkillDetail(selectedSkillID, true)
      await Promise.all([
        loadItems(),
        loadPublishedConfigState(selectedSkillID, detail?.published_version?.id),
      ])
    } catch (error: unknown) {
      message.error((error as Error).message || '保存 Skill 配置失败')
    } finally {
      setConfigSaving(false)
    }
  }

  const renderConfigInput = (field: SkillConfigField) => {
    switch (field.type) {
      case 'textarea':
        if (isSecretField(field)) {
          return <Input.Password placeholder={field.placeholder || '输入后点击保存'} />
        }
        return <Input.TextArea rows={4} placeholder={field.placeholder || '请输入'} />
      case 'password':
        return <Input.Password placeholder={field.configured ? '已配置，如需更新请重新输入' : '输入后点击保存'} />
      case 'number':
        return <InputNumber style={{ width: '100%' }} placeholder={field.placeholder || '请输入数字'} />
      case 'boolean':
        return <Switch />
      case 'select':
        return (
          <Select
            placeholder={field.placeholder || '请选择'}
            allowClear={!field.required}
            options={(field.options || []).map((option) => ({
              label: option.label,
              value: option.value,
            }))}
          />
        )
      case 'text':
      default:
        if (isSecretField(field)) {
          return <Input.Password placeholder={field.configured ? '已配置，如需更新请重新输入' : '输入后点击保存'} />
        }
        return <Input placeholder={field.placeholder || '请输入'} />
    }
  }

  const renderDrawerActions = () => {
    if (!selectedSkill) {
      return null
    }

    const canEnable = selectedSkill.status === 'disabled' || selectedSkill.status === 'deprecated'
    const canDisable = selectedSkill.status === 'published'

    return (
      <Space>
        {canEnable ? (
          <Button
            icon={<PlayCircleOutlined />}
            loading={actionKey === 'enable'}
            onClick={() => void handleEnable()}
          >
            重新启用
          </Button>
        ) : null}
        {canDisable ? (
          <Popconfirm
            title={`确认停用「${selectedSkill.name}」吗？`}
            description="停用后它不会再被编排层推荐或执行，但已发布版本信息会保留，之后可以重新启用。"
            onConfirm={() => void handleDisable()}
          >
            <Button icon={<PauseCircleOutlined />} loading={actionKey === 'disable'}>
              停用 Skill
            </Button>
          </Popconfirm>
        ) : null}
        <Popconfirm
          title={`确认删除「${selectedSkill.name}」吗？`}
          description="删除会移除整个 Skill、版本和运行记录，这个操作不可恢复。"
          onConfirm={() => void handleDelete()}
          okText="删除"
          cancelText="取消"
          okButtonProps={{ danger: true }}
        >
          <Button danger icon={<DeleteOutlined />} loading={actionKey === 'delete'}>
            删除 Skill
          </Button>
        </Popconfirm>
      </Space>
    )
  }

  const configVersionOptions = (selectedSkill?.versions || []).map((version) => ({
    label: `${version.version} · ${version.display_name}`,
    value: version.id,
  }))

  const selectedConfigVersionID = selectedConfig?.selected_version?.id
  const showPublishedConfigWarning = Boolean(
    publishedConfigState &&
    !publishedConfigState.config_complete &&
    publishedConfigState.missing_required_keys.length > 0,
  )

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <div>
          <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>
            Skill 管理
          </Title>
          <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
            当前支持 `generator_skill` 与 `interactive_web_skill` ZIP 导入。专家端默认只展示脱敏后的元数据、校验报告、运行摘要和自测日志摘录。
          </Text>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void loadItems()}>
            刷新
          </Button>
          <Button type="primary" icon={<CloudUploadOutlined />} onClick={() => setShowImportDrawer(true)}>
            导入 Skill ZIP
          </Button>
        </Space>
      </div>

      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
        message="生命周期说明"
        description="Skill 上传后默认为草稿；只有自测通过的版本才能发布。停用是可恢复的下线操作，删除会永久移除整个 Skill。"
      />

      {items.length === 0 ? (
        <Card loading={loading}>
          <Empty description="当前还没有导入 Skill。导入 ZIP 后，这里会展示 Skill 摘要与版本状态。" />
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {items.map((item) => (
            <Col key={item.id} xs={24} md={12} xl={8}>
              <Card
                hoverable
                loading={loading}
                title={<Text style={{ color: 'var(--text-primary)' }}>{item.name}</Text>}
                extra={renderSkillStatus(item.status)}
                actions={[
                  <Button key="view" type="link" onClick={() => void openDetail(item.id)}>
                    查看详情
                  </Button>,
                ]}
                style={{ height: '100%', background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
              >
                <Space size={[0, 8]} wrap style={{ marginBottom: 12 }}>
                  <Tag color="blue">{item.skill_type || 'generator_skill'}</Tag>
                  {item.category ? <Tag color="purple">{item.category}</Tag> : null}
                  {item.input_source_mode ? <Tag>{item.input_source_mode}</Tag> : null}
                  <Tag>{item.version_count} 个版本</Tag>
                  <Tag>{item.run_count} 次运行</Tag>
                </Space>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>能力画像</Text>
                  <Paragraph style={{ color: 'var(--text-secondary)', marginBottom: 0 }}>
                    {item.capability_profile || '未声明 capability_profile'}
                  </Paragraph>
                </div>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>Skill 说明</Text>
                  <Paragraph style={{ color: 'var(--text-secondary)', minHeight: 44, marginBottom: 0 }} ellipsis={{ rows: 2 }}>
                    {item.description || '未填写说明'}
                  </Paragraph>
                </div>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>最新版本</Text>
                  <Paragraph style={{ color: 'var(--text-secondary)', marginBottom: 0 }}>
                    {item.latest_version ? `${item.latest_version.version} · ${item.latest_version.display_name}` : '暂无'}
                  </Paragraph>
                </div>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>已发布版本</Text>
                  <Paragraph style={{ color: 'var(--text-secondary)', marginBottom: 0 }}>
                    {item.published_version ? `${item.published_version.version} · ${item.published_version.display_name}` : '未发布'}
                  </Paragraph>
                </div>

                <div>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>最近更新时间</Text>
                  <div>
                    <Text style={{ color: 'var(--text-secondary)' }}>{formatDate(item.updated_at)}</Text>
                  </div>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      )}

      <Drawer
        title={selectedSkill ? `Skill 详情：${selectedSkill.name}` : 'Skill 详情'}
        open={detailOpen}
        width={960}
        onClose={() => {
          setDetailOpen(false)
          setSelectedSkillID(null)
          setSelectedSkill(null)
          setSelectedConfig(null)
          setPublishedConfigState(null)
          syncConfigForm(null)
        }}
        extra={renderDrawerActions()}
      >
        {detailLoading && !selectedSkill ? (
          <div style={{ paddingTop: 80, textAlign: 'center' }}>
            <Spin />
          </div>
        ) : !selectedSkill ? (
          <Empty description="未选择 Skill" />
        ) : (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Card size="small" style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}>
              <Space direction="vertical" size={10} style={{ width: '100%' }}>
                <Space wrap>
                  {renderSkillStatus(selectedSkill.status)}
                  {selectedSkill.input_source_mode ? <Tag>{selectedSkill.input_source_mode}</Tag> : null}
                  <Tag color="blue">{selectedSkill.skill_type || 'generator_skill'}</Tag>
                  {selectedSkill.category ? <Tag color="purple">{selectedSkill.category}</Tag> : null}
                </Space>

                <div>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>概览</Text>
                  <Paragraph style={{ color: 'var(--text-primary)', marginTop: 8, marginBottom: 8 }}>
                    {selectedSkill.description || '未填写 Skill 说明。'}
                  </Paragraph>
                  <Space direction="vertical" size={4} style={{ width: '100%' }}>
                    <Text style={{ color: 'var(--text-secondary)' }}>Slug：{selectedSkill.slug}</Text>
                    <Text style={{ color: 'var(--text-secondary)' }}>
                      能力画像：{selectedSkill.capability_profile || '未声明 capability_profile'}
                    </Text>
                    <Text style={{ color: 'var(--text-secondary)' }}>
                      评测类型：{selectedSkill.assessment_types?.length ? selectedSkill.assessment_types.join(' / ') : '未声明'}
                    </Text>
                    <Text style={{ color: 'var(--text-secondary)' }}>
                      最新版本：{selectedSkill.latest_version ? `${selectedSkill.latest_version.version} · ${selectedSkill.latest_version.display_name}` : '暂无'}
                    </Text>
                    <Text style={{ color: 'var(--text-secondary)' }}>
                      已发布版本：{selectedSkill.published_version ? `${selectedSkill.published_version.version} · ${selectedSkill.published_version.display_name}` : '未发布'}
                    </Text>
                    <Text style={{ color: 'var(--text-secondary)' }}>
                      创建时间：{formatDate(selectedSkill.created_at)}，最近更新：{formatDate(selectedSkill.updated_at)}
                    </Text>
                  </Space>
                </div>

                <InfoBlock title="权限摘要" value={selectedSkill.permissions} emptyText="未声明额外权限。" />
                <InfoBlock title="嵌入数据集摘要" value={selectedSkill.embedded_dataset_summary} emptyText="当前没有嵌入数据集摘要。" />
              </Space>
            </Card>

            <Card
              size="small"
              title="配置"
              extra={(
                <Space>
                  {selectedSkill.versions.length > 0 ? (
                    <Select
                      style={{ minWidth: 260 }}
                      placeholder="选择要查看的版本"
                      value={selectedConfigVersionID}
                      options={configVersionOptions}
                      onChange={(value) => void handleVersionConfigChange(value)}
                    />
                  ) : null}
                  <Button
                    icon={<ReloadOutlined />}
                    loading={configLoading}
                    onClick={() => (selectedSkillID ? void loadSkillConfig(selectedSkillID, selectedConfigVersionID) : undefined)}
                  >
                    刷新配置
                  </Button>
                  <Button type="primary" loading={configSaving} onClick={() => void handleSaveConfig()}>
                    保存配置
                  </Button>
                </Space>
              )}
              style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
            >
              {showPublishedConfigWarning ? (
                <Alert
                  type="warning"
                  showIcon
                  style={{ marginBottom: 16 }}
                  message="已发布版本当前配置不完整"
                  description={`当前发布版本缺少必填配置：${publishedConfigState?.missing_required_keys.join('、')}。它不会进入企业侧推荐或运行入口，补齐后会自动恢复。`}
                />
              ) : null}

              <Alert
                type="info"
                showIcon
                style={{ marginBottom: 16 }}
                message="配置说明"
                description="配置值按 Skill 级共享保存，但读取和校验始终以当前选中版本的 schema 为准。secret / password 字段不会回显真实值；如需删除已保存值，请使用“清空已保存值”。"
              />

              {configLoading && !selectedConfig ? (
                <div style={{ padding: '24px 0', textAlign: 'center' }}>
                  <Spin />
                </div>
              ) : !selectedConfig ? (
                <Empty description="当前无法加载配置视图。" />
              ) : selectedConfig.fields.length === 0 ? (
                <Empty description="该版本不需要额外配置。" />
              ) : (
                <Space direction="vertical" style={{ width: '100%' }} size="middle">
                  {!selectedConfig.config_complete ? (
                    <Alert
                      type="warning"
                      showIcon
                      message="当前选中版本配置未补齐"
                      description={`缺少必填配置：${selectedConfig.missing_required_keys.join('、')}。如果这是已发布版本，它不会进入企业侧推荐或运行。`}
                    />
                  ) : (
                    <Alert
                      type="success"
                      showIcon
                      message="当前选中版本配置完整"
                      description="该版本的必填配置已经满足，后续发布或运行时会按此 schema 注入运行时环境变量。"
                    />
                  )}

                  {selectedConfig.selected_version ? (
                    <Text style={{ color: 'var(--text-secondary)' }}>
                      当前查看版本：{selectedConfig.selected_version.version} · {selectedConfig.selected_version.display_name}（{selectedConfig.selected_version.status}）
                    </Text>
                  ) : null}

                  <Form form={configForm} layout="vertical" onValuesChange={handleConfigValuesChange}>
                    {selectedConfig.fields.map((field) => (
                      <Card
                        key={field.key}
                        size="small"
                        style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}
                        bodyStyle={{ paddingBottom: 8 }}
                      >
                        <Form.Item
                          name={field.key}
                          label={(
                            <Space wrap>
                              <Text style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{field.label}</Text>
                              {field.required ? <Tag color="red">必填</Tag> : null}
                              {field.configured ? <Tag color="green">已保存</Tag> : <Tag>未保存</Tag>}
                              {pendingConfigClears[field.key] ? <Tag color="orange">待清空</Tag> : null}
                              <Tag>{field.type}</Tag>
                            </Space>
                          )}
                          valuePropName={field.type === 'boolean' ? 'checked' : 'value'}
                          extra={(
                            <Space direction="vertical" size={2}>
                              <Text style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
                                ENV：{field.env_name}
                              </Text>
                              <Text style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
                                {field.description || '当前字段没有额外说明。'}
                              </Text>
                              {field.default !== undefined && !isSecretField(field) ? (
                                <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>
                                  默认值：{String(field.default)}
                                </Text>
                              ) : null}
                              {isSecretField(field) ? (
                                <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>
                                  留空表示不修改已保存值；如需删除，请点下方“清空已保存值”。
                                </Text>
                              ) : null}
                            </Space>
                          )}
                          style={{ marginBottom: 8 }}
                        >
                          {renderConfigInput(field)}
                        </Form.Item>

                        <Space wrap>
                          <Button
                            size="small"
                            type={pendingConfigClears[field.key] ? 'default' : 'link'}
                            disabled={!field.configured && !pendingConfigClears[field.key]}
                            onClick={() => {
                              if (pendingConfigClears[field.key]) {
                                cancelConfigClearPending(field.key)
                              } else {
                                markConfigClearPending(field.key)
                              }
                            }}
                          >
                            {pendingConfigClears[field.key] ? '取消清空' : '清空已保存值'}
                          </Button>
                        </Space>
                      </Card>
                    ))}
                  </Form>
                </Space>
              )}
            </Card>

            <Card
              size="small"
              title="版本区"
              style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
            >
              <Alert
                type="info"
                showIcon
                style={{ marginBottom: 16 }}
                message="自测规则"
                description="generator_skill 会执行 skill.yaml 声明的 execution.self_test_entrypoint；interactive_web_skill 会执行平台托管的 HTML 入口校验，并要求页面资源自包含或使用绝对 URL。服务端只保留校验报告和最多 2000 字符的日志摘录。"
              />

              {selectedSkill.versions.length === 0 ? (
                <Empty description="当前还没有版本数据。" />
              ) : (
                <Space direction="vertical" style={{ width: '100%' }} size="middle">
                  {selectedSkill.versions.map((version) => (
                    <Card
                      key={version.id}
                      size="small"
                      title={(
                        <Space wrap>
                          <Text style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{version.display_name || selectedSkill.name}</Text>
                          <Tag color="blue">{version.version}</Tag>
                        </Space>
                      )}
                      extra={renderVersionStatus(version.status)}
                      style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}
                    >
                      <Space direction="vertical" size={10} style={{ width: '100%' }}>
                        <Space size={[0, 8]} wrap>
                          {version.input_source_mode ? <Tag>{version.input_source_mode}</Tag> : null}
                          {version.execution_runtime ? <Tag color="purple">{version.execution_runtime}</Tag> : null}
                          {(version.assessment_types || []).map((assessmentType) => (
                            <Tag key={assessmentType} color="cyan">
                              {assessmentType}
                            </Tag>
                          ))}
                        </Space>

                        <Paragraph style={{ color: 'var(--text-secondary)', marginBottom: 0 }}>
                          {version.summary || '未填写版本摘要。'}
                        </Paragraph>

                        <Space direction="vertical" size={4} style={{ width: '100%' }}>
                          <Text style={{ color: 'var(--text-secondary)' }}>Manifest 版本：{version.manifest_version || '1.0'}</Text>
                          <Text style={{ color: 'var(--text-secondary)' }}>最近更新时间：{formatDate(version.updated_at)}</Text>
                        </Space>

                        <InfoBlock title="权限摘要" value={version.permissions} emptyText="未声明额外权限。" />
                        <InfoBlock title="嵌入数据集摘要" value={version.embedded_dataset_summary} emptyText="当前没有嵌入数据集摘要。" />
                        <InfoBlock title="校验报告" value={version.validation_report} emptyText="当前没有自测或校验报告。" />

                        <Space>
                          <Button
                            icon={<PlayCircleOutlined />}
                            loading={actionKey === `self-test:${version.id}`}
                            onClick={() => void handleSelfTest(version)}
                          >
                            对该版本执行自测
                          </Button>
                          <Button
                            type="primary"
                            disabled={!canPublishVersion(version.status)}
                            loading={actionKey === `publish:${version.id}`}
                            onClick={() => void handlePublish(version)}
                          >
                            发布该版本
                          </Button>
                        </Space>
                      </Space>
                    </Card>
                  ))}
                </Space>
              )}
            </Card>

            <Card
              size="small"
              title="运行区"
              extra={(
                <Button
                  size="small"
                  icon={<ReloadOutlined />}
                  loading={runRefreshing}
                  onClick={() => (selectedSkillID ? void loadSkillRuns(selectedSkillID) : undefined)}
                >
                  刷新运行记录
                </Button>
              )}
              style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
            >
              {selectedSkill.runs.length === 0 ? (
                <Empty description="当前还没有运行记录。" />
              ) : (
                <Space direction="vertical" style={{ width: '100%' }} size="middle">
                  {selectedSkill.runs.map((run) => (
                    <Card
                      key={run.id}
                      size="small"
                      title={(
                        <Space wrap>
                          <Text style={{ color: 'var(--text-primary)', fontWeight: 600 }}>
                            {run.run_type === 'self_test' ? '自测运行' : '生成运行'}
                          </Text>
                          {run.skill_version ? <Tag color="blue">{run.skill_version}</Tag> : null}
                          <Tag>{run.trigger_source || 'unknown'}</Tag>
                        </Space>
                      )}
                      extra={renderRunStatus(run.status)}
                      style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}
                    >
                      <Space direction="vertical" size={10} style={{ width: '100%' }}>
                        <Space direction="vertical" size={4} style={{ width: '100%' }}>
                          <Text style={{ color: 'var(--text-secondary)' }}>
                            开始时间：{formatDate(run.started_at)}，完成时间：{formatDate(run.completed_at)}
                          </Text>
                          <Text style={{ color: 'var(--text-secondary)' }}>
                            创建时间：{formatDate(run.created_at)}
                            {typeof run.exit_code === 'number' ? `，退出码：${run.exit_code}` : ''}
                          </Text>
                        </Space>

                        {run.error_message ? (
                          <Alert type="error" showIcon message="运行错误" description={run.error_message} />
                        ) : null}

                        <InfoBlock title="校验报告" value={run.validation_report} emptyText="当前没有校验报告。" />
                        <InfoBlock title="数据集摘要" value={run.payload_dataset_summary} emptyText="当前没有数据集摘要。" />

                        {run.run_type === 'self_test' ? (
                          <div>
                            <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>自测日志摘录</Text>
                            <Paragraph
                              style={{
                                color: 'var(--text-secondary)',
                                marginTop: 8,
                                marginBottom: 0,
                                whiteSpace: 'pre-wrap',
                                background: 'rgba(255,255,255,0.04)',
                                borderRadius: 8,
                                padding: 12,
                              }}
                            >
                              {run.log_excerpt || '当前没有可展示的自测日志摘录。'}
                            </Paragraph>
                          </div>
                        ) : null}
                      </Space>
                    </Card>
                  ))}
                </Space>
              )}
            </Card>
          </Space>
        )}
      </Drawer>

      <Drawer
        title="导入 Skill ZIP"
        open={showImportDrawer}
        width={520}
        onClose={() => {
          setShowImportDrawer(false)
          form.resetFields()
        }}
        extra={(
          <Button type="primary" icon={<UploadOutlined />} loading={importing} onClick={() => void handleImport()}>
            导入
          </Button>
        )}
      >
        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message="导入说明"
          description="当前支持 generator_skill 与 interactive_web_skill ZIP。导入后 Skill 会先处于草稿状态，需要对目标版本执行自测，自测通过后再发布。"
        />

        <Form form={form} layout="vertical">
          <Form.Item
            name="file"
            label="ZIP 文件"
            rules={[{ required: true, message: '请上传 Skill ZIP 文件' }]}
            extra="建议使用 docs/skills/generator-skill-template 对应的 ZIP 结构。"
          >
            <Upload accept=".zip" maxCount={1} beforeUpload={() => false}>
              <Button icon={<UploadOutlined />}>选择 ZIP 文件</Button>
            </Upload>
          </Form.Item>
          <Form.Item label="说明">
            <Input.TextArea
              rows={4}
              value="专家端不会展示 prompt 正文、示例原文、result_payload 或完整运行日志。导入后可以查看脱敏后的版本信息和运行摘要。"
              readOnly
            />
          </Form.Item>
        </Form>
      </Drawer>
    </div>
  )
}
