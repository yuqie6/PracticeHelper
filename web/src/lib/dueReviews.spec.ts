import { describe, expect, it } from 'vitest';

import { buildDueReviewTarget, resolveDueReviewHeadline } from './dueReviews';

describe('due review helpers', () => {
  it('prefers direct training links when topic is available', () => {
    expect(
      buildDueReviewTarget({
        id: 1,
        session_id: 'sess_1',
        topic: 'redis',
        next_review_at: '2026-03-22T00:00:00Z',
        interval_days: 1,
      }),
    ).toBe('/train?mode=basics&topic=redis');
  });

  it('still goes straight to training for weakness-level reviews without topic', () => {
    expect(
      buildDueReviewTarget({
        id: 2,
        session_id: 'sess_2',
        weakness_tag_id: 'weak_1',
        weakness_label: '缓存一致性',
        next_review_at: '2026-03-22T00:00:00Z',
        interval_days: 1,
      }),
    ).toBe('/train?mode=basics');
  });

  it('falls back to review details when it is still a legacy session-level item', () => {
    expect(
      buildDueReviewTarget({
        id: 3,
        session_id: 'sess_3',
        review_card_id: 'review_3',
        next_review_at: '2026-03-22T00:00:00Z',
        interval_days: 1,
      }),
    ).toBe('/reviews/review_3');
  });

  it('uses weakness label as the primary headline when available', () => {
    expect(
      resolveDueReviewHeadline({
        id: 4,
        session_id: 'sess_4',
        weakness_label: '表达不够具体',
        topic: 'go',
        next_review_at: '2026-03-22T00:00:00Z',
        interval_days: 1,
      }),
    ).toBe('表达不够具体');
  });

  it('returns empty headline when only topic is available', () => {
    expect(
      resolveDueReviewHeadline({
        id: 5,
        session_id: 'sess_5',
        topic: 'redis',
        next_review_at: '2026-03-22T00:00:00Z',
        interval_days: 1,
      }),
    ).toBe('');
  });
});
