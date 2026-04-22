import { useEffect, useState } from 'react'
import { Avatar, Button, InputNumber, Spin, message } from 'antd'
import { ApiOutlined, BugOutlined, CheckCircleOutlined, DownloadOutlined, ExportOutlined, LoadingOutlined, RobotOutlined, SafetyOutlined, ThunderboltOutlined, UserOutlined } from '@ant-design/icons'

import type { ChatMessage, PlanInfo, WelcomeCapability } from '../../services/chat'
import { reportService } from '../../services/report'
import { LABELS, RESOURCE_MODE_LABELS, normalizePlanForDisplay, safetyScoreFromRiskScore } from './chatDisplay'

const RISK_COLORS: Record<string, string> = {
  critical: '#cf1322',
  high: '#ff4d4f',
  medium: '#fa8c16',
  low: '#52c41a',
  info: '#69db7c',
}

const RISK_LABELS: Record<string, string> = {
  critical: '严重风险',
  high: '高风险',
  medium: '中风险',
  low: '低风险',
  info: '未发现明显风险',
}

function capabilityVisual(item: WelcomeCapability) {
  switch (item.tone) {
    case 'governance':
      return { icon: <SafetyOutlined />, color: '#69db7c' }
    case 'engine':
      return { icon: <ThunderboltOutlined />, color: '#ffd43b' }
    case 'tool':
      return { icon: <ApiOutlined />, color: '#4d96ff' }
    default:
      return { icon: <BugOutlined />, color: '#ff6b6b' }
  }
}

function PlanConfirmCard({ plan, onConfirm, confirmed }: { plan: PlanInfo; onConfirm: (count: number) => void; confirmed: boolean }) {
  const normalizedPlan = normalizePlanForDisplay(plan)
  const [testCount, setTestCount] = useState(normalizedPlan.test_count ?? 20)

  useEffect(() => {
    setTestCount(normalizedPlan.test_count ?? 20)
  }, [plan, normalizedPlan.test_count])

  const rows: [string, string][] = [
    ['任务名称', normalizedPlan.name],
    ['评估目标', normalizedPlan.goal],
    ['目标类型', normalizedPlan.target_type],
  ]
  if (normalizedPlan.target_url) rows.push(['目标地址', normalizedPlan.target_url])
  if (normalizedPlan.target_model) rows.push(['模型', normalizedPlan.target_model])
  rows.push(['资源模式', RESOURCE_MODE_LABELS[normalizedPlan.resource_mode_preference || ''] || '自动选择'])

  return (
    <div style={{ background: 'var(--bg-card)', border: '1px solid rgba(26,109,255,0.3)', borderRadius: 12, padding: '16px 20px', maxWidth: 500, marginTop: 8 }}>
      <div style={{ color: '#4d96ff', fontWeight: 600, marginBottom: 12, fontSize: 14 }}>📋 评估计划</div>
      <div style={{ display: 'grid', gap: 8, marginBottom: 16 }}>
        {rows.map(([label, value]) => (
          <div key={label} style={{ display: 'flex', gap: 8, fontSize: 13 }}>
            <span style={{ color: 'var(--text-muted)', minWidth: 70 }}>{label}</span>
            <span style={{ color: 'var(--text-primary)', wordBreak: 'break-all' }}>{value}</span>
          </div>
        ))}
        <div style={{ display: 'flex', gap: 8, fontSize: 13, flexWrap: 'wrap' }}>
          <span style={{ color: 'var(--text-muted)', minWidth: 70 }}>评估类型</span>
          <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
            {(normalizedPlan.assessment_types ?? []).map(type => (
              <span key={type} style={{ background: 'rgba(26,109,255,0.15)', color: '#4d96ff', padding: '2px 8px', borderRadius: 4, fontSize: 12 }}>
                {LABELS[type] || type}
              </span>
            ))}
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8, alignItems: 'center', fontSize: 13 }}>
          <span style={{ color: 'var(--text-muted)', minWidth: 70 }}>测试问题数</span>
          <InputNumber min={1} max={2000} value={testCount} onChange={(value) => setTestCount(typeof value === 'number' ? value : 20)} disabled={confirmed} size="small" />
          <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>表示本次最多执行多少条最终测试问题</span>
        </div>
      </div>
      {confirmed ? (
        <div style={{ color: '#69db7c', fontSize: 13, display: 'flex', alignItems: 'center', gap: 6 }}>
          <CheckCircleOutlined /> 已确认，正在按 {testCount} 条测试问题启动评估...
        </div>
      ) : (
        <Button type="primary" size="small" onClick={() => onConfirm(testCount)}>
          确认执行
        </Button>
      )}
    </div>
  )
}

