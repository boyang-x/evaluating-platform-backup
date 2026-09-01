import { useEffect, useState } from 'react'
import { Card, Descriptions, Tag, Typography } from 'antd'
import { adminService, type AdminOverview } from '../../services/admin'

const { Text, Title } = Typography

export function SystemHealth() {
  const [overview, setOverview] = useState<AdminOverview | null>(null)

  useEffect(() => {
    adminService.getOverview().then(setOverview).catch(() => {})
  }, [])

  return (
    <div>
      <div style={{ marginBottom: 16 }}>
        <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>系统健康</Title>
        <Text style={{ color: 'var(--text-secondary)', fontSize: 13 }}>查看当前 BFF 与 maclaw 主链路配置状态。</Text>
      </div>
      <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)', borderRadius: 8 }}>
        <Descriptions column={1}>
          <Descriptions.Item label="maclaw runtime">
            <Tag color={overview?.maclaw_runtime_configured ? 'green' : 'red'}>
              {overview?.maclaw_runtime_configured ? 'configured' : 'not configured'}
            </Tag>
          </Descriptions.Item>
          <Descriptions.Item label="maclaw 映射账号">{overview?.maclaw_mapped_accounts ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="发布资源">{overview?.published_resources ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="shadow 资源">{overview?.shadow_resources ?? '-'}</Descriptions.Item>
          <Descriptions.Item label="说明">
            管理员页面只展示安全摘要；敏感 token、credential、payload 和 evidence content 不进入浏览器响应。
          </Descriptions.Item>
        </Descriptions>
      </Card>
    </div>
  )
}
