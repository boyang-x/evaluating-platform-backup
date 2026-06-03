export interface RuntimeRunLike {
  id?: string
  status?: string
  response_source?: string
  metadata?: Record<string, string | undefined>
}

export function isPlanConfirmRun(run?: RuntimeRunLike) {
  return Boolean(run?.response_source === 'plan_confirm' || run?.metadata?.evaluation_event_type === 'plan_confirm')
}

export function isConfirmedEvaluationRun(run?: RuntimeRunLike) {
  const metadata = run?.metadata || {}
  return (
    metadata.evaluation_action === 'confirm_plan' ||
    metadata.evaluation_execution_grant === 'true' ||
    metadata.evaluation_event_type === 'evaluation_run' ||
    run?.response_source === 'evaluation_run'
  )
}

export function shouldStreamRuntimeRun(run?: RuntimeRunLike) {
  const status = run?.status || ''
  return Boolean(
    run?.id &&
    !isPlanConfirmRun(run) &&
    isConfirmedEvaluationRun(run) &&
    (status === 'running' || status === 'pending' || status === 'queued'),
  )
}
