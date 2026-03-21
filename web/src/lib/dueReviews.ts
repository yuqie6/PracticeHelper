import type { ReviewScheduleItem } from '../api/client';

export function buildDueReviewTarget(item: ReviewScheduleItem): string {
  if (item.topic) {
    const params = new URLSearchParams();
    params.set('mode', 'basics');
    params.set('topic', item.topic);
    return `/train?${params.toString()}`;
  }

  if (item.weakness_tag_id || item.weakness_label || item.weakness_kind) {
    return '/train?mode=basics';
  }

  if (item.review_card_id) {
    return `/reviews/${item.review_card_id}`;
  }

  if (item.session_id) {
    return `/sessions/${item.session_id}`;
  }

  return '/train';
}

export function resolveDueReviewHeadline(item: ReviewScheduleItem): string {
  if (item.weakness_label) {
    return item.weakness_label;
  }
  return '';
}
