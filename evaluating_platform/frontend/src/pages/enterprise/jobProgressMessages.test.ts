import type { ChatMessage } from '../../services/chat'
import type { EvaluationJob, EvaluationJobRecovery } from '../../services/maclawRuntime'
import { applyEvaluationJobRecovery, buildEvaluationJobProgressMessage, jobRunToStream, markEvaluationJobRetryStarted } from './jobProgressMessages'

function assertEqual(actual: unknown, expected: unknown) {
  if (actual !== expected) {
    throw new Error(`Expected ${String(expected)}, got ${String(actual)}`)
  }
}

const failedProgress: ChatMessage = {
  id: 'job-old-failed',
  session_id: 'session-1',
  role: 'assistant',
  content: '',
  metadata: {
    card_type: 'progress',
    job_id: 'job-old',
    phase: 'failed',
    recovery_action: 'retry',
  },
  created_at: '2026-05-13T00:00:00Z',
}

const untouchedProgress: ChatMessage = {
  ...failedProgress,
  id: 'job-other-failed',
  metadata: {
    ...failedProgress.metadata,
    job_id: 'job-other',
  },
}

const updated = markEvaluationJobRetryStarted([failedProgress, untouchedProgress], 'job-old', 'job-new')

assertEqual(updated[0].metadata.retry_started, true)
assertEqual(updated[0].metadata.retry_job_id, 'job-new')
assertEqual(updated[1].metadata.retry_started, undefined)

const jobWithSteps: EvaluationJob = {
  id: 'job-steps',
  kind: 'evaluation.run',
  status: 'running',
  progress: {
    phase: 'target_call',
    step: 'calling_target',
    step_status: 'running',
    steps: [
      { step: 'materializing_resources', status: 'succeeded', retryable_on_restart: true },
      { step: 'calling_target', status: 'running', started_at: '2026-05-13T01:00:00Z' },
      { step: 'saving_results', status: 'pending' },
    ],
  },
}
const progressMessage = buildEvaluationJobProgressMessage(jobWithSteps, 'session-1', 'queued')
const steps = progressMessage.metadata.steps as Array<{ step: string; status: string }> | undefined
assertEqual(steps?.[1].step, 'calling_target')
assertEqual(steps?.[1].status, 'running')
assertEqual((steps?.[1] as { started_at?: string } | undefined)?.started_at, '2026-05-13T01:00:00Z')
assertEqual(progressMessage.metadata.phase, 'target_call')
assertEqual(progressMessage.metadata.status_text, '正在调用被测模型。')

const jobWithCounts: EvaluationJob = {
  id: 'job-counts',
  kind: 'evaluation.run',
  status: 'running',
  progress: {
    phase: 'target_calls',
    status_text: '正在调用被测模型。',
    planned_count: 20,
    executed_count: 7,
    current_stage: 'target_calls',
  },
}
const countProgressMessage = buildEvaluationJobProgressMessage(jobWithCounts, 'session-1', 'queued')
assertEqual(countProgressMessage.metadata.planned_count, 20)
assertEqual(countProgressMessage.metadata.executed_count, 7)
assertEqual(countProgressMessage.metadata.current_stage, 'target_calls')

const pendingQueuedJob: EvaluationJob = {
  id: 'job-pending',
  kind: 'evaluation.run',
  status: 'pending',
}
const queuedMessage = buildEvaluationJobProgressMessage(pendingQueuedJob, 'session-1', 'queued')
assertEqual(queuedMessage.metadata.status_text, '评测任务已提交，正在排队执行。')

const jobWithoutRecoveryAction: EvaluationJob = {
  id: 'job-recovery',
  kind: 'evaluation.run',
  status: 'failed',
  progress: {
    phase: 'running',
    step: 'materializing_resources',
    status_text: 'interrupted',
  },
}

const retryRecovery: EvaluationJobRecovery = {
  job_id: 'job-recovery',
  run_id: 'run-recovery',
  action: 'retry',
  strategy: 'retry_safe',
  step: 'materializing_resources',
  completed_step_ids: ['materializing_resources'],
  can_retry: true,
  manual_review_required: false,
  reason: 'interrupted before target call',
  recovery_indexed: true,
  replacement_job_endpoint: '/api/v1/jobs/job-recovery/retry',
}

const jobWithRecovery = applyEvaluationJobRecovery(jobWithoutRecoveryAction, retryRecovery)
const recoveryMessage = buildEvaluationJobProgressMessage(jobWithRecovery, 'session-1', 'failed')

assertEqual(jobWithRecovery.progress?.recovery_action, 'retry')
assertEqual(jobWithRecovery.progress?.recovery_strategy, 'retry_safe')
assertEqual(jobWithRecovery.progress?.restart_interrupted, true)
assertEqual(jobWithRecovery.progress?.run_id, 'run-recovery')
assertEqual(recoveryMessage.metadata.recovery_action, 'retry')

const resumeRecovery: EvaluationJobRecovery = {
  job_id: 'job-recovery',
  run_id: 'run-recovery',
  action: 'manual_review',
  strategy: 'manual_review',
  step: 'saving_results',
  can_resume: true,
  resume_step: 'saving_results',
  resume_job_endpoint: '/api/v1/jobs/job-recovery/resume',
  can_retry: false,
  manual_review_required: false,
  reason: 'saved artifacts can be finalized safely',
  recovery_indexed: true,
}

const resumeJob = applyEvaluationJobRecovery(jobWithoutRecoveryAction, resumeRecovery)
const resumeMessage = buildEvaluationJobProgressMessage(resumeJob, 'session-1', 'failed')

assertEqual(resumeJob.progress?.recovery_action, 'resume')
assertEqual(resumeJob.progress?.can_resume, true)
assertEqual(resumeJob.progress?.resume_step, 'saving_results')
assertEqual(resumeMessage.metadata.recovery_action, 'resume')
assertEqual(resumeMessage.metadata.can_resume, true)

const succeededConfirmJob: EvaluationJob = {
  id: 'run-confirmed',
  kind: 'evaluation.run',
  status: 'succeeded',
  progress: {
    run_id: 'run-confirmed',
    session_id: 'session-1',
  },
}

assertEqual(jobRunToStream(succeededConfirmJob, 'session-1'), undefined)

const runningConfirmJob: EvaluationJob = {
  ...succeededConfirmJob,
  status: 'running',
}

assertEqual(jobRunToStream(runningConfirmJob, 'session-1')?.id, 'run-confirmed')
