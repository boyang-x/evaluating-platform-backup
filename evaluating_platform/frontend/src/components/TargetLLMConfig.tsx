import { useState, useEffect } from 'react'
import { Card, Form, Input, Button, Typography, Space, message, Tag, Select } from 'antd'
import { ApiOutlined, CheckCircleOutlined, SettingOutlined } from '@ant-design/icons'
import { enterpriseService } from '../services/enterprise'

const { Text } = Typography

interface TargetLLMFormValues {
  base_url: string
  api_key: string
  model?: string
  connector_type?: string
}

export function TargetLLMConfig() {
  const [form] = Form.useForm<TargetLLMFormValues>()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [testing, setTesting] = useState(false)
  const [configured, setConfigured] = useState(false)

  useEffect(() => {
    enterpriseService.getTargetLLMConfig()
      .then(data => {
        form.setFieldsValue({
          base_url: data.base_url || '',
          api_key: '', // API key is masked from server
          model: data.model || '',
          connector_type: data.connector_type || 'openai',
        })
        setConfigured(true)
      })
      .catch(() => { setConfigured(false) })
      .finally(() => setLoading(false))
  }, [form])

  const handleSave = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      await enterpriseService.updateTargetLLMConfig({
        base_url: values.base_url,
        api_key: values.api_key,
        model: values.model,
        connector_type: values.connector_type,
      })
      message.success('被测 LLM 配置已保存')
      setConfigured(true)
    } catch (err: unknown) {
      message.error((err as Error).message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleTest = async () => {
    try {
      const values = await form.validateFields()
      setTesting(true)
      await enterpriseService.testTargetLLMConnection({
        base_url: values.base_url,
        api_key: values.api_key,
        model: values.model,
        connector_type: values.connector_type,
      })
      message.success('连接测试成功')
    } catch (err: unknown) {
      message.error((err as Error).message || '连接测试失败')
    } finally {
      setTesting(false)
    }
  }

  if (loading) return null

  return (
    <Card
      title={
        <Space>
          <SettingOutlined />
          <span>被测 LLM 配置</span>
          {configured && <Tag color="success" icon={<CheckCircleOutlined />}>已配置</Tag>}
        </Space>
      }
      extra={
        <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>
          需要进行安全评估的目标 LLM
        </Text>
      }
    >
      <Form form={form} layout="vertical" style={{ maxWidth: 480 }}>
        <Form.Item name="base_url" label="Base URL"
          rules={[{ required: true, message: '请输入 LLM API 地址' }]}>
          <Input placeholder="https://api.openai.com/v1" />
        </Form.Item>
        <Form.Item name="api_key" label="API Key"
          rules={[{ required: !configured, message: '请输入 API Key' }]}
          extra={configured ? '留空则保持原有 API Key 不变' : undefined}>
          <Input.Password placeholder={configured ? '••••••••（已配置，留空不修改）' : '请输入 API Key'} />
        </Form.Item>
        <Form.Item name="model" label="模型名称">
          <Input placeholder="如：gpt-4o-mini（可选）" />
        </Form.Item>
        <Form.Item name="connector_type" label="连接器类型" initialValue="openai">
          <Select>
            <Select.Option value="openai">openai</Select.Option>
            <Select.Option value="dify">dify</Select.Option>
            <Select.Option value="custom">custom</Select.Option>
          </Select>
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" onClick={handleSave} loading={saving}>
              保存配置
            </Button>
            <Button icon={<ApiOutlined />} onClick={handleTest} loading={testing}>
              测试连接
            </Button>
          </Space>
        </Form.Item>
      </Form>
    </Card>
  )
}
