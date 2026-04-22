import { useEffect, useState } from 'react'
import { Button, Result, Spin, Typography } from 'antd'
import { ArrowLeftOutlined } from '@ant-design/icons'
import { chatService, type SkillLaunchDocument } from '../../services/chat'

const { Text, Title } = Typography

export function EnterpriseSkillOpen({ skillId }: { skillId: string }) {
  const [loading, setLoading] = useState(true)
  const [item, setItem] = useState<SkillLaunchDocument | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false

    const load = async () => {
      setLoading(true)
      setError('')
      try {
        const doc = await chatService.getSkillLaunchDocument(skillId)
        if (cancelled) return
        setItem(doc)
        document.title = doc.document_title || doc.skill_name || 'Interactive Skill'
      } catch (err) {
        if (cancelled) return
        setError((err as Error).message || '加载 Skill 页面失败')
      } finally {
        if (!cancelled) {
          setLoading(false)
        }
      }
    }

    void load()
    return () => {
      cancelled = true
    }
  }, [skillId])

  if (loading) {
    return (
      <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'var(--bg-base)' }}>
        <Spin size="large" />
      </div>
    )
  }

  if (error || !item) {
    return (
      <div style={{ minHeight: '100vh', background: 'var(--bg-base)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24 }}>
        <Result
          status="error"
          title="Skill 页面加载失败"
          subTitle={error || '未获取到 Skill 页面内容'}
          extra={
            <Button type="primary" onClick={() => window.location.assign('/enterprise')}>
              返回企业助手
            </Button>
          }
        />
      </div>
    )
  }

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg-base)', display: 'flex', flexDirection: 'column' }}>
      <div style={{ height: 64, padding: '0 20px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: '1px solid var(--border-color)', background: 'var(--bg-surface)' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Button icon={<ArrowLeftOutlined />} onClick={() => window.location.assign('/enterprise')}>
            返回企业助手
          </Button>
          <div>
            <Title level={5} style={{ color: 'var(--text-primary)', margin: 0 }}>
              {item.document_title || item.skill_name}
            </Title>
            <Text style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
              {item.skill_name} · 版本 {item.version}
            </Text>
          </div>
        </div>
        <Text style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
          该页面仅打开互动 Skill，不会启动评测流程
        </Text>
      </div>

      <div style={{ flex: 1, padding: 16 }}>
        <iframe
          title={item.document_title || item.skill_name}
          srcDoc={item.html}
          sandbox="allow-scripts allow-forms allow-downloads"
          style={{ width: '100%', height: 'calc(100vh - 96px)', border: '1px solid var(--border-color)', borderRadius: 12, background: '#fff' }}
        />
      </div>
    </div>
  )
}
