import type { TrainingSessionSummary } from '../api/client';
import type { Translate } from './labels';
import { formatModeLabel, formatTopicLabel } from './labels';

export function formatSessionName(
  t: Translate,
  session: TrainingSessionSummary,
): string {
  if (session.project_name) {
    return session.project_name;
  }

  if (session.topic) {
    return formatTopicLabel(t, session.topic);
  }

  if (session.mode) {
    return formatModeLabel(t, session.mode);
  }

  return t('common.unknownSession');
}

export function buildSessionTarget(
  session: Pick<TrainingSessionSummary, 'id' | 'status' | 'review_id'>,
): string {
  if (session.status === 'completed' && session.review_id) {
    return `/reviews/${session.review_id}`;
  }

  return `/sessions/${session.id}`;
}
