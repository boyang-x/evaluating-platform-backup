import { useCallback, useEffect, useRef, useState, type Dispatch, type MouseEvent, type SetStateAction } from 'react'
import { Button, Input, message } from 'antd'
import { DeleteOutlined, LoadingOutlined, PlusOutlined, SendOutlined } from '@ant-design/icons'

import { chatService } from '../../services/chat'
import type { ChatMessage, ChatSession, WelcomeCapability } from '../../services/chat'
import { MessageBubble, TypingIndicator, WelcomePanel } from './ChatCards'
import { DEFAULT_WELCOME_CAPABILITIES, formatSessionDate, normalizeSessionTimestamps, prepareMessagesForDisplay } from './chatDisplay'

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

function extractSendErrorMessage(error: unknown) {
  if (typeof error === 'object' && error !== null) {
    const response = (error as { response?: { data?: { error?: unknown } } }).response
    const detail = response?.data?.error
    if (typeof detail === 'string' && detail.trim()) {
      return detail
    }
  }
  return '发送失败，请检查编排 LLM 配置或稍后重试'
}

async function sendPrompt(
  sessionId: string,
  text: string,
  append: boolean,
  setMessages: Dispatch<SetStateAction<ChatMessage[]>>,
  setSending: (value: boolean) => void,
  onSessionUpdated: () => void,
) {
  setSending(true)
  const tempUserMessage = buildTemporaryUserMessage(sessionId, text)
  setMessages(previous => (append ? [...previous, tempUserMessage] : [tempUserMessage]))

  try {
    const assistantMessage = await chatService.sendMessage(sessionId, text)
    setMessages(previous => {
      const withoutTemp = previous.filter(message => message.id !== tempUserMessage.id)
      return append ? [...withoutTemp, tempUserMessage, assistantMessage] : [tempUserMessage, assistantMessage]
    })
    onSessionUpdated()
  } catch (error) {
    setMessages(previous => previous.filter(message => message.id !== tempUserMessage.id))
    message.error(extractSendErrorMessage(error))
  } finally {
    setSending(false)
  }
}

export function ChatPage() {
  const [sessions, setSessions] = useState<ChatSession[]>([])
  const [activeId, setActiveId] = useState<string | null>(null)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [welcomeCapabilities, setWelcomeCapabilities] = useState<WelcomeCapability[]>(DEFAULT_WELCOME_CAPABILITIES)
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const [loadingSession, setLoadingSession] = useState(false)
  const [confirmedMsgs, setConfirmedMsgs] = useState<Set<string>>(new Set())
  const [isRunning, setIsRunning] = useState(false)

  const messageViewportRef = useRef<HTMLDivElement>(null)
  const bottomRef = useRef<HTMLDivElement>(null)
  const shouldStickToBottomRef = useRef(true)

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
    setMessages(prepareMessagesForDisplay(snapshot.messages || [], snapshot.session))
    setIsRunning(snapshot.session.state === 'running')
  }, [])

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

  const openSession = useCallback(async (id: string) => {
    shouldStickToBottomRef.current = true
    setActiveId(id)
    setLoadingSession(true)
    try {
      const snapshot = await chatService.getSession(id)
      applySessionSnapshot(snapshot)
    } catch {
      // Ignore open failure and keep the previous content rendered.
    } finally {
      setLoadingSession(false)
    }
  }, [applySessionSnapshot])

  const createSession = useCallback(async () => {
    const session = normalizeSessionTimestamps(await chatService.createSession())
    setSessions(previous => [session, ...previous])
    setActiveId(session.id)
    setMessages([])
    setIsRunning(false)
    setConfirmedMsgs(new Set())
    shouldStickToBottomRef.current = true
    return session
  }, [])

  const deleteSession = useCallback(async (id: string, event: MouseEvent) => {
    event.stopPropagation()
    await chatService.deleteSession(id)
    setSessions(previous => previous.filter(session => session.id !== id))
    if (activeId === id) {
      setActiveId(null)
      setMessages([])
      setIsRunning(false)
      setConfirmedMsgs(new Set())
    }
  }, [activeId])

  const sendToSession = useCallback(async (sessionId: string, text: string, append: boolean) => {
    await sendPrompt(sessionId, text, append, setMessages, setSending, loadSessions)
  }, [loadSessions])

  const submitPrompt = useCallback(async (text: string) => {
    const trimmed = text.trim()
    if (!trimmed || sending || isRunning) return

    if (activeId && messages.length === 0) {
      await sendToSession(activeId, trimmed, false)
      return
    }

    const session = await createSession()
    await sendToSession(session.id, trimmed, false)
  }, [activeId, createSession, isRunning, messages.length, sendToSession, sending])

  const handleConfirmPlan = useCallback(async (msgId: string, testCount: number) => {
    if (!activeId) return
    try {
      shouldStickToBottomRef.current = true
      await chatService.confirmPlan(activeId, testCount)
      setConfirmedMsgs(previous => new Set([...previous, msgId]))
      setIsRunning(true)
      const snapshot = await chatService.getSession(activeId)
      applySessionSnapshot(snapshot)
    } catch {
      message.error('启动评估失败，请稍后重试')
    }
  }, [activeId, applySessionSnapshot])

  const submitCurrentInput = useCallback(async () => {
    const trimmed = input.trim()
    if (!trimmed || sending || isRunning) return
    setInput('')
    if (activeId) {
      await sendToSession(activeId, trimmed, true)
      return
    }
    await submitPrompt(trimmed)
  }, [activeId, input, isRunning, sendToSession, sending, submitPrompt])

  useEffect(() => {
    loadSessions()
  }, [loadSessions])

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

  useEffect(() => {
    if (!activeId || !isRunning) return
    const timer = setInterval(async () => {
      try {
        const snapshot = await chatService.getSession(activeId)
        applySessionSnapshot(snapshot)
      } catch {
        // Silent retry on next poll.
      }
    }, 3000)
    return () => clearInterval(timer)
  }, [activeId, applySessionSnapshot, isRunning])

  const showWelcomeState = !loadingSession && (!activeId || messages.length === 0)
  const canSubmit = input.trim() !== '' && !sending && !isRunning

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
          {loadingSession ? (
            <div style={{ textAlign: 'center', paddingTop: 60 }}>
              <LoadingOutlined style={{ fontSize: 24, color: '#4d96ff' }} />
            </div>
          ) : showWelcomeState ? (
            <WelcomePanel items={welcomeCapabilities} onQuickPrompt={submitPrompt} />
          ) : (
            <div style={{ padding: '24px 32px' }}>
              {messages.map(msg => (
                <MessageBubble
                  key={msg.id}
                  msg={msg}
                  onPlanConfirm={(rounds) => handleConfirmPlan(msg.id, rounds)}
                  confirmedMsgs={confirmedMsgs}
                  isRunning={isRunning}
                />
              ))}
              {sending && <TypingIndicator />}
              <div ref={bottomRef} />
            </div>
          )}
        </div>

        <div style={{ padding: '12px 24px 20px', borderTop: '1px solid var(--border-color)', background: 'var(--bg-surface)' }}>
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
              placeholder={isRunning ? '评估任务执行中，请稍候...' : '描述你的评估需求，按 Enter 发送，Shift+Enter 换行...'}
              autoSize={{ minRows: 1, maxRows: 5 }}
              disabled={isRunning}
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

      <style>{'@keyframes bounce{0%,80%,100%{transform:translateY(0);opacity:0.4}40%{transform:translateY(-6px);opacity:1}}'}</style>
    </div>
  )
}
