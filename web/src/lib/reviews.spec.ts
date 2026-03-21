import { describe, expect, it } from 'vitest';

import type { EvaluationLogEntry, ReviewScheduleItem } from '../api/client';
import {
  buildDueReviewTarget,
  getDueReviewTitle,
  hasEvaluationRawOutput,
  normalizeEvaluationRawOutput,
} from './reviews';

function makeDueReview(
  partial?: Partial<ReviewScheduleItem>,
): ReviewScheduleItem {
  return {
    id: 1,
    session_id: 'sess_1',
    next_review_at: '2026-03-22T00:00:00Z',
    interval_days: 1,
    ...partial,
  };
}

function makeEvaluationLog(
  partial?: Partial<EvaluationLogEntry>,
): EvaluationLogEntry {
  return {
    id: 1,
    session_id: 'sess_1',
    flow_name: 'evaluate_answer',
    latency_ms: 12.5,
    created_at: '2026-03-22T00:00:00Z',
    ...partial,
  };
}

describe('review helpers', () => {
  it('builds due review target with topic when available', () => {
    expect(
      buildDueReviewTarget(
        makeDueReview({
          topic: 'redis',
        }),
      ),
    ).toBe('/train?mode=basics&topic=redis');
  });

  it('falls back to basics training when due review has no topic', () => {
    expect(buildDueReviewTarget(makeDueReview())).toBe('/train?mode=basics');
  });

  it('prefers weakness label for due review title', () => {
    expect(
      getDueReviewTitle(
        makeDueReview({
          weakness_label: '缓存一致性',
          topic: 'redis',
        }),
      ),
    ).toBe('缓存一致性');
  });

  it('normalizes raw output for audit display', () => {
    const entry = makeEvaluationLog({
      raw_output: '  {"score":82}  ',
    });

    expect(normalizeEvaluationRawOutput(entry)).toBe('{"score":82}');
    expect(hasEvaluationRawOutput(entry)).toBe(true);
    expect(
      hasEvaluationRawOutput(makeEvaluationLog({ raw_output: '   ' })),
    ).toBe(false);
  });
});
