import type { ChatMessage } from '../../services/chat'
import type { EvaluationJob, EvaluationJobRecovery, RuntimeRun } from '../../services/maclawRuntime'

export function applyEvaluationJobRecovery(job: EvaluationJob, recovery?: EvaluationJobRecovery): EvaluationJob {
  if (!recovery) {
    return job
  }
  const progress = job.progress || {}
  const recoveryAction = recovery.can_resume ? 'resume' : recovery.action || progress.recovery_action
  return {
    ...job,
    progress: {
      ...progress,
      step: recovery.step || progress.step,
      run_id: recovery.run_id || progress.run_id,
      status_text: progress.status_text || recovery.reason,
      restart_interrupted: progress.restart_interrupted || recovery.recovery_indexed || recovery.manual_review_required || recovery.can_retry,
      recovery_strategy: recovery.strategy || progress.recovery_strategy,
      recovery_action: recoveryAction,
      can_resume: recovery.can_resume || progress.can_resume,
      resume_step: recovery.resume_step || progress.resume_step,
      resume_job_endpoint: recovery.resume_job_endpoint || progress.resume_job_endpoint,
    },
  }
}

export function buildEvaluationJobProgressMessage(
  job: EvaluationJob,
  sessionId: string,
  phase: 'queued' | 'failed' | 'canceled',
  options?: { retryOfJobId?: string; resumeOfJobId?: string },
): ChatMessage {
  const progress = job.progress
  const hasRun = Boolean(progress?.run_id || job.result?.run?.id || job.status === 'running' || job.status === 'succeeded')
  const displayPhase = phase === 'queued' && hasRun
    ? (progress?.phase || (job.status === 'succeeded' ? 'reporting' : 'starting'))
    : phase
  const defaultStatus = displayPhase === 'queued'
    ? '评测任务已提交，正在排队执行。'
    : phase === 'canceled'
      ? '评测任务已取消。'
      : displayPhase === 'target_call' || displayPhase === 'calling_target'
        ? '正在调用被测模型。'
        : displayPhase === 'compose_payloads' || displayPhase === 'composing_payloads'
          ? '正在组合评估载荷。'
          : displayPhase === 'reporting'
            ? '正在生成评估报告。'
            : phase === 'failed'
              ? '评测任务未能完成。'
              : '评测任务已启动，正在连接执行进度。'

  return {
    id: `job-${job.id}-${phase}`,
    session_id: sessionId,
    role: 'assistant',
    content: '',
    metadata: {
      card_type: 'progress',
      job_id: job.id,
      retry_of_job_id: options?.retryOfJobId,
      resume_of_job_id: options?.resumeOfJobId,
      assessment_id: progress?.run_id || job.result?.run?.id,
      phase: displayPhase,
      status_text: job.error || progress?.status_text || defaultStatus,
      duration_ms: progress?.duration_ms,
      stage_durations_json: progress?.stage_durations_json,
      steps: progress?.steps,
      restart_interrupted: progress?.restart_interrupted,
      recovery_strategy: progress?.recovery_strategy,
      recovery_action: progress?.recovery_action,
      can_resume: progress?.can_resume,
      resume_step: progress?.resume_step,
      resume_job_endpoint: progress?.resume_job_endpoint,
    },
    created_at: job.completed_at || job.created_at || new Date().toISOString(),
  }
}

export function jobRunToStream(job: EvaluationJob, sessionId: string): RuntimeRun | undefined {
  if (job.status === 'succeeded' || job.status === 'failed' || job.status === 'canceled') {
    return undefined
  }
  if (job.progress?.run_id) {
    return {
      id: job.progress.run_id,
      session_id: job.progress.session_id || sessionId,
      status: job.status === 'pending' ? 'pending' : 'running',
      metadata: { job_id: job.id, progress_phase: job.progress.phase || job.status || 'running' },
    }
  }
  return job.result?.run
}

export function markEvaluationJobRecoveryStarted(
  messages: ChatMessage[],
  originalJobId: string,
  replacementJobId: string,
  kind: 'retry' | 'resume',
): ChatMessage[] {
  return messages.map(message => {
    if (message.metadata?.card_type !== 'progress' || message.metadata.job_id !== originalJobId) {
      return message
    }
    return {
      ...message,
      metadata: {
        ...message.metadata,
        [`${kind}_started`]: true,
        [`${kind}_job_id`]: replacementJobId,
      },
    }
  })
}

export function markEvaluationJobRetryStarted(messages: ChatMessage[], originalJobId: string, retryJobId: string): ChatMessage[] {
  return markEvaluationJobRecoveryStarted(messages, originalJobId, retryJobId, 'retry')
}
