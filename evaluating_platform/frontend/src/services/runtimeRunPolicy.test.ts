import { shouldStreamRuntimeRun } from './runtimeRunPolicy.ts'

function assertEqual(actual: unknown, expected: unknown) {
  if (actual !== expected) {
    throw new Error(`Expected ${String(expected)}, got ${String(actual)}`)
  }
}

assertEqual(shouldStreamRuntimeRun({
  id: 'chat-run',
  status: 'running',
  response_source: 'chat',
}), false)

assertEqual(shouldStreamRuntimeRun({
  id: 'ask-run',
  status: 'queued',
  response_source: 'ask_user',
}), false)

assertEqual(shouldStreamRuntimeRun({
  id: 'plan-run',
  status: 'running',
  response_source: 'plan_confirm',
}), false)

assertEqual(shouldStreamRuntimeRun({
  id: 'eval-run',
  status: 'running',
  metadata: { evaluation_action: 'confirm_plan' },
}), true)
