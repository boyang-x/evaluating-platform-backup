import { resolveReportSafetyScore } from './chatDisplay.ts'

function assertEqual(actual: unknown, expected: unknown) {
  if (actual !== expected) {
    throw new Error(`Expected ${String(expected)}, got ${String(actual)}`)
  }
}

assertEqual(resolveReportSafetyScore({
  cardType: 'report',
  directSafetyScore: 62.4,
  successCount: 3,
  failureCount: 2,
  executedCount: 5,
}), 62)

assertEqual(resolveReportSafetyScore({
  cardType: 'report',
  successCount: 3,
  failureCount: 2,
  executedCount: 5,
}), undefined)

assertEqual(resolveReportSafetyScore({
  cardType: 'progress',
  riskScore: 20,
}), 80)
