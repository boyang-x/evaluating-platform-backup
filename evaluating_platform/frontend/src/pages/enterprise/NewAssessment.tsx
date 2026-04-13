import { useState } from 'react'
import {
  Form, Input, Select, Button, Steps, Card, Typography, Space,
  Divider, Tag, Switch, message, Row, Col
} from 'antd'
import {
  ApiOutlined, AimOutlined, ToolOutlined, PlayCircleOutlined,
  CheckCircleOutlined, LoadingOutlined,
} from '@ant-design/icons'
import { useNavigate, useLocation } from 'react-router-dom'
import { assessmentService } from '../../services/assessment'

const { Title, Text, Paragraph } = Typography
const { TextArea } = Input

const builtinTools = [
  { name: 'prompt_injection', category: 'llm', label: '提示词注入', desc: '检测 LLM 对注入攻击的防御', price: '¥0.5/次' },
  { name: 'jailbreak', category: 'llm', label: '越狱测试', desc: '测试安全护栏绕过能力', price: '¥0.8/次' },
  { name: 'compliance_check', category: 'compliance', label: '合规检查', desc: '数据安全与内容合规检测', price: '¥0.3/次' },
  { name: 'goal_hijacking', category: 'agent', label: '目标劫持', desc: '测试 Agent 目标被篡改的风险', price: '¥1.0/次' },
  { name: 'tool_poisoning', category: 'agent', label: '工具投毒', desc: '测试工具调用链注入攻击', price: '¥1.2/次' },
]

const categoryColor: Record<string, string> = {
  llm: '#4d96ff',
  agent: '#ff7a45',
  compliance: '#52c41a',
}

const categoryLabel: Record<string, string> = {
  llm: '大模型',
  agent: 'AI Agent',
  compliance: '合规',
}

interface LocationState {
  template_id?: string
  template_name?: string
}

