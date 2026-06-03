import { useCallback, useEffect, useRef, useState, type Dispatch, type MouseEvent, type SetStateAction } from 'react'
import { Button, Drawer, Form, Input, Select, Space, message } from 'antd'
import { ApiOutlined, DeleteOutlined, LoadingOutlined, PlusOutlined, SendOutlined } from '@ant-design/icons'

import type { ChatMessage, ChatSession, WelcomeCapability } from '../../services/chat'
import { maclawRuntimeChatService as chatService } from '../../services/maclawRuntime'
import type { EvaluationTarget, RuntimeRun } from '../../services/maclawRuntime'
import { MessageBubble, TypingIndicator, WelcomePanel } from './ChatCards'
import { DEFAULT_WELCOME_CAPABILITIES, confirmedPlanStateFromMessages, formatSessionDate, messagesForSession, normalizeSessionTimestamps, pendingAssistantUserMessageId, prepareMessagesForDisplay } from './chatDisplay'
import { applyEvaluationJobRecovery, buildEvaluationJobProgressMessage, jobRunToStream, markEvaluationJobRecoveryStarted, markEvaluationJobRetryStarted } from './jobProgressMessages'

const { TextArea } = Input

type SessionSnapshot = {
  session: ChatSession
  messages: ChatMessage[]
}

function buildTemporaryUserMessage(sessionId: string, content: string): ChatMessage {
  return {
    id: `tmp-${Date.now()}`,
    session_id: sessionId,
    role: 'user',
    content,
    metadata: { card_type: 'text' },
    created_at: new Date().toISOString(),
  }
}

function buildConfirmProgressMessage(sessionId: string, planMessageId: string, plannedCount: number): ChatMessage {
  return {
    id: `confirm-progress-${planMessageId}`,
    session_id: sessionId,
    role: 'assistant',
    content: '',
    metadata: {
      card_type: 'progress',
      phase: 'starting',
      status_text: '已确认执行，正在启动 MaClaw 评估任务...',
      planned_count: plannedCount,
      executed_count: 0,
    },
    created_at: new Date().toISOString(),
  }
}

function findRestorableProgressJobId(messages: ChatMessage[]): string | undefined {
  for (let i = messages.length - 1; i >= 0; i -= 1) {
    const metadata = messages[i].metadata || {}
    if (metadata.card_type !== 'progress') continue
    const phase = String(metadata.phase || '').toLowerCase()
    if (phase === 'failed' || phase === 'canceled' || phase === 'cancelled' || phase === 'completed' || phase === 'report') {
      continue
    }
    const jobId = typeof metadata.job_id === 'string' ? metadata.job_id.trim() : ''
    if (jobId) return jobId
  }
  return undefined
}

function mostRecentSession(sessions: ChatSession[]): ChatSession | undefined {
  return [...sessions].sort((left, right) => {
    const leftTime = Date.parse(left.updated_at || left.created_at || '')
    const rightTime = Date.parse(right.updated_at || right.created_at || '')
    return (Number.isFinite(rightTime) ? rightTime : 0) - (Number.isFinite(leftTime) ? leftTime : 0)
  })[0]
}

function extractSendErrorMessage(error: unknown) {
  if (error instanceof Error && error.message.trim()) {
    return error.message
  }
  if (typeof error === 'object' && error !== null) {
    const response = (error as { response?: { data?: { error?: unknown } } }).response
    const detail = response?.data?.error
    if (typeof detail === 'string' && detail.trim()) {
      return detail
    }
  }
  return '发送失败，请检查 maclaw 模型配置、被测模型连接或稍后重试'
}

async function sendPrompt(
  sessionId: string,
  text: string,
  append: boolean,
  setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
  setSendingSessionId: Dispatch<SetStateAction<string | null>>,
  onSessionUpdated: () => void,
  getActiveSessionId: () => string | null,
) {
  setSendingSessionId(sessionId)
  const tempUserMessage = buildTemporaryUserMessage(sessionId, text)
  if (getActiveSessionId() === sessionId) {
    setMessages(previous => (append ? [...messagesForSession(previous, sessionId), tempUserMessage] : [tempUserMessage]))
  }

  try {
    const result = await chatService.sendMessage(sessionId, text)
    const assistantMessage = result.message
    if (getActiveSessionId() === sessionId) {
      setMessages(previous => {
        const withoutTemp = messagesForSession(previous, sessionId).filter(message => message.id !== tempUserMessage.id)
        return append ? [...withoutTemp, tempUserMessage, assistantMessage] : [tempUserMessage, assistantMessage]
      })
    }
    onSessionUpdated()
  } catch (error) {
    if (getActiveSessionId() === sessionId) {
      setMessages(previous => messagesForSession(previous, sessionId).filter(message => message.id !== tempUserMessage.id))
      message.error(extractSendErrorMessage(error))
    }
  } finally {
    setSendingSessionId(current => (current === sessionId ? null : current))
  }
}