function ReportCard({
  cardType,
  assessmentId,
  riskLevel,
  riskScore,
  summary,
  isRunning,
  statusText,
  phase,
  executedCount,
  plannedCount,
}: {
  cardType?: string
  assessmentId?: string
  riskLevel?: string
  riskScore?: number
  summary?: string
  isRunning?: boolean
  statusText?: string
  phase?: string
  executedCount?: number
  plannedCount?: number
}) {
  const [downloading, setDownloading] = useState(false)
  const safetyScore = safetyScoreFromRiskScore(riskScore)

  const handleDownload = async () => {
    if (!assessmentId) return
    setDownloading(true)
    try {
      const blob = await reportService.downloadPDF(assessmentId)
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `report-${assessmentId}.pdf`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
    } catch {
      message.error('PDF 下载失败，请稍后重试')
    }
    setDownloading(false)
  }

  if (cardType === 'report' && assessmentId) {
    const normalizedRiskLevel = (riskLevel || '').toLowerCase()
    const color = RISK_COLORS[normalizedRiskLevel] || '#4d96ff'
    const label = RISK_LABELS[normalizedRiskLevel] || riskLevel || '未知'
    return (
      <div style={{ background: 'var(--bg-card)', border: `1px solid ${color}40`, borderRadius: 12, padding: '16px 20px', maxWidth: 420, marginTop: 8 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
          <CheckCircleOutlined style={{ color: '#69db7c', fontSize: 18 }} />
          <span style={{ color: 'var(--text-primary)', fontSize: 14, fontWeight: 600 }}>评估已完成</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
          <span style={{ background: color, color: '#fff', padding: '2px 10px', borderRadius: 4, fontSize: 12, fontWeight: 500 }}>{label}</span>
          <span style={{ color: 'var(--text-primary)', fontSize: 20, fontWeight: 700 }}>
            {safetyScore}
            <span style={{ fontSize: 12, color: 'var(--text-muted)', marginLeft: 2 }}>安全分</span>
          </span>
        </div>
        {summary && <div style={{ color: 'var(--text-secondary)', fontSize: 13, lineHeight: 1.6, marginBottom: 12 }}>{summary}</div>}
        <Button type="primary" icon={<DownloadOutlined />} size="small" loading={downloading} onClick={handleDownload}>
          下载 PDF 报告
        </Button>
      </div>
    )
  }

  if (cardType === 'progress') {
    const phaseLabel = phase === 'reporting'
      ? '正在生成报告'
      : phase === 'executed'
        ? '测试已完成'
        : phase === 'starting'
          ? '初始化评估'
          : '正在执行测试'
    const showSpinner = isRunning && phase !== 'executed'
    return (
      <div style={{ background: 'var(--bg-card)', border: '1px solid rgba(105,219,124,0.3)', borderRadius: 12, padding: '14px 18px', maxWidth: 420, marginTop: 8 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 10 }}>
          {showSpinner ? <Spin indicator={<LoadingOutlined style={{ color: '#4d96ff' }} />} /> : <CheckCircleOutlined style={{ color: '#69db7c', fontSize: 18 }} />}
          <div>
            <div style={{ color: 'var(--text-primary)', fontSize: 14, fontWeight: 600 }}>{phaseLabel}</div>
            <div style={{ color: 'var(--text-secondary)', fontSize: 13 }}>{statusText || '评估任务执行中，请稍候...'}</div>
          </div>
        </div>
        {typeof plannedCount === 'number' && plannedCount > 0 && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, color: 'var(--text-muted)', fontSize: 12 }}>
            <span style={{ background: 'rgba(26,109,255,0.12)', color: '#4d96ff', padding: '2px 8px', borderRadius: 4 }}>
              已执行 {typeof executedCount === 'number' ? executedCount : 0}/{plannedCount} 条
            </span>
            {phase === 'reporting' && <span>测试已完成，报告生成后会自动提供下载卡片。</span>}
          </div>
        )}
      </div>
    )
  }

  return null
}

function SkillLaunchCard({ skillName, openUrl }: { skillName?: string; openUrl?: string }) {
  if (!openUrl) return null
  return (
    <div style={{ background: 'var(--bg-card)', border: '1px solid rgba(26,109,255,0.3)', borderRadius: 12, padding: '16px 20px', maxWidth: 500, marginTop: 8 }}>
      <div style={{ color: '#4d96ff', fontWeight: 600, marginBottom: 8, fontSize: 14 }}>互动 Skill</div>
      <div style={{ color: 'var(--text-secondary)', fontSize: 13, lineHeight: 1.7, marginBottom: 14 }}>
        {skillName ? `已匹配到「${skillName}」页面工具。` : '已匹配到一个可直接打开的页面工具。'}
        点击下面按钮会在新标签页打开，不会启动评测流程。
      </div>
      <Button type="primary" icon={<ExportOutlined />} size="small" onClick={() => window.open(openUrl, '_blank', 'noopener,noreferrer')}>
        新标签页打开
      </Button>
    </div>
  )
}

