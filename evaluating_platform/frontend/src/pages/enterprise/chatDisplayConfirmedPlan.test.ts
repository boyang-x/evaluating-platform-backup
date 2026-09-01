import type { ChatMessage } from '../../services/chat'
import { confirmedPlanStateFromMessages } from './chatDisplay.ts'

function assertEqual(actual: unknown, expected: unknown) {
  if (actual !== expected) {
    throw new Error(`Expected ${String(expected)}, got ${String(actual)}`)
  }
}

const messages: ChatMessage[] = [
  {
    id: 'msg_plan_old',
    session_id: 'session-a',
    role: 'assistant',
    content: '',
    metadata: { card_type: 'plan_confirm' },
    created_at: '2026-05-27T01:00:00Z',
  },
  {
    id: 'msg_confirm_old',
    session_id: 'session-a',
    role: 'user',
    content: '确认执行',
    metadata: { evaluation_action: 'confirm_plan', test_count: '5' },
    created_at: '2026-05-27T01:01:00Z',
  },
  {
    id: 'msg_plan_new',
    session_id: 'session-a',
    role: 'assistant',
    content: '',
    metadata: { card_type: 'plan_confirm' },
    created_at: '2026-05-27T01:02:00Z',
  },
  {
    id: 'msg_confirm_new',
    session_id: 'session-a',
    role: 'user',
    content: '确认执行',
    metadata: { evaluation_action: 'confirm_plan', plan_message_id: 'msg_plan_new', test_count: '10' },
    created_at: '2026-05-27T01:03:00Z',
  },
]

const state = confirmedPlanStateFromMessages(messages)

assertEqual(state.confirmedIds.has('msg_plan_old'), true)
assertEqual(state.confirmedIds.has('msg_plan_new'), true)
assertEqual(state.testCounts.msg_plan_old, 5)
assertEqual(state.testCounts.msg_plan_new, 10)
