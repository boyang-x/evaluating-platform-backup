import { useEffect, useState } from 'react'
import {
  Button,
  Card,
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
  Typography,
  Upload,
  message,
} from 'antd'
import {
  DeleteOutlined,
  EyeOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import { expertService } from '../../services/expert'

const { Paragraph, Text, Title } = Typography
const { TextArea } = Input

interface SampleRecord {
  id: string
  sub_type: string
  name: string
  description: string
  sample_count: number
  file_hash: string
  file_size: number
  created_at: string
}

interface Payload {
  index: number
  data: string
}

const subTypeLabel: Record<string, string> = {
  direct_injection: '直接注入',
  malicious_instruction: '恶意指令',
  compliance_detection: '合规检测',
  malicious_poisoning: '恶意投毒',
}

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

function formatSize(value: number) {
  if (!value) {
    return '0 KB'
  }
  return `${(value / 1024).toFixed(1)} KB`
}

export function SampleManager() {
  const [samples, setSamples] = useState<SampleRecord[]>([])
  const [loading, setLoading] = useState(true)
  const [showUpload, setShowUpload] = useState(false)
  const [previewData, setPreviewData] = useState<Payload[] | null>(null)
  const [previewName, setPreviewName] = useState('')
  const [filterType, setFilterType] = useState<string>('')
  const [expandedRows, setExpandedRows] = useState<Record<number, boolean>>({})
  const [form] = Form.useForm()

  const loadSamples = () => {
    setLoading(true)
    expertService
      .listSamples({ sub_type: filterType || undefined, limit: 100 })
      .then((res) => setSamples(res.items || []))
      .catch(() => message.error('加载样本失败'))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    loadSamples()
  }, [filterType])

  const handleUpload = async () => {
    try {
      const values = await form.validateFields()
      const formData = new FormData()
      formData.append('sub_type', values.sub_type)
      formData.append('name', values.name)
      formData.append('description', values.description || '')
      formData.append('file', values.file.file.originFileObj || values.file.file)
      await expertService.uploadSample(formData)
      message.success('样本上传成功')
      setShowUpload(false)
      form.resetFields()
      loadSamples()
    } catch (error: unknown) {
      message.error((error as Error).message || '上传样本失败')
    }
  }

  const handlePreview = async (id: string, name: string, sampleCount: number) => {
    try {
      const res = await expertService.previewSample(id, sampleCount || 20)
      setPreviewData(res.preview || [])
      setPreviewName(name)
    } catch {
      message.error('预览样本失败')
    }
  }

  const handleDelete = async (id: string) => {
    try {
      await expertService.deleteSample(id)
      message.success('样本已删除')
      loadSamples()
    } catch (error: unknown) {
      message.error((error as Error).message || '删除样本失败')
    }
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
        <div>
          <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>
            攻击样本管理
          </Title>
          <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
            一张卡片代表一份上传样本集，方便和模板集合保持统一的管理样式。
          </Text>
        </div>
        <Space>
          <Select
            value={filterType}
            onChange={setFilterType}
            style={{ width: 160 }}
            options={[
              { value: '', label: '全部类型' },
              ...Object.entries(subTypeLabel).map(([value, label]) => ({ value, label })),
            ]}
          />
          <Button type="primary" icon={<UploadOutlined />} onClick={() => setShowUpload(true)}>
            上传样本集
          </Button>
        </Space>
      </div>

      {samples.length === 0 ? (
        <Card loading={loading}>
          <Empty description="当前还没有样本集，上传 CSV 后会以卡片方式展示在这里。" />
        </Card>
      ) : (
        <Row gutter={[16, 16]}>
          {samples.map((sample) => (
            <Col key={sample.id} xs={24} md={12} xl={8}>
              <Card
                loading={loading}
                hoverable
                title={<Text style={{ color: 'var(--text-primary)' }}>{sample.name}</Text>}
                extra={<Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>{sample.sample_count} 条样本</Text>}
                actions={[
                  <Button key="preview" type="link" icon={<EyeOutlined />} onClick={() => handlePreview(sample.id, sample.name, sample.sample_count)}>
                    预览样本
                  </Button>,
                  <Popconfirm key="delete" title={`确认删除样本集“${sample.name}”吗？`} onConfirm={() => handleDelete(sample.id)}>
                    <Button type="link" danger icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>,
                ]}
                style={{ height: '100%', background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
              >
                <Space size={[0, 8]} wrap style={{ marginBottom: 12 }}>
                  <Tag color="blue">{subTypeLabel[sample.sub_type] || sample.sub_type}</Tag>
                  <Tag>{formatSize(sample.file_size)}</Tag>
                </Space>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>样本集说明</Text>
                  <Paragraph style={{ color: 'var(--text-secondary)', minHeight: 44, marginBottom: 0 }} ellipsis={{ rows: 2 }}>
                    {sample.description || `这个样本集包含 ${sample.sample_count} 条攻击样本。`}
                  </Paragraph>
                </div>

                <div style={{ marginBottom: 12 }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>上传时间</Text>
                  <div>
                    <Text style={{ color: 'var(--text-secondary)' }}>{formatDate(sample.created_at)}</Text>
                  </div>
                </div>

                <div>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>文件指纹</Text>
                  <Paragraph style={{ color: 'var(--text-secondary)', marginBottom: 0 }} ellipsis={{ rows: 1 }}>
                    {sample.file_hash || '未记录'}
                  </Paragraph>
                </div>
              </Card>
            </Col>
          ))}
        </Row>
      )}

      <Modal
        title="上传攻击样本集"
        open={showUpload}
        onCancel={() => {
          setShowUpload(false)
          form.resetFields()
        }}
        onOk={handleUpload}
        okText="上传"
        width={540}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="sub_type" label="样本类型" rules={[{ required: true, message: '请选择样本类型' }]}> 
            <Select options={Object.entries(subTypeLabel).map(([value, label]) => ({ value, label }))} />
          </Form.Item>
          <Form.Item name="name" label="样本集名称" rules={[{ required: true, message: '请输入样本集名称' }]}> 
            <Input placeholder="如：内容合规检测样本集" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={2} placeholder="简要描述样本内容" />
          </Form.Item>
          <Form.Item
            name="file"
            label="CSV 文件"
            rules={[{ required: true, message: '请上传 CSV 文件' }]}
            extra="支持 UTF-8、GBK、GB18030、UTF-16 编码。格式为两列：序号、内容。"
          >
            <Upload accept=".csv" maxCount={1} beforeUpload={() => false}>
              <Button icon={<UploadOutlined />}>选择 CSV 文件</Button>
            </Upload>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`样本预览：${previewName}`}
        open={!!previewData}
        onCancel={() => {
          setPreviewData(null)
          setExpandedRows({})
        }}
        footer={null}
        width={880}
      >
        {!previewData || previewData.length === 0 ? (
          <Empty description="暂无样本数据" />
        ) : (
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            {previewData.map((payload) => {
              const safeText = payload.data || ''
              const isExpanded = expandedRows[payload.index]
              const shouldTruncate = safeText.length > 500
              const displayText = shouldTruncate && !isExpanded ? `${safeText.slice(0, 500)}...` : safeText

              return (
                <Card
                  key={payload.index}
                  size="small"
                  title={<Text style={{ color: 'var(--text-primary)' }}>样本 {payload.index}</Text>}
                  style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
                >
                  <Paragraph style={{ color: 'var(--text-primary)', marginBottom: shouldTruncate ? 8 : 0, whiteSpace: 'pre-wrap' }}>
                    {displayText}
                  </Paragraph>
                  {shouldTruncate ? (
                    <Button
                      type="link"
                      size="small"
                      onClick={() => setExpandedRows((prev) => ({ ...prev, [payload.index]: !prev[payload.index] }))}
                      style={{ paddingLeft: 0 }}
                    >
                      {isExpanded ? '收起' : '展开'}
                    </Button>
                  ) : null}
                </Card>
              )
            })}
          </Space>
        )}
      </Modal>
    </div>
  )
}