export function WelcomePanel({ items, onQuickPrompt }: { items: WelcomeCapability[]; onQuickPrompt: (text: string) => void }) {
  return (
    <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 32, minHeight: '100%' }}>
      <div style={{ textAlign: 'center' }}>
        <div style={{ width: 64, height: 64, borderRadius: 16, background: 'rgba(26,109,255,0.15)', border: '1px solid rgba(26,109,255,0.4)', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 16px' }}>
          <RobotOutlined style={{ fontSize: 32, color: '#4d96ff' }} />
        </div>
        <div style={{ color: 'var(--text-primary)', fontSize: 22, fontWeight: 600 }}>AI 安全评估助手</div>
        <div style={{ color: 'var(--text-muted)', fontSize: 14, marginTop: 8 }}>描述你的评估需求，我来帮你制定评估方案</div>
      </div>
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', justifyContent: 'center', maxWidth: 560 }}>
        {items.map(item => {
          const visual = capabilityVisual(item)
          return (
            <div
              key={item.id}
              title={item.description || item.label}
              onClick={() => onQuickPrompt(item.prompt)}
              style={{ padding: '12px 18px', background: 'var(--bg-card)', border: '1px solid var(--border-color)', borderRadius: 10, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: 'var(--text-primary)', transition: 'border-color 0.2s' }}
              onMouseEnter={(event) => { event.currentTarget.style.borderColor = visual.color }}
              onMouseLeave={(event) => { event.currentTarget.style.borderColor = 'var(--border-color)' }}
            >
              <span style={{ color: visual.color }}>{visual.icon}</span>
              {item.label}
            </div>
          )
        })}
      </div>
    </div>
  )
}

export function TypingIndicator() {
  return (
    <div style={{ display: 'flex', gap: 10, marginBottom: 20 }}>
      <Avatar size={34} icon={<RobotOutlined />} style={{ background: 'rgba(105,219,124,0.2)', border: '1px solid rgba(105,219,124,0.4)', flexShrink: 0 }} />
      <div style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)', borderRadius: '4px 12px 12px 12px', padding: '12px 16px', display: 'flex', gap: 4, alignItems: 'center' }}>
        {[0, 1, 2].map(index => (
          <div key={index} style={{ width: 6, height: 6, borderRadius: '50%', background: '#4d96ff', animation: `bounce 1.2s ${index * 0.2}s infinite` }} />
        ))}
      </div>
    </div>
  )
}

export function MessageBubble({
  msg,
  onPlanConfirm,
  confirmedMsgs,
  isRunning,
}: {
  msg: ChatMessage
  onPlanConfirm: (count: number) => void
  confirmedMsgs: Set<string>
  isRunning: boolean
}) {
  const isUser = msg.role === 'user'
  const cardType = msg.metadata?.card_type
  const hideTextBubble = !isUser && (cardType === 'progress' || cardType === 'report' || cardType === 'skill_launch')

  return (
    <div style={{ display: 'flex', flexDirection: isUser ? 'row-reverse' : 'row', gap: 10, marginBottom: 20, alignItems: 'flex-start' }}>
      <Avatar size={34} icon={isUser ? <UserOutlined /> : <RobotOutlined />} style={{ background: isUser ? 'rgba(26,109,255,0.2)' : 'rgba(105,219,124,0.2)', border: `1px solid ${isUser ? 'rgba(26,109,255,0.4)' : 'rgba(105,219,124,0.4)'}`, flexShrink: 0 }} />
      <div style={{ maxWidth: '72%' }}>
        {msg.content && !hideTextBubble && (
          <div style={{ background: isUser ? 'rgba(26,109,255,0.15)' : 'var(--bg-card)', border: `1px solid ${isUser ? 'rgba(26,109,255,0.3)' : 'var(--border-color)'}`, borderRadius: isUser ? '12px 4px 12px 12px' : '4px 12px 12px 12px', padding: '10px 14px', color: 'var(--text-primary)', fontSize: 14, lineHeight: 1.6, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
            {msg.content}
          </div>
        )}
        {cardType === 'plan_confirm' && !isUser && msg.metadata?.plan && (
          <PlanConfirmCard plan={msg.metadata.plan as PlanInfo} onConfirm={onPlanConfirm} confirmed={confirmedMsgs.has(msg.id)} />
        )}
        {(cardType === 'progress' || cardType === 'report') && !isUser && (
          <ReportCard
            cardType={cardType}
            assessmentId={msg.metadata?.assessment_id as string | undefined}
            riskLevel={msg.metadata?.risk_level as string | undefined}
            riskScore={msg.metadata?.risk_score as number | undefined}
            summary={msg.metadata?.summary as string | undefined}
            statusText={msg.metadata?.status_text as string | undefined}
            phase={msg.metadata?.phase as string | undefined}
            executedCount={msg.metadata?.executed_count as number | undefined}
            plannedCount={msg.metadata?.planned_count as number | undefined}
            isRunning={isRunning}
          />
        )}
        {cardType === 'skill_launch' && !isUser && (
          <SkillLaunchCard skillName={msg.metadata?.skill_name as string | undefined} openUrl={msg.metadata?.open_url as string | undefined} />
        )}
        <div style={{ color: 'var(--text-muted)', fontSize: 11, marginTop: 4, textAlign: isUser ? 'right' : 'left' }}>
          {new Date(msg.created_at).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}
        </div>
      </div>
    </div>
  )
}