export function NewAssessment() {
  const navigate = useNavigate()
  const location = useLocation()
  const state = (location.state as LocationState) || {}

  const [currentStep, setCurrentStep] = useState(0)
  const [selectedTools, setSelectedTools] = useState<string[]>(['prompt_injection', 'jailbreak', 'compliance_check'])
  const [useAutoSelect, setUseAutoSelect] = useState(true)
  const [isRunning, setIsRunning] = useState(false)
  const [form] = Form.useForm()

  const toggleTool = (name: string) => {
    setSelectedTools(prev =>
      prev.includes(name) ? prev.filter(t => t !== name) : [...prev, name]
    )
  }

  const estimatedCost = builtinTools
    .filter(t => selectedTools.includes(t.name))
    .reduce((sum, t) => sum + parseFloat(t.price.replace('¥', '').replace('/次', '')), 0)
    .toFixed(1)

  const handleLaunch = async () => {
    try {
      await form.validateFields()
      const values = form.getFieldsValue()
      setIsRunning(true)

      const data = {
        name: values.name,
        goal: values.goal,
        target_type: values.targetType,
        target_url: values.targetUrl,
        target_key: values.targetKey || '',
        target_model: values.targetModel || '',
        template_id: state.template_id || '',
      }

      await assessmentService.create(data)
      message.success('评估任务已启动，正在执行中...')
      navigate('/enterprise/assessments')
    } catch (err: unknown) {
      const e = err as { status?: number; message?: string }
      if (e?.status === 402 || (err instanceof Error && err.message.includes('余额'))) {
        message.error('余额不足，请充值后重试')
        navigate('/enterprise/billing')
      } else {
        message.error((err instanceof Error ? err.message : '') || '请先填写完整的目标系统信息')
      }
    } finally {
      setIsRunning(false)
    }
  }

  const steps = [
    { title: '目标系统', icon: <ApiOutlined /> },
    { title: '评估目标', icon: <AimOutlined /> },
    { title: '选择工具', icon: <ToolOutlined /> },
    { title: '确认执行', icon: <PlayCircleOutlined /> },
  ]

  return (
    <div style={{ maxWidth: 860, margin: '0 auto' }}>
      <div style={{ marginBottom: 24 }}>
        <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>发起新评估</Title>
        <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>
          配置目标系统，AI Agent 将自动规划并执行安全评估
          {state.template_name && <span style={{ color: '#4d96ff', marginLeft: 8 }}>· 使用模板: {state.template_name}</span>}
        </Text>
      </div>

      <Steps current={currentStep} items={steps} style={{ marginBottom: 32 }} onChange={setCurrentStep} />

      <Form form={form} layout="vertical">
        {currentStep === 0 && (
          <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}>
            <Title level={5} style={{ color: 'var(--text-primary)', marginTop: 0 }}>
              <ApiOutlined style={{ marginRight: 8, color: '#4d96ff' }} />目标系统 API 配置
            </Title>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item name="targetType" label={<Text style={{ color: 'var(--text-secondary)' }}>系统类型</Text>}
                  rules={[{ required: true }]}>
                  <Select options={[
                    { value: 'openai', label: 'OpenAI Compatible API' },
                    { value: 'agent', label: 'AI Agent 系统' },
                    { value: 'custom', label: '自定义 HTTP 接口' },
                  ]} placeholder="选择接口类型" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item name="targetModel" label={<Text style={{ color: 'var(--text-secondary)' }}>模型名称</Text>}>
                  <Input placeholder="gpt-4o / claude-3-5-sonnet / ..." />
                </Form.Item>
              </Col>
            </Row>
            <Form.Item name="targetUrl" label={<Text style={{ color: 'var(--text-secondary)' }}>API Endpoint</Text>}
              rules={[{ required: true, message: '请输入 API 地址' }]}>
              <Input placeholder="https://api.openai.com/v1" />
            </Form.Item>
            <Form.Item name="targetKey" label={<Text style={{ color: 'var(--text-secondary)' }}>API Key</Text>}>
              <Input.Password placeholder="sk-..." />
            </Form.Item>
            <div style={{ textAlign: 'right' }}>
              <Button type="primary" onClick={() => setCurrentStep(1)}>下一步</Button>
            </div>
          </Card>
        )}

        {currentStep === 1 && (
          <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}>
            <Title level={5} style={{ color: 'var(--text-primary)', marginTop: 0 }}>
              <AimOutlined style={{ marginRight: 8, color: '#4d96ff' }} />描述评估目标
            </Title>
            <Form.Item name="name" label={<Text style={{ color: 'var(--text-secondary)' }}>评估名称</Text>}
              rules={[{ required: true }]}>
              <Input placeholder="如：某 AI 助手安全评估 2026Q1" />
            </Form.Item>
            <Form.Item name="goal" label={<Text style={{ color: 'var(--text-secondary)' }}>评估目标（自然语言描述）</Text>}
              rules={[{ required: true }]}>
              <TextArea rows={4} placeholder="例如：测试该模型是否容易受到提示词注入攻击..." />
            </Form.Item>
            <Paragraph style={{ color: 'var(--text-muted)', fontSize: 12 }}>
              AI Agent 将根据你的描述自动规划评估步骤，选择合适的工具进行检测
            </Paragraph>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Button onClick={() => setCurrentStep(0)}>上一步</Button>
              <Button type="primary" onClick={() => setCurrentStep(2)}>下一步</Button>
            </div>
          </Card>
        )}

        {currentStep === 2 && (
          <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
              <Title level={5} style={{ color: 'var(--text-primary)', margin: 0 }}>
                <ToolOutlined style={{ marginRight: 8, color: '#4d96ff' }} />选择评估工具
              </Title>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>AI 自动选择</Text>
                <Switch checked={useAutoSelect} onChange={setUseAutoSelect} />
              </div>
            </div>
            {useAutoSelect && (
              <div style={{
                padding: '10px 14px',
                background: 'rgba(26, 109, 255, 0.08)',
                border: '1px solid rgba(26, 109, 255, 0.2)',
                borderRadius: 6,
                marginBottom: 16,
              }}>
                <Text style={{ color: '#4d96ff', fontSize: 13 }}>
                  AI Agent 将根据评估目标自动选择最合适的工具组合，你也可以手动调整
                </Text>
              </div>
            )}
            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>
              {builtinTools.map(tool => {
                const selected = selectedTools.includes(tool.name)
                return (
                  <div key={tool.name} onClick={() => toggleTool(tool.name)} style={{
                    padding: '14px 16px',
                    background: selected ? 'rgba(26, 109, 255, 0.1)' : 'var(--bg-surface)',
                    border: `1px solid ${selected ? 'rgba(26, 109, 255, 0.5)' : 'var(--border-color)'}`,
                    borderRadius: 8, cursor: 'pointer', transition: 'all 0.2s',
                  }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                        <Text style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{tool.label}</Text>
                        <Tag color={categoryColor[tool.category]} style={{ fontSize: 11 }}>
                          {categoryLabel[tool.category]}
                        </Tag>
                      </div>
                      {selected && <CheckCircleOutlined style={{ color: '#4d96ff' }} />}
                    </div>
                    <Text style={{ color: 'var(--text-secondary)', fontSize: 12 }}>{tool.desc}</Text>
                    <div style={{ marginTop: 6 }}>
                      <Text style={{ color: 'var(--text-muted)', fontSize: 11 }}>{tool.price}</Text>
                    </div>
                  </div>
                )
              })}
            </div>
            <Divider style={{ borderColor: 'var(--border-color)' }} />
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Text style={{ color: 'var(--text-secondary)' }}>
                已选 {selectedTools.length} 个工具，预估费用：
                <span style={{ color: '#4d96ff', fontWeight: 600, marginLeft: 4 }}>¥{estimatedCost}</span>
              </Text>
              <Space>
                <Button onClick={() => setCurrentStep(1)}>上一步</Button>
                <Button type="primary" onClick={() => setCurrentStep(3)}>下一步</Button>
              </Space>
            </div>
          </Card>
        )}

        {currentStep === 3 && (
          <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}>
            <Title level={5} style={{ color: 'var(--text-primary)', marginTop: 0 }}>
              <PlayCircleOutlined style={{ marginRight: 8, color: '#4d96ff' }} />确认执行
            </Title>
            <div style={{
              padding: 20, background: 'var(--bg-surface)',
              border: '1px solid var(--border-color)', borderRadius: 8, marginBottom: 20,
            }}>
              <Row gutter={[16, 12]}>
                <Col span={12}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>评估名称</Text>
                  <div><Text style={{ color: 'var(--text-primary)' }}>
                    {form.getFieldValue('name') || '（未填写）'}
                  </Text></div>
                </Col>
                <Col span={12}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>目标类型</Text>
                  <div><Text style={{ color: 'var(--text-primary)' }}>
                    {form.getFieldValue('targetType') || '（未选择）'}
                  </Text></div>
                </Col>
                <Col span={24}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>工具清单</Text>
                  <div style={{ marginTop: 4, display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                    {selectedTools.map(t => {
                      const tool = builtinTools.find(bt => bt.name === t)
                      return tool ? (
                        <Tag key={t} color={categoryColor[tool.category]}>{tool.label}</Tag>
                      ) : null
                    })}
                  </div>
                </Col>
                <Col span={12}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>预估费用</Text>
                  <div><Text style={{ color: '#4d96ff', fontWeight: 600 }}>¥{estimatedCost}</Text></div>
                </Col>
              </Row>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <Button onClick={() => setCurrentStep(2)}>上一步</Button>
              <Button
                type="primary" size="large"
                icon={isRunning ? <LoadingOutlined /> : <PlayCircleOutlined />}
                onClick={handleLaunch}
                loading={isRunning}
                style={{ minWidth: 140 }}
              >
                {isRunning ? '启动中...' : '立即执行评估'}
              </Button>
            </div>
          </Card>
        )}
      </Form>
    </div>
  )
}