export function ChatPage() {
  const [targetForm] = Form.useForm()
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [welcomeCapabilities, setWelcomeCapabilities] = useState<WelcomeCapability[]>(DEFAULT_WELCOME_CAPABILITIES)
  const [input, setInput] = useState('')
  const [sendingSessionId, setSendingSessionId] = useState<string | null>(null)
  const [loadingSessionId, setLoadingSessionId] = useState<string | null>(null)
  const [confirmedMsgs, setConfirmedMsgs] = useState<Set<string>>(new Set())
  const [confirmedTestCounts, setConfirmedTestCounts] = useState<Record<string, number>>({})
  const [resumingJobIds, setResumingJobIds] = useState<Set<string>>(new Set())
  const [retryingJobIds, setRetryingJobIds] = useState<Set<string>>(new Set())
  const [runningSessionIds, setRunningSessionIds] = useState<Set<string>>(new Set())
  const [targetDrawerOpen, setTargetDrawerOpen] = useState(false)
  const [savingTarget, setSavingTarget] = useState(false)
  const [currentTarget, setCurrentTarget] = useState<EvaluationTarget | null>(null)

  const messageViewportRef = useRef<HTMLDivElement>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const shouldStickToBottomRef = useRef(true)
  const activeIdRef = useRef<string | null>(null)
  const streamStopRef = useRef<(() => void) | null>(null)

  const markSessionRunning = useCallback((sessionId: string, running: boolean) => {
    if (!sessionId) return
    setRunningSessionIds(previous => {
      const next = new Set(previous)
      if (running) {
        next.add(sessionId)
      } else {
        next.delete(sessionId)
      }
      return next
    })
  }, [])

  const updateScrollStickiness = useCallback(() => {
    const viewport = messageViewportRef.current
    if (!viewport) {
      shouldStickToBottomRef.current = true
      return
    }
    const distanceFromBottom = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight
    shouldStickToBottomRef.current = distanceFromBottom < 96
  }, [])

  const applySessionSnapshot = useCallback((snapshot: SessionSnapshot) => {
    const rawMessages = snapshot.messages || []
    const confirmedState = confirmedPlanStateFromMessages(rawMessages)
    setConfirmedMsgs(confirmedState.confirmedIds)
    setConfirmedTestCounts(confirmedState.testCounts)
    setMessages(prepareMessagesForDisplay(rawMessages, snapshot.session))
    markSessionRunning(snapshot.session.id, snapshot.session.state === 'running')
  }, [markSessionRunning])

  useEffect(() => {
    activeIdRef.current = activeId
  }, [activeId])

  const loadSessions = useCallback(async () => {
    try {
      const res = await chatService.listSessions()
      setSessions((res.items || []).map(normalizeSessionTimestamps))
    } catch {
      // Ignore sidebar refresh failures and preserve current UI.
    }
  }, [])

  const loadWelcomeCapabilities = useCallback(async () => {
    try {
      const items = (await chatService.getWelcomeCapabilities(6)).slice(0, 6)
      setWelcomeCapabilities(items.length > 0 ? items : DEFAULT_WELCOME_CAPABILITIES)
    } catch {
      setWelcomeCapabilities(DEFAULT_WELCOME_CAPABILITIES)
    }
  }, [])

  const refreshCurrentTarget = useCallback(async () => {
    try {
      const res = await chatService.listEvaluationTargets()
      setCurrentTarget((res.items || [])[0] || null)
    } catch {
      setCurrentTarget(null)
    }
  }, [])

  const startRunStream = useCallback((run: RuntimeRun, sessionId: string) => {
    if (!run.id) return
    streamStopRef.current?.()
    markSessionRunning(sessionId, true)
    streamStopRef.current = chatService.streamRunEvents(
      run.id,
      (eventMessage) => {
        if (activeIdRef.current !== sessionId) return
        setMessages(previous => prepareMessagesForDisplay([
          ...previous.filter(message => message.id !== eventMessage.id),
          eventMessage,
        ]))
      },
      async () => {
        streamStopRef.current = null
        markSessionRunning(sessionId, false)
        await loadSessions()
        if (activeIdRef.current !== sessionId) return
        try {
          const snapshot = await chatService.getSession(sessionId)
          applySessionSnapshot(snapshot)
        } catch {
          // Keep streamed cards visible if final snapshot refresh fails.
        }
      },
      () => {
        streamStopRef.current = null
        markSessionRunning(sessionId, false)
        message.error('maclaw 事件流连接失败，请稍后刷新会话。')
      },
    )
  }, [applySessionSnapshot, loadSessions, markSessionRunning])

  const waitForEvaluationJob = useCallback(async (jobId: string, sessionId: string) => {
    for (let attempt = 0; attempt < 120; attempt += 1) {
      if (activeIdRef.current !== sessionId) return
      const job = await chatService.getEvaluationJob(jobId)
      if (job.status === 'succeeded') {
        setMessages(previous => prepareMessagesForDisplay(
          previous.filter(message => message.id !== `job-${jobId}-queued`),
        ))
        markSessionRunning(sessionId, false)
        await loadSessions()
        if (activeIdRef.current !== sessionId) return
        try {
          const snapshot = await chatService.getSession(sessionId)
          applySessionSnapshot(snapshot)
        } catch {
          // Keep the progress card visible if final snapshot refresh fails.
        }
        return
      }
      if (job.status === 'failed' || job.status === 'canceled') {
        const phase = job.status === 'canceled' ? 'canceled' : 'failed'
        let cardJob = job
        if (!job.progress?.recovery_action) {
          try {
            const recovery = await chatService.getEvaluationJobRecovery(job.id)
            cardJob = applyEvaluationJobRecovery(job, recovery)
          } catch {
            cardJob = job
          }
        }
        const card = buildEvaluationJobProgressMessage(cardJob, sessionId, phase)
        setMessages(previous => prepareMessagesForDisplay([
          ...previous.filter(message => message.id !== card.id),
          card,
        ]))
        markSessionRunning(sessionId, false)
        if (cardJob.progress?.recovery_action === 'resume') {
          message.warning('评测任务中断，可在卡片中继续恢复。')
        } else if (cardJob.progress?.recovery_action === 'retry') {
          message.warning('评测任务中断，可在卡片中重试。')
        } else {
          message.error(cardJob.error || '评测任务未能完成，请稍后重试。')
        }
        return
      }
      const run = jobRunToStream(job, sessionId)
      if (run?.id) {
        setMessages(previous => prepareMessagesForDisplay(
          previous.filter(message => message.id !== `job-${jobId}-queued`),
        ))
        startRunStream(run, sessionId)
        return
      }
      await new Promise(resolve => window.setTimeout(resolve, 1000))
    }
    markSessionRunning(sessionId, false)
    message.error('评测任务排队超时，请稍后刷新会话。')
  }, [applySessionSnapshot, loadSessions, markSessionRunning, startRunStream])

  const handleRetryJob = useCallback(async (jobId: string, sessionId: string) => {
    if (activeIdRef.current !== sessionId) return
    setRetryingJobIds(previous => new Set([...previous, jobId]))
    markSessionRunning(sessionId, true)
    shouldStickToBottomRef.current = true
    try {
      const retryJob = await chatService.retryEvaluationJob(jobId)
      const queued = buildEvaluationJobProgressMessage(retryJob, sessionId, 'queued', { retryOfJobId: jobId })
      setMessages(previous => prepareMessagesForDisplay([
        ...markEvaluationJobRetryStarted(previous, jobId, retryJob.id).filter(message => message.id !== queued.id),
        queued,
      ]))
      void waitForEvaluationJob(retryJob.id, sessionId)
    } catch {
      markSessionRunning(sessionId, false)
      message.error('重试评测任务失败，请稍后再试。')
    } finally {
      setRetryingJobIds(previous => {
        const next = new Set(previous)
        next.delete(jobId)
        return next
      })
    }
  }, [markSessionRunning, waitForEvaluationJob])

  const handleResumeJob = useCallback(async (jobId: string, sessionId: string) => {
    if (activeIdRef.current !== sessionId) return
    setResumingJobIds(previous => new Set([...previous, jobId]))
    markSessionRunning(sessionId, true)
    shouldStickToBottomRef.current = true
    try {
      const resumeJob = await chatService.resumeEvaluationJob(jobId)
      const queued = buildEvaluationJobProgressMessage(resumeJob, sessionId, 'queued', { resumeOfJobId: jobId })
      setMessages(previous => prepareMessagesForDisplay([
        ...markEvaluationJobRecoveryStarted(previous, jobId, resumeJob.id, 'resume').filter(message => message.id !== queued.id),
        queued,
      ]))
      void waitForEvaluationJob(resumeJob.id, sessionId)
    } catch {
      markSessionRunning(sessionId, false)
      message.error('恢复评测任务失败，请稍后再试。')
    } finally {
      setResumingJobIds(previous => {
        const next = new Set(previous)
        next.delete(jobId)
        return next
      })
    }
  }, [markSessionRunning, waitForEvaluationJob])

  const waitForAssistantReply = useCallback(async (sessionId: string, userMessageId: string) => {
    if (!sessionId || !userMessageId) return
    setSendingSessionId(sessionId)
    try {
      for (let attempt = 0; attempt < 60; attempt += 1) {
        await new Promise(resolve => window.setTimeout(resolve, 1500))
        if (activeIdRef.current !== sessionId) return
        const snapshot = await chatService.getSession(sessionId)
        applySessionSnapshot(snapshot)
        if (pendingAssistantUserMessageId(snapshot.messages) !== userMessageId) {
          await loadSessions()
          return
        }
      }
      if (activeIdRef.current === sessionId) {
        message.warning('MaClaw 仍在生成回复，请稍后刷新会话。')
      }
    } catch {
      if (activeIdRef.current === sessionId) {
        message.error('刷新 MaClaw 回复状态失败，请稍后重试。')
      }
    } finally {
      setSendingSessionId(current => (current === sessionId ? null : current))
    }
  }, [applySessionSnapshot, loadSessions])

  const openSession = useCallback(async (id: string) => {
    streamStopRef.current?.()
    streamStopRef.current = null
    shouldStickToBottomRef.current = true
    activeIdRef.current = id
    setActiveId(id)
    setLoadingSessionId(id)
    try {
      const snapshot = await chatService.getSession(id)
      applySessionSnapshot(snapshot)
      const jobId = findRestorableProgressJobId(snapshot.messages || [])
      if (jobId) {
        markSessionRunning(id, true)
        void waitForEvaluationJob(jobId, id)
      } else {
        const pendingUserMessageId = pendingAssistantUserMessageId(snapshot.messages || [])
        if (pendingUserMessageId) {
          void waitForAssistantReply(id, pendingUserMessageId)
        }
      }
    } catch {
      // Ignore open failure and keep the previous content rendered.
    } finally {
      setLoadingSessionId(current => (current === id ? null : current))
    }
  }, [applySessionSnapshot, markSessionRunning, waitForAssistantReply, waitForEvaluationJob])

  const createSession = useCallback(async () => {
    streamStopRef.current?.()
    streamStopRef.current = null
    const session = normalizeSessionTimestamps(await chatService.createSession())
    setSessions(previous => [session, ...previous])
    activeIdRef.current = session.id
    setActiveId(session.id)
    setMessages([])
    markSessionRunning(session.id, false)
    setConfirmedMsgs(new Set())
    setConfirmedTestCounts({})
    shouldStickToBottomRef.current = true
    return session
  }, [markSessionRunning])

  const deleteSession = useCallback(async (id: string, event: MouseEvent) => {
    event.stopPropagation()
    await chatService.deleteSession(id)
    streamStopRef.current?.()
    streamStopRef.current = null
    setSessions(previous => previous.filter(session => session.id !== id))
    if (activeId === id) {
      activeIdRef.current = null
      setActiveId(null)
      setMessages([])
      setSendingSessionId(current => (current === id ? null : current))
      markSessionRunning(id, false)
      setConfirmedMsgs(new Set())
      setConfirmedTestCounts({})
    }
  }, [activeId, markSessionRunning])

  const sendToSession = useCallback(async (sessionId: string, text: string, append: boolean) => {
    await sendPrompt(sessionId, text, append, setMessages, setSendingSessionId, loadSessions, () => activeIdRef.current)
  }, [loadSessions])

  const submitPrompt = useCallback(async (text: string) => {
    const trimmed = text.trim()
    const activeSending = Boolean(activeId && sendingSessionId === activeId)
    const activeRunning = Boolean(activeId && runningSessionIds.has(activeId))
    if (!trimmed || activeSending || activeRunning) return

    if (activeId && messagesForSession(messages, activeId).length === 0) {
      await sendToSession(activeId, trimmed, false)
      return
    }

    const session = await createSession()
    await sendToSession(session.id, trimmed, false)
  }, [activeId, createSession, messages, runningSessionIds, sendToSession, sendingSessionId])

  const handleConfirmPlan = useCallback(async (msgId: string, testCount: number) => {
    if (!activeId) return
    const sessionId = activeId
    const progressMessage = buildConfirmProgressMessage(sessionId, msgId, testCount)
    try {
      shouldStickToBottomRef.current = true
      setConfirmedMsgs(previous => new Set([...previous, msgId]))
      setConfirmedTestCounts(previous => ({ ...previous, [msgId]: testCount }))
      markSessionRunning(sessionId, true)
      setMessages(previous => prepareMessagesForDisplay([
        ...messagesForSession(previous, sessionId).filter(message => message.id !== progressMessage.id),
        progressMessage,
      ]))
      const result = await chatService.confirmPlan(sessionId, testCount, msgId)
      if (activeIdRef.current !== sessionId) {
        if (result.job?.status === 'succeeded' || (!result.run && !result.job)) {
          markSessionRunning(sessionId, false)
        }
        return
      }
      if (result.job?.status === 'succeeded') {
        const snapshot = await chatService.getSession(sessionId)
        if (activeIdRef.current !== sessionId) return
        applySessionSnapshot(snapshot)
        await loadSessions()
        markSessionRunning(sessionId, false)
        return
      }
      setMessages(previous => prepareMessagesForDisplay([
        ...messagesForSession(previous, sessionId).filter(message => message.id !== progressMessage.id),
        result.message,
      ]))
      if (result.run?.id) {
        startRunStream(result.run, sessionId)
      } else if (result.job?.id) {
        const run = jobRunToStream(result.job, sessionId)
        if (run?.id) {
          startRunStream(run, sessionId)
        } else {
          void waitForEvaluationJob(result.job.id, sessionId)
        }
      } else {
        const snapshot = await chatService.getSession(sessionId)
        if (activeIdRef.current !== sessionId) return
        applySessionSnapshot(snapshot)
      }
    } catch (error) {
      const errorMessage = extractSendErrorMessage(error)
      markSessionRunning(sessionId, false)
      setConfirmedMsgs(previous => {
        const next = new Set(previous)
        next.delete(msgId)
        return next
      })
      setConfirmedTestCounts(previous => {
        const next = { ...previous }
        delete next[msgId]
        return next
      })
      if (activeIdRef.current === sessionId) {
        setMessages(previous => prepareMessagesForDisplay(messagesForSession(previous, sessionId).filter(message => message.id !== progressMessage.id)))
        message.error(errorMessage)
      }
    }
  }, [activeId, applySessionSnapshot, loadSessions, markSessionRunning, startRunStream, waitForEvaluationJob])

  const submitCurrentInput = useCallback(async () => {
    const trimmed = input.trim()
    const activeSending = Boolean(activeId && sendingSessionId === activeId)
    const activeRunning = Boolean(activeId && runningSessionIds.has(activeId))
    if (!trimmed || activeSending || activeRunning) return
    setInput('')
    if (activeId) {
      await sendToSession(activeId, trimmed, true)
      return
    }
    await submitPrompt(trimmed)
  }, [activeId, input, runningSessionIds, sendToSession, sendingSessionId, submitPrompt])

  const openTargetDrawer = useCallback(async () => {
    setTargetDrawerOpen(true)
    try {
      const res = await chatService.listEvaluationTargets()
      const target = (res.items || [])[0]
      if (target) {
        setCurrentTarget(target)
        targetForm.setFieldsValue({
          name: target.name,
          provider: target.provider || 'openai',
          base_url: target.base_url,
          model: target.model,
          credential_secret: '',
        })
      } else {
        setCurrentTarget(null)
        targetForm.setFieldsValue({ name: '默认被测模型', provider: 'openai', auth_type: 'bearer' })
      }
    } catch {
      targetForm.setFieldsValue({ name: '默认被测模型', provider: 'openai', auth_type: 'bearer' })
    }
  }, [targetForm])

  const saveAndProbeTarget = useCallback(async () => {
    const values = await targetForm.validateFields()
    setSavingTarget(true)
    try {
      const target = await chatService.saveEvaluationTarget({
        name: values.name,
        kind: 'llm',
        provider: values.provider,
        base_url: values.base_url,
        model: values.model,
        auth_type: 'bearer',
        credential_secret: values.credential_secret,
        status: 'published',
        metadata: {
          health_url: `${String(values.base_url || '').replace(/\/+$/, '')}/models`,
        },
      })
      setCurrentTarget(target)
      const probe = await chatService.probeEvaluationTarget(target.id)
      if (probe.status === 'healthy') {
        message.success('被测模型连接正常，后续计划会优先使用该企业 target')
      } else {
        message.warning(probe.message || probe.error || '被测模型已保存，但连通性检查未通过')
      }
      targetForm.setFieldValue('credential_secret', '')
    } catch (error) {
      message.error((error as Error).message || '保存被测模型失败')
    } finally {
      setSavingTarget(false)
    }
  }, [targetForm])

  useEffect(() => {
    loadSessions()
  }, [loadSessions])

  useEffect(() => {
    if (activeId || sessions.length === 0 || loadingSessionId) return
    const latest = mostRecentSession(sessions)
    if (latest) {
      void openSession(latest.id)
    }
  }, [activeId, loadingSessionId, openSession, sessions])

  useEffect(() => {
    void refreshCurrentTarget()
  }, [refreshCurrentTarget])

  useEffect(() => {
    loadWelcomeCapabilities()
    const handleFocus = () => { loadWelcomeCapabilities() }
    window.addEventListener('focus', handleFocus)
    return () => window.removeEventListener('focus', handleFocus)
  }, [loadWelcomeCapabilities])

  useEffect(() => {
    if (!shouldStickToBottomRef.current) return
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  useEffect(() => () => {
    streamStopRef.current?.()
  }, [])

  const activeLoading = Boolean(activeId && loadingSessionId === activeId)
  const activeSending = Boolean(activeId && sendingSessionId === activeId)
  const activeRunning = Boolean(activeId && runningSessionIds.has(activeId))
  const visibleMessages = messagesForSession(messages, activeId)
  const showWelcomeState = !activeLoading && (!activeId || visibleMessages.length === 0)
  const canSubmit = input.trim() !== '' && !activeSending && !activeRunning

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 64px)', overflow: 'hidden' }}>
      <div style={{ width: 260, borderRight: '1px solid var(--border-color)', display: 'flex', flexDirection: 'column', background: 'var(--bg-surface)', flexShrink: 0 }}>
        <div style={{ padding: '12px 12px 8px' }}>
          <Button type="primary" icon={<PlusOutlined />} block onClick={createSession} style={{ borderRadius: 8 }}>
            新建对话
          </Button>
        </div>
        <div style={{ flex: 1, overflowY: 'auto', padding: '0 8px' }}>
          {sessions.length === 0 && (
            <div style={{ color: 'var(--text-muted)', fontSize: 12, textAlign: 'center', marginTop: 32 }}>
              暂无对话记录
            </div>
          )}
          {sessions.map(session => (
            <div
              key={session.id}
              onClick={() => openSession(session.id)}
              style={{ padding: '10px 12px', borderRadius: 8, cursor: 'pointer', marginBottom: 2, background: activeId === session.id ? 'rgba(26,109,255,0.12)' : 'transparent', border: `1px solid ${activeId === session.id ? 'rgba(26,109,255,0.3)' : 'transparent'}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8 }}
            >
              <div style={{ flex: 1, overflow: 'hidden' }}>
                <div style={{ color: 'var(--text-primary)', fontSize: 13, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {session.title}
                </div>
                <div style={{ color: 'var(--text-muted)', fontSize: 11, marginTop: 2 }}>
                  {formatSessionDate(session.updated_at)}
                </div>
              </div>
              <DeleteOutlined style={{ color: 'var(--text-muted)', fontSize: 13, flexShrink: 0 }} onClick={(event) => deleteSession(session.id, event)} />
            </div>
          ))}
        </div>
      </div>

      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <div ref={messageViewportRef} onScroll={updateScrollStickiness} style={{ flex: 1, overflowY: 'auto' }}>
          {activeLoading ? (
            <div style={{ textAlign: 'center', paddingTop: 60 }}>
              <LoadingOutlined style={{ fontSize: 24, color: '#4d96ff' }} />
            </div>
          ) : showWelcomeState ? (
            <WelcomePanel items={welcomeCapabilities} onQuickPrompt={submitPrompt} />
          ) : (
            <div style={{ padding: '24px 32px' }}>
              {visibleMessages.map(msg => (
                <MessageBubble
                  key={msg.id}
                  msg={msg}
                  onPlanConfirm={(rounds, planMessageId) => handleConfirmPlan(planMessageId || msg.id, rounds)}
                  onQuickReply={(text) => {
                    if (!activeId || activeSending || activeRunning) return
                    void sendToSession(activeId, text, true)
                  }}
                  confirmedMsgs={confirmedMsgs}
                  confirmedTestCounts={confirmedTestCounts}
                  isRunning={activeRunning}
                  onResumeJob={handleResumeJob}
                  resumingJobIds={resumingJobIds}
                  onRetryJob={handleRetryJob}
                  retryingJobIds={retryingJobIds}
                />
              ))}
              {activeSending && <TypingIndicator />}
              <div ref={bottomRef} />
            </div>
          )}
        </div>

        <div style={{ padding: '12px 24px 20px', borderTop: '1px solid var(--border-color)', background: 'var(--bg-surface)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
            <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>
              {currentTarget ? `当前被测模型：${currentTarget.name}${currentTarget.model ? ` / ${currentTarget.model}` : ''}` : '请先配置本企业被测 LLM'}
            </span>
            <Button size="small" icon={<ApiOutlined />} onClick={() => { void openTargetDrawer() }}>
              被测模型连接
            </Button>
          </div>
          <div style={{ display: 'flex', gap: 10, alignItems: 'flex-end' }}>
            <TextArea
              value={input}
              onChange={(event) => setInput(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter' && !event.shiftKey) {
                  event.preventDefault()
                  if (!canSubmit) return
                  void submitCurrentInput()
                }
              }}
              placeholder={activeRunning ? '评估任务执行中，请稍候...' : '描述你的评估需求，按 Enter 发送，Shift+Enter 换行...'}
              autoSize={{ minRows: 1, maxRows: 5 }}
              disabled={activeRunning}
              style={{ flex: 1, background: 'var(--bg-card)', borderColor: 'var(--border-color)', color: 'var(--text-primary)', borderRadius: 10, resize: 'none' }}
            />
            <Button
              type="primary"
              icon={<SendOutlined />}
              onClick={() => { void submitCurrentInput() }}
              disabled={!canSubmit}
              style={{ height: 40, borderRadius: 10 }}
            />
          </div>
        </div>
      </div>

      <Drawer
        title="企业被测 LLM 连接"
        open={targetDrawerOpen}
        onClose={() => setTargetDrawerOpen(false)}
        size="large"
        footer={(
          <Space style={{ float: 'right' }}>
            <Button onClick={() => setTargetDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={savingTarget} onClick={() => { void saveAndProbeTarget() }}>保存并测试</Button>
          </Space>
        )}
      >
        <Form form={targetForm} layout="vertical" initialValues={{ name: '默认被测模型', provider: 'openai', auth_type: 'bearer' }}>
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input placeholder="默认被测模型" />
          </Form.Item>
          <Form.Item label="Provider" name="provider" rules={[{ required: true, message: '请选择 provider' }]}>
            <Select options={[
              { value: 'openai', label: 'OpenAI Compatible' },
              { value: 'anthropic', label: 'Anthropic Compatible' },
              { value: 'custom', label: 'Custom HTTP' },
            ]} />
          </Form.Item>
          <Form.Item label="Base URL" name="base_url" rules={[{ required: true, message: '请输入被测模型 Base URL' }]}>
            <Input placeholder="https://api.example.com/v1" />
          </Form.Item>
          <Form.Item label="Model" name="model" rules={[{ required: true, message: '请输入模型名' }]}>
            <Input placeholder="gpt-4.1 或本地模型名" />
          </Form.Item>
          <Form.Item label="API Key" name="credential_secret" extra="密钥只写入 maclaw，不会回显明文。留空表示沿用已保存密钥。">
            <Input.Password placeholder="sk-..." autoComplete="new-password" />
          </Form.Item>
        </Form>
      </Drawer>

      <style>{'@keyframes bounce{0%,80%,100%{transform:translateY(0);opacity:0.4}40%{transform:translateY(-6px);opacity:1}}'}</style>
    </div>
  )
}
