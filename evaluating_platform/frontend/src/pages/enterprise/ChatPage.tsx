import { useState, useEffect, useRef, useCallback } from 'react'
import { Button, Input, Avatar, Spin, InputNumber, message } from 'antd'
import { PlusOutlined, SendOutlined, DeleteOutlined, RobotOutlined, UserOutlined, SafetyOutlined, BugOutlined, ThunderboltOutlined, ApiOutlined, CheckCircleOutlined, LoadingOutlined, DownloadOutlined } from '@ant-design/icons'
import { chatService } from '../../services/chat'
import type { ChatSession, ChatMessage, PlanInfo } from '../../services/chat'
import { reportService } from '../../services/report'

const { TextArea } = Input
const QUICK_PROMPTS = [
  { icon: <BugOutlined />, text: '测试提示词注入攻击', color: '#ff6b6b' },
  { icon: <SafetyOutlined />, text: '评估 Agent 安全性', color: '#4d96ff' },
  { icon: <ThunderboltOutlined />, text: '检测越狱攻击风险', color: '#ffd43b' },
  { icon: <ApiOutlined />, text: '内容合规性检查', color: '#69db7c' },
]
const LABELS: Record<string, string> = { prompt_injection: '提示词注入', jailbreak: '越狱攻击', goal_hijacking: '目标劫持', tool_poisoning: '工具投毒', compliance_check: '合规检查' }
const RESOURCE_MODE_LABELS: Record<string, string> = { sample_template: '样本+模板组合', composed_attack: '已组合攻击' }
function PlanConfirmCard({ plan, onConfirm, confirmed }: { plan: PlanInfo; onConfirm: (count: number) => void; confirmed: boolean }) {
  const [testCount, setTestCount] = useState(plan.test_count ?? 20)

  useEffect(() => {
    setTestCount(plan.test_count ?? 20)
  }, [plan])

  const rows: [string, string][] = [['任务名称', plan.name], ['评估目标', plan.goal], ['目标类型', plan.target_type], ['目标地址', plan.target_url]]
  if (plan.target_model) rows.push(['模型', plan.target_model])
  rows.push(['资源模式', RESOURCE_MODE_LABELS[plan.resource_mode_preference || ''] || '自动选择'])
  return (
    <div style={{ background: 'var(--bg-card)', border: '1px solid rgba(26,109,255,0.3)', borderRadius: 12, padding: '16px 20px', maxWidth: 500, marginTop: 8 }}>
      <div style={{ color: '#4d96ff', fontWeight: 600, marginBottom: 12, fontSize: 14 }}>📋 评估计划</div>
      <div style={{ display: 'grid', gap: 8, marginBottom: 16 }}>
        {rows.map(([k, v]) => (
          <div key={k} style={{ display: 'flex', gap: 8, fontSize: 13 }}>
            <span style={{ color: 'var(--text-muted)', minWidth: 70 }}>{k}</span>
            <span style={{ color: 'var(--text-primary)', wordBreak: 'break-all' }}>{v}</span>
          </div>
        ))}
        <div style={{ display: 'flex', gap: 8, fontSize: 13, flexWrap: 'wrap' }}>
          <span style={{ color: 'var(--text-muted)', minWidth: 70 }}>评估类型</span>
          <div style={{ display: 'flex', gap: 4, flexWrap: 'wrap' }}>
            {(plan.assessment_types ?? []).map(t => (
              <span key={t} style={{ background: 'rgba(26,109,255,0.15)', color: '#4d96ff', padding: '2px 8px', borderRadius: 4, fontSize: 12 }}>
                {LABELS[t] || t}
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
const RISK_COLORS: Record<string, string> = { high: '#ff4d4f', medium: '#fa8c16', low: '#52c41a' }
const RISK_LABELS: Record<string, string> = { high: '高风险', medium: '中风险', low: '低风险' }

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

  const handleDownload = async () => {
    if (!assessmentId) return
    setDownloading(true)
    try {
      const blob = await reportService.downloadPDF(assessmentId)
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `report-${assessmentId}.pdf`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)
    } catch {
      message.error('PDF 下载失败，请稍后重试')
    }
    setDownloading(false)
  }

  // Completed report state
  if (cardType === 'report' && assessmentId) {
    const color = RISK_COLORS[riskLevel || ''] || '#4d96ff'
    const label = RISK_LABELS[riskLevel || ''] || riskLevel || '未知'
    return (
      <div style={{ background: 'var(--bg-card)', border: `1px solid ${color}40`, borderRadius: 12, padding: '16px 20px', maxWidth: 420, marginTop: 8 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
          <CheckCircleOutlined style={{ color: '#69db7c', fontSize: 18 }} />
          <span style={{ color: 'var(--text-primary)', fontSize: 14, fontWeight: 600 }}>评估已完成</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 10 }}>
          <span style={{ background: color, color: '#fff', padding: '2px 10px', borderRadius: 4, fontSize: 12, fontWeight: 500 }}>{label}</span>
          {riskScore != null && <span style={{ color: 'var(--text-primary)', fontSize: 20, fontWeight: 700 }}>{riskScore}<span style={{ fontSize: 12, color: 'var(--text-muted)', marginLeft: 2 }}>分</span></span>}
        </div>
        {summary && <div style={{ color: 'var(--text-secondary)', fontSize: 13, lineHeight: 1.6, marginBottom: 12 }}>{summary}</div>}
        <Button type="primary" icon={<DownloadOutlined />} size="small" loading={downloading} onClick={handleDownload}>下载 PDF 报告</Button>
      </div>
    )
  }

  if (cardType === 'progress') {
    const phaseLabel = phase === 'reporting'
      ? '正在生成报告'
      : phase === 'executed'
        ? '测试已完成'
        : phase === 'starting'
          ? '正在启动任务'
          : '正在执行评估'
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
        {(typeof plannedCount === 'number' && plannedCount > 0) && (
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
function MessageBubble({ msg, onPlanConfirm, confirmedMsgs, isRunning }: { msg: ChatMessage; onPlanConfirm: (count: number) => void; confirmedMsgs: Set<string>; isRunning: boolean }) {
  const isUser = msg.role === 'user'
  const cardType = msg.metadata?.card_type
  return (<div style={{ display: 'flex', flexDirection: isUser ? 'row-reverse' : 'row', gap: 10, marginBottom: 20, alignItems: 'flex-start' }}><Avatar size={34} icon={isUser ? <UserOutlined /> : <RobotOutlined />} style={{ background: isUser ? 'rgba(26,109,255,0.2)' : 'rgba(105,219,124,0.2)', border: '1px solid ' + (isUser ? 'rgba(26,109,255,0.4)' : 'rgba(105,219,124,0.4)'), flexShrink: 0 }} /><div style={{ maxWidth: '72%' }}>{msg.content && (<div style={{ background: isUser ? 'rgba(26,109,255,0.15)' : 'var(--bg-card)', border: '1px solid ' + (isUser ? 'rgba(26,109,255,0.3)' : 'var(--border-color)'), borderRadius: isUser ? '12px 4px 12px 12px' : '4px 12px 12px 12px', padding: '10px 14px', color: 'var(--text-primary)', fontSize: 14, lineHeight: 1.6, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>{msg.content}</div>)}{cardType === 'plan_confirm' && !isUser && msg.metadata?.plan && (<PlanConfirmCard plan={msg.metadata.plan as unknown as PlanInfo} onConfirm={onPlanConfirm} confirmed={confirmedMsgs.has(msg.id)} />)}{(cardType === 'progress' || cardType === 'report') && !isUser && (<ReportCard cardType={cardType} assessmentId={msg.metadata?.assessment_id as string | undefined} riskLevel={msg.metadata?.risk_level as string | undefined} riskScore={msg.metadata?.risk_score as number | undefined} summary={msg.metadata?.summary as string | undefined} statusText={msg.metadata?.status_text as string | undefined} phase={msg.metadata?.phase as string | undefined} executedCount={msg.metadata?.executed_count as number | undefined} plannedCount={msg.metadata?.planned_count as number | undefined} isRunning={isRunning} />)}<div style={{ color: 'var(--text-muted)', fontSize: 11, marginTop: 4, textAlign: isUser ? 'right' : 'left' }}>{new Date(msg.created_at).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}</div></div></div>)
}
async function doSend(sid: string, text: string, setMessages: React.Dispatch<React.SetStateAction<ChatMessage[]>>, setSending: (v: boolean) => void, loadSessions: () => void, append: boolean) {
  setSending(true)
  const tempId = 'tmp-' + Date.now()
  const tempUser: ChatMessage = { id: tempId, session_id: sid, role: 'user', content: text, metadata: { card_type: 'text' }, created_at: new Date().toISOString() }
  setMessages(prev => append ? [...prev, tempUser] : [tempUser])
  try { const aiMsg = await chatService.sendMessage(sid, text); setMessages(prev => append ? [...prev.filter(m => m.id !== tempId), tempUser, aiMsg] : [tempUser, aiMsg]); loadSessions() } catch { setMessages(prev => prev.filter(m => m.id !== tempId)) }
  setSending(false)
}
export function ChatPage() {
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const [loadingSession, setLoadingSession] = useState(false)
  const [confirmedMsgs, setConfirmedMsgs] = useState<Set<string>>(new Set())
  const [isRunning, setIsRunning] = useState(false)
  const bottomRef = useRef<HTMLDivElement>(null)
  const loadSessions = useCallback(async () => { try { const res = await chatService.listSessions(); setSessions(res.items || []) } catch { /* ignore */ } }, [])
  useEffect(() => { loadSessions() }, [loadSessions])
  useEffect(() => { bottomRef.current?.scrollIntoView({ behavior: 'smooth' }) }, [messages])
  useEffect(() => {
    if (!activeId || !isRunning) return
    const timer = setInterval(async () => {
      try {
        const res = await chatService.getSession(activeId)
        setMessages(res.messages || [])
        if (res.session.state === 'completed' || res.session.state === 'failed') {
          setIsRunning(false)
        }
      } catch { /* silent retry on next interval */ }
    }, 3000)
    return () => clearInterval(timer)
  }, [activeId, isRunning])
  const openSession = async (id: string) => { setActiveId(id); setLoadingSession(true); try { const res = await chatService.getSession(id); setMessages(res.messages || []); setIsRunning(res.session.state === 'running') } catch { /* ignore */ } setLoadingSession(false) }
  const newSession = async () => { const s = await chatService.createSession(); setSessions(prev => [s, ...prev]); setActiveId(s.id); setMessages([]) }
  const deleteSession = async (id: string, e: React.MouseEvent) => { e.stopPropagation(); await chatService.deleteSession(id); setSessions(prev => prev.filter(s => s.id !== id)); if (activeId === id) { setActiveId(null); setMessages([]) } }
  const handleSend = async () => { if (!input.trim() || !activeId || sending) return; const text = input; setInput(''); await doSend(activeId, text, setMessages, setSending, loadSessions, true) }
  const startWithPrompt = async (text: string) => { if (sending) return; const s = await chatService.createSession(); setSessions(prev => [s, ...prev]); setActiveId(s.id); await doSend(s.id, text, setMessages, setSending, loadSessions, false) }
  const handleConfirmPlan = async (msgId: string, testCount: number) => {
    if (!activeId) return
    try {
      await chatService.confirmPlan(activeId, testCount)
      setConfirmedMsgs(prev => new Set([...prev, msgId]))
      setIsRunning(true)
      const updated = await chatService.getSession(activeId)
      setMessages(updated.messages || [])
    } catch {
      message.error('启动评估失败，请稍后重试')
    }
  }
  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 64px)', overflow: 'hidden' }}>
      <div style={{ width: 260, borderRight: '1px solid var(--border-color)', display: 'flex', flexDirection: 'column', background: 'var(--bg-surface)', flexShrink: 0 }}>
        <div style={{ padding: '12px 12px 8px' }}><Button type="primary" icon={<PlusOutlined />} block onClick={newSession} style={{ borderRadius: 8 }}>新建对话</Button></div>
        <div style={{ flex: 1, overflowY: 'auto', padding: '0 8px' }}>
          {sessions.length === 0 && <div style={{ color: 'var(--text-muted)', fontSize: 12, textAlign: 'center', marginTop: 32 }}>暂无对话记录</div>}
          {sessions.map(s => (<div key={s.id} onClick={() => openSession(s.id)} style={{ padding: '10px 12px', borderRadius: 8, cursor: 'pointer', marginBottom: 2, background: activeId === s.id ? 'rgba(26,109,255,0.12)' : 'transparent', border: '1px solid ' + (activeId === s.id ? 'rgba(26,109,255,0.3)' : 'transparent'), display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8 }}><div style={{ flex: 1, overflow: 'hidden' }}><div style={{ color: 'var(--text-primary)', fontSize: 13, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{s.title}</div><div style={{ color: 'var(--text-muted)', fontSize: 11, marginTop: 2 }}>{new Date(s.updated_at).toLocaleDateString('zh-CN')}</div></div><DeleteOutlined style={{ color: 'var(--text-muted)', fontSize: 13, flexShrink: 0 }} onClick={(e) => deleteSession(s.id, e)} /></div>))}
        </div>
      </div>
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        {!activeId ? (
          <div style={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: 32 }}>
            <div style={{ textAlign: 'center' }}><div style={{ width: 64, height: 64, borderRadius: 16, background: 'rgba(26,109,255,0.15)', border: '1px solid rgba(26,109,255,0.4)', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 16px' }}><RobotOutlined style={{ fontSize: 32, color: '#4d96ff' }} /></div><div style={{ color: 'var(--text-primary)', fontSize: 22, fontWeight: 600 }}>AI 安全评估助手</div><div style={{ color: 'var(--text-muted)', fontSize: 14, marginTop: 8 }}>描述你的评估需求，我来帮你制定评估方案</div></div>
            <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', justifyContent: 'center', maxWidth: 560 }}>{QUICK_PROMPTS.map(p => (<div key={p.text} onClick={() => startWithPrompt(p.text)} style={{ padding: '12px 18px', background: 'var(--bg-card)', border: '1px solid var(--border-color)', borderRadius: 10, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8, fontSize: 13, color: 'var(--text-primary)', transition: 'border-color 0.2s' }} onMouseEnter={e => (e.currentTarget.style.borderColor = p.color)} onMouseLeave={e => (e.currentTarget.style.borderColor = 'var(--border-color)')}><span style={{ color: p.color }}>{p.icon}</span>{p.text}</div>))}</div>
          </div>
        ) : (
          <div style={{ flex: 1, overflowY: 'auto', padding: '24px 32px' }}>
            {loadingSession ? <div style={{ textAlign: 'center', paddingTop: 60 }}><LoadingOutlined style={{ fontSize: 24, color: '#4d96ff' }} /></div> : (<>{messages.map(msg => (<MessageBubble key={msg.id} msg={msg} onPlanConfirm={(rounds) => handleConfirmPlan(msg.id, rounds)} confirmedMsgs={confirmedMsgs} isRunning={isRunning} />))}{sending && (<div style={{ display: 'flex', gap: 10, marginBottom: 20 }}><Avatar size={34} icon={<RobotOutlined />} style={{ background: 'rgba(105,219,124,0.2)', border: '1px solid rgba(105,219,124,0.4)', flexShrink: 0 }} /><div style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)', borderRadius: '4px 12px 12px 12px', padding: '12px 16px', display: 'flex', gap: 4, alignItems: 'center' }}>{[0,1,2].map(i => (<div key={i} style={{ width: 6, height: 6, borderRadius: '50%', background: '#4d96ff', animation: 'bounce 1.2s ' + (i*0.2) + 's infinite' }} />))}</div></div>)}<div ref={bottomRef} /></>)}
          </div>
        )}
        <div style={{ padding: '12px 24px 20px', borderTop: '1px solid var(--border-color)', background: 'var(--bg-surface)' }}>
          <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end' }}>
            <TextArea value={input} onChange={e => setInput(e.target.value)} onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); if (!input.trim() || sending || isRunning) return; if (activeId) { handleSend() } else { const text = input; setInput(''); startWithPrompt(text) } } }} placeholder={isRunning ? '评估任务执行中，请稍候...' : '描述你的评估需求，按 Enter 发送，Shift+Enter 换行...'} autoSize={{ minRows: 1, maxRows: 5 }} disabled={isRunning} style={{ flex: 1, background: 'var(--bg-card)', borderColor: 'var(--border-color)', color: 'var(--text-primary)', borderRadius: 10, resize: 'none' }} />
            <Button type="primary" icon={<SendOutlined />} onClick={() => { if (!input.trim() || sending || isRunning) return; if (activeId) { handleSend() } else { const text = input; setInput(''); startWithPrompt(text) } }} disabled={!input.trim() || sending || isRunning} style={{ height: 40, borderRadius: 10 }} />
          </div>
        </div>
      </div>
      <style>{'@keyframes bounce{0%,80%,100%{transform:translateY(0);opacity:0.4}40%{transform:translateY(-6px);opacity:1}}'}</style>
    </div>
  )
}
