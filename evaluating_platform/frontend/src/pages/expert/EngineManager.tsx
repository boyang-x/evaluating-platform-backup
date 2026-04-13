import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  Button,
  Card,
  Checkbox,
  Col,
  Empty,
  Form,
  Input,
  Modal,
  Popconfirm,
  Row,
  Select,
  Space,
  Tag,
  Tabs,
  Typography,
  Upload,
  message,
} from 'antd'
import {
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  PlusOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import { expertService } from '../../services/expert'
import { AuxLLMConfig } from '../../components/AuxLLMConfig'
import { ComposedAttackManager } from './ComposedAttackManager'

const { Paragraph, Text, Title } = Typography
const { TextArea } = Input

interface TemplateRecord {
  id: string
  upload_batch_id?: string
  upload_batch_name?: string
  sub_type: string
  name: string
  description: string
  content: string
  created_at: string
  updated_at?: string
}

interface TemplateBatch {
  id: string
  name: string
  templates: TemplateRecord[]
  subTypes: string[]
  latestUpdatedAt: string
}

interface TemplateBatchAccumulator extends TemplateBatch {
  clusterTimestamp: number
  heuristicKey?: string
}

const templateSubTypeLabel: Record<string, string> = {
  role_play: '角色扮演',
  multilingual: '多语言增强',
  encoding_evasion: '编码规避',
}

const availableLanguages = [
  { value: 'en', label: 'English' },
  { value: 'ja', label: '日语' },
  { value: 'ko', label: '韩语' },
  { value: 'fr', label: '法语' },
  { value: 'de', label: '德语' },
  { value: 'es', label: '西班牙语' },
  { value: 'ru', label: '俄语' },
  { value: 'ar', label: '阿拉伯语' },
]

const LEGACY_BATCH_WINDOW_MS = 15000

function formatDate(value?: string) {
  if (!value) {
    return '未记录时间'
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

function toTimestamp(value?: string) {
  const time = new Date(value || '').getTime()
  return Number.isNaN(time) ? 0 : time
}

function buildTemplateBatches(templates: TemplateRecord[]) {
  const explicitCounts = new Map<string, number>()
  templates.forEach((template) => {
    if (template.upload_batch_id) {
      explicitCounts.set(template.upload_batch_id, (explicitCounts.get(template.upload_batch_id) || 0) + 1)
    }
  })

  const explicitGroups = new Map<string, TemplateBatchAccumulator>()
  const legacyGroupsByKey = new Map<string, TemplateBatchAccumulator[]>()
  const accumulators: TemplateBatchAccumulator[] = []

  const sortedTemplates = [...templates].sort((a, b) => toTimestamp(a.created_at) - toTimestamp(b.created_at))

  sortedTemplates.forEach((template) => {
    const explicitBatchID = template.upload_batch_id || ''
    const explicitBatchName = template.upload_batch_name || template.name || '未命名模板集合'
    const updatedAt = template.updated_at || template.created_at || ''
    const updatedAtMs = toTimestamp(updatedAt)

    if (explicitBatchID && (explicitCounts.get(explicitBatchID) || 0) > 1) {
      const existing = explicitGroups.get(explicitBatchID)
      if (existing) {
        existing.templates.push(template)
        if (!existing.subTypes.includes(template.sub_type)) {
          existing.subTypes.push(template.sub_type)
        }
        if (updatedAtMs > toTimestamp(existing.latestUpdatedAt)) {
          existing.latestUpdatedAt = updatedAt
          existing.clusterTimestamp = updatedAtMs
        }
        return
      }

      const created: TemplateBatchAccumulator = {
        id: explicitBatchID,
        name: explicitBatchName,
        templates: [template],
        subTypes: [template.sub_type],
        latestUpdatedAt: updatedAt,
        clusterTimestamp: updatedAtMs,
      }
      explicitGroups.set(explicitBatchID, created)
      accumulators.push(created)
      return
    }

    const heuristicName = (template.upload_batch_name || template.name || '').trim().toLowerCase()
    const heuristicKey = `${heuristicName}::${template.sub_type}`
    const candidates = legacyGroupsByKey.get(heuristicKey) || []
    const matched = [...candidates].reverse().find((group) => Math.abs(updatedAtMs - group.clusterTimestamp) <= LEGACY_BATCH_WINDOW_MS)

    if (matched) {
      matched.templates.push(template)
      if (updatedAtMs > toTimestamp(matched.latestUpdatedAt)) {
        matched.latestUpdatedAt = updatedAt
        matched.clusterTimestamp = updatedAtMs
      }
      return
    }

    const created: TemplateBatchAccumulator = {
      id: explicitBatchID || `legacy-${template.id}`,
      name: explicitBatchName,
      templates: [template],
      subTypes: [template.sub_type],
      latestUpdatedAt: updatedAt,
      clusterTimestamp: updatedAtMs,
      heuristicKey,
    }
    legacyGroupsByKey.set(heuristicKey, [...candidates, created])
    accumulators.push(created)
  })

  return accumulators
    .map((batch) => ({
      id: batch.id,
      name: batch.name,
      subTypes: batch.subTypes,
      latestUpdatedAt: batch.latestUpdatedAt,
      templates: [...batch.templates].sort((a, b) => toTimestamp(b.updated_at || b.created_at) - toTimestamp(a.updated_at || a.created_at)),
    }))
    .sort((a, b) => toTimestamp(b.latestUpdatedAt) - toTimestamp(a.latestUpdatedAt))
}

export function EngineManager() {
  const [templates, setTemplates] = useState<TemplateRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [filterType, setFilterType] = useState<string>('')
  const [showTemplateModal, setShowTemplateModal] = useState(false)
  const [showCSVUpload, setShowCSVUpload] = useState(false)
  const [showBatchModal, setShowBatchModal] = useState(false)
  const [selectedBatch, setSelectedBatch] = useState<TemplateBatch | null>(null)
  const [editingTemplate, setEditingTemplate] = useState<TemplateRecord | null>(null)
  const [enhanceStrategy, setEnhanceStrategy] = useState<string>('none')
  const [targetLanguages, setTargetLanguages] = useState<string[]>(['en', 'ja', 'ko'])
  const [templateForm] = Form.useForm()
  const [csvForm] = Form.useForm()

  const loadTemplates = useCallback(() => {
    setLoading(true)
    expertService
      .listTemplates({ sub_type: filterType || undefined, limit: 200 })
      .then((res) => setTemplates(res.items || []))
      .catch(() => message.error('加载模板失败'))
      .finally(() => setLoading(false))
  }, [filterType])

  useEffect(() => {
    loadTemplates()
  }, [loadTemplates])

  const templateBatches = useMemo(() => buildTemplateBatches(templates), [templates])

  const refreshSelectedBatch = useCallback((batchID: string) => {
    const nextBatch = templateBatches.find((item) => item.id === batchID) || null
    setSelectedBatch(nextBatch)
    if (!nextBatch) {
      setShowBatchModal(false)
    }
  }, [templateBatches])

  useEffect(() => {
    if (selectedBatch) {
      refreshSelectedBatch(selectedBatch.id)
    }
  }, [templateBatches, refreshSelectedBatch, selectedBatch])

  const handleSaveTemplate = async () => {
    try {
      const values = await templateForm.validateFields()
      const payload = {
        sub_type: values.sub_type,
        name: values.name,
        description: values.description || '',
        content: values.content,
      }
      if (editingTemplate) {
        await expertService.updateTemplate(editingTemplate.id, payload)
        message.success('模板已更新')
      } else {
        await expertService.createTemplate(payload)
        message.success('模板已创建')
      }
      setShowTemplateModal(false)
      setEditingTemplate(null)
      templateForm.resetFields()
      loadTemplates()
    } catch (error: unknown) {
      message.error((error as Error).message || '保存模板失败')
    }
  }

  const handleEditTemplate = (record: TemplateRecord) => {
    setEditingTemplate(record)
    templateForm.setFieldsValue({
      sub_type: record.sub_type,
      name: record.name,
      description: record.description,
      content: record.content,
    })
    setShowTemplateModal(true)
  }

  const handleDeleteTemplate = async (id: string) => {
    try {
      await expertService.deleteTemplate(id)
      message.success('模板已删除')
      loadTemplates()
    } catch (error: unknown) {
      message.error((error as Error).message || '删除模板失败')
    }
  }

  const handleDeleteBatch = async (batch: TemplateBatch) => {
    try {
      await Promise.all(batch.templates.map((template) => expertService.deleteTemplate(template.id)))
      message.success(`模板集合“${batch.name}”已删除`)
      if (selectedBatch?.id === batch.id) {
        setShowBatchModal(false)
        setSelectedBatch(null)
      }
      loadTemplates()
    } catch (error: unknown) {
      message.error((error as Error).message || '删除模板集合失败')
    }
  }

  const handleCSVUpload = async () => {
    try {
      const values = await csvForm.validateFields()
      const formData = new FormData()
      formData.append('sub_type', values.sub_type)
      formData.append('name', values.name)
      formData.append('file', values.file.file.originFileObj || values.file.file)
      await expertService.uploadTemplateCSV(formData)
      message.success('模板集合上传成功')
      setShowCSVUpload(false)
      csvForm.resetFields()
      loadTemplates()
    } catch (error: unknown) {
      message.error((error as Error).message || '上传模板失败')
    }
  }

  const templateTab = (
    <div>
      <Space style={{ marginBottom: 16 }} wrap>
        <Select
          value={filterType}
          onChange={setFilterType}
          style={{ width: 180 }}
          options={[
            { value: '', label: '全部类型' },
            ...Object.entries(templateSubTypeLabel).map(([value, label]) => ({ value, label })),
          ]}
        />
        <Button
          icon={<UploadOutlined />}
          onClick={() => {
            csvForm.resetFields()
            setShowCSVUpload(true)
          }}
        >
          上传模板集合
        </Button>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={() => {
            setEditingTemplate(null)
            templateForm.resetFields()
            setShowTemplateModal(true)
          }}
        >
          新建单条模板
        </Button>
      </Space>

      {templateBatches.length === 0 ? (
        <Card loading={loading}>
          <Empty description="当前还没有模板集合，上传一次 CSV 后会在这里按批次展示。" />
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {templateBatches.map((batch) => (
            <Col key={batch.id} xs={24} md={12} xl={8}>
              <Card
                loading={loading}
                hoverable
                title={<Text style={{ color: 'var(--text-primary)' }}>{batch.name}</Text>}
                extra={<Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>{batch.templates.length} 条模板</Text>}
                actions={[
                  <Button key="view" type="link" icon={<EyeOutlined />} onClick={() => { setSelectedBatch(batch); setShowBatchModal(true) }}>
                    查看集合
                  </Button>,
                  <Popconfirm
                    key="delete"
                    title={`确认删除模板集合“${batch.name}”吗？`}
                    description="会删除这个集合下的全部模板。"
                    onConfirm={() => handleDeleteBatch(batch)}
                  >
                    <Button type="link" danger icon={<DeleteOutlined />}>
                      删除整组
                    </Button>
                  </Popconfirm>,
                ]}
                style={{ height: '100%', background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
              >
                <Space size={[0, 8]} wrap style={{ marginBottom: 12 }}>
                  {batch.subTypes.map((subType) => (
                    <Tag key={subType} color="purple">
                      {templateSubTypeLabel[subType] || subType}
                    </Tag>
                  ))}
                </Space>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>集合说明</Text>
                  <Paragraph style={{ color: 'var(--text-secondary)', minHeight: 44, marginBottom: 0 }} ellipsis={{ rows: 2 }}>
                    {batch.templates[0]?.description || `这个集合包含 ${batch.templates.length} 条模板，会在内部作为同一批次管理。`}
                  </Paragraph>
                </div>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>最近更新时间</Text>
                  <div>
                    <Text style={{ color: 'var(--text-secondary)' }}>{formatDate(batch.latestUpdatedAt)}</Text>
                  </div>
                </div>

                <div>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>模板编号预览</Text>
                  <Space wrap style={{ marginTop: 8 }}>
                    {batch.templates.slice(0, 8).map((template, index) => (
                      <Tag key={template.id}>{`模板 ${index + 1}`}</Tag>
                    ))}
                    {batch.templates.length > 8 ? <Tag>{`+${batch.templates.length - 8}`}</Tag> : null}
                  </Space>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      )}
    </div>
  )

  const enhanceTab = (
    <Space direction="vertical" style={{ width: '100%' }} size="middle">
      <div>
        <Text style={{ color: 'var(--text-secondary)', marginBottom: 8, display: 'block' }}>增强策略</Text>
        <Select
          value={enhanceStrategy}
          onChange={setEnhanceStrategy}
          style={{ width: 240 }}
          options={[
            { value: 'none', label: '不增强，保持原始内容' },
            { value: 'multilingual', label: '多语言增强' },
          ]}
        />
      </div>

      {enhanceStrategy === 'multilingual' && (
        <div>
          <Text style={{ color: 'var(--text-secondary)', marginBottom: 8, display: 'block' }}>目标语言</Text>
          <Checkbox.Group value={targetLanguages} onChange={(value) => setTargetLanguages(value as string[])}>
            <Space wrap>
              {availableLanguages.map((lang) => (
                <Checkbox key={lang.value} value={lang.value}>
                  {lang.label}
                </Checkbox>
              ))}
            </Space>
          </Checkbox.Group>
        </div>
      )}

      <AuxLLMConfig />
    </Space>
  )

  return (
    <div>
      <div style={{ marginBottom: 20 }}>
        <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>
          引擎管理
        </Title>
        <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
          模板管理按上传批次展示，一张卡片代表一次上传的整批模板；历史数据也会尽量按同一波上传自动归并。
        </Text>
      </div>

      <Tabs
        defaultActiveKey="templates"
        items={[
          { key: 'templates', label: '模板管理', children: templateTab },
          { key: 'composed', label: '已组合攻击', children: <ComposedAttackManager /> },
          { key: 'enhance', label: '增强引擎', children: enhanceTab },
        ]}
      />

      <Modal
        title={editingTemplate ? '编辑模板' : '新建模板'}
        open={showTemplateModal}
        onCancel={() => {
          setShowTemplateModal(false)
          setEditingTemplate(null)
          templateForm.resetFields()
        }}
        onOk={handleSaveTemplate}
        okText="保存"
        width={720}
      >
        <Form form={templateForm} layout="vertical">
          <Form.Item name="sub_type" label="模板类型" rules={[{ required: true, message: '请选择模板类型' }]}> 
            <Select options={Object.entries(templateSubTypeLabel).map(([value, label]) => ({ value, label }))} />
          </Form.Item>
          <Form.Item name="name" label="模板名称" rules={[{ required: true, message: '请输入模板名称' }]}> 
            <Input placeholder="如：合规检测角色扮演模板" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input placeholder="简要说明这个模板的用途" />
          </Form.Item>
          <Form.Item
            name="content"
            label="模板内容"
            rules={[{ required: true, message: '请输入模板内容' }]}
            extra="模板内容会在内部与样本问题拼接后，再送往被测 LLM。"
          >
            <TextArea rows={8} placeholder="请输入模板内容，例如角色设定、上下文限制或测试前缀。" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="上传模板集合"
        open={showCSVUpload}
        onCancel={() => {
          setShowCSVUpload(false)
          csvForm.resetFields()
        }}
        onOk={handleCSVUpload}
        okText="上传"
        width={560}
      >
        <Form form={csvForm} layout="vertical">
          <Form.Item name="name" label="集合名称" rules={[{ required: true, message: '请输入集合名称' }]}> 
            <Input placeholder="如：2026-04 合规测试模板集" />
          </Form.Item>
          <Form.Item name="sub_type" label="模板类型" rules={[{ required: true, message: '请选择模板类型' }]}> 
            <Select options={Object.entries(templateSubTypeLabel).map(([value, label]) => ({ value, label }))} />
          </Form.Item>
          <Form.Item
            name="file"
            label="CSV 文件"
            rules={[{ required: true, message: '请上传 CSV 文件' }]}
            extra="格式：两列（序号, 模板内容），无需表头。一次上传会作为一个模板集合管理。"
          >
            <Upload accept=".csv" maxCount={1} beforeUpload={() => false}>
              <Button icon={<UploadOutlined />}>选择 CSV 文件</Button>
            </Upload>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={selectedBatch ? `模板集合：${selectedBatch.name}` : '模板集合详情'}
        open={showBatchModal}
        onCancel={() => {
          setShowBatchModal(false)
          setSelectedBatch(null)
        }}
        footer={null}
        width={920}
      >
        {!selectedBatch ? null : (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Card size="small" style={{ background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
              <Space direction="vertical" size={4} style={{ width: '100%' }}>
                <Text style={{ color: 'var(--text-secondary)' }}>模板数量：{selectedBatch.templates.length}</Text>
                <Text style={{ color: 'var(--text-secondary)' }}>最近更新时间：{formatDate(selectedBatch.latestUpdatedAt)}</Text>
                <Space size={[0, 8]} wrap>
                  {selectedBatch.subTypes.map((subType) => (
                    <Tag key={subType} color="purple">
                      {templateSubTypeLabel[subType] || subType}
                    </Tag>
                  ))}
                </Space>
              </Space>
            </Card>

            <Row gutter={[12, 12]}>
              {selectedBatch.templates.map((template, index) => (
                <Col key={template.id} span={24}>
                  <Card
                    size="small"
                    title={<Text style={{ color: 'var(--text-primary)' }}>模板 {index + 1}</Text>}
                    extra={
                      <Space>
                        <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEditTemplate(template)}>
                          编辑
                        </Button>
                        <Popconfirm title="确认删除这条模板吗？" onConfirm={() => handleDeleteTemplate(template.id)}>
                          <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                            删除
                          </Button>
                        </Popconfirm>
                      </Space>
                    }
                    style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
                  >
                    <Space direction="vertical" size={6} style={{ width: '100%' }}>
                      <Text style={{ color: 'var(--text-secondary)' }}>
                        类型：{templateSubTypeLabel[template.sub_type] || template.sub_type}
                      </Text>
                      {template.description ? (
                        <Paragraph style={{ color: 'var(--text-secondary)', marginBottom: 0 }}>{template.description}</Paragraph>
                      ) : null}
                      <Paragraph
                        style={{ color: 'var(--text-primary)', marginBottom: 0, whiteSpace: 'pre-wrap' }}
                        ellipsis={{ rows: 4, expandable: true, symbol: '展开' }}
                      >
                        {template.content}
                      </Paragraph>
                    </Space>
                  </Card>
                </Col>
              ))}
            </Row>
          </Space>
        )}
      </Modal>
    </div>
  )
}
