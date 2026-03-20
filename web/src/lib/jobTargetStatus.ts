import type { JobTarget, JobTargetRef } from '../api/client';
import type { Translate } from './labels';

type JobTargetStatus =
  | JobTarget['latest_analysis_status']
  | JobTargetRef['latest_analysis_status']
  | undefined
  | null;

type JobTargetStatusVariant =
  | 'homeActive'
  | 'trainSelection'
  | 'trainFallback'
  | 'jobsReadiness'
  | 'jobsSnapshot';

const KNOWN_STATUSES = new Set([
  'idle',
  'running',
  'succeeded',
  'failed',
  'stale',
]);

export function isJobTargetReady(status: JobTargetStatus): boolean {
  return status === 'succeeded';
}

export function describeJobTargetStatus(
  t: Translate,
  variant: JobTargetStatusVariant,
  status: JobTargetStatus,
  named?: Record<string, unknown>,
): string {
  const normalized =
    typeof status === 'string' && KNOWN_STATUSES.has(status)
      ? status
      : 'unknown';
  return t(`jobTargetStatus.${variant}.${normalized}`, named);
}
