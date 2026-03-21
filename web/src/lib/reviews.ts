import type { EvaluationLogEntry, ReviewScheduleItem } from '../api/client';

export function buildDueReviewTarget(item: ReviewScheduleItem): string {
  const params = new URLSearchParams({
    mode: 'basics',
  });

  if (item.topic) {
    params.set('topic', item.topic);
  }

  return `/train?${params.toString()}`;
}

export function getDueReviewTitle(item: ReviewScheduleItem): string {
  return item.weakness_label?.trim() || item.topic?.trim() || 'Review';
}

export function normalizeEvaluationRawOutput(
  entry: Pick<EvaluationLogEntry, 'raw_output'>,
): string {
  return entry.raw_output?.trim() ?? '';
}

export function hasEvaluationRawOutput(
  entry: Pick<EvaluationLogEntry, 'raw_output'>,
): boolean {
  return normalizeEvaluationRawOutput(entry).length > 0;
}
