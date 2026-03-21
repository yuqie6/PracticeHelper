import type { Dashboard } from '../api/client';

import {
  formatModeLabel,
  formatStatusLabel,
  formatTopicLabel,
  formatWeaknessKindLabel,
  type Translate,
} from './labels';

export function describeWeakness(
  dashboard: Dashboard | null,
  t: Translate,
): string {
  const weakness = dashboard?.weaknesses?.[0];
  if (!weakness) {
    return t('home.cards.weaknessEmpty');
  }

  return t('home.cards.weaknessDescription', {
    kind: formatWeaknessKindLabel(t, weakness.kind),
    label: weakness.label,
  });
}

export function describeSession(
  dashboard: Dashboard | null,
  t: Translate,
): string {
  const session = dashboard?.recent_sessions?.[0];
  if (!session) {
    return t('home.cards.sessionEmpty');
  }

  // 首页最近训练卡片优先展示用户最容易识别的名字：项目名 > topic > 模式名。
  const name =
    session.project_name ||
    (session.topic
      ? formatTopicLabel(t, session.topic)
      : formatModeLabel(t, session.mode));

  return t('home.cards.sessionDescription', {
    name,
    status: formatStatusLabel(t, session.status),
  });
}

export function describeProfile(
  dashboard: Dashboard | null,
  t: Translate,
): string {
  const profile = dashboard?.profile;
  if (!profile || !hasMeaningfulProfile(profile)) {
    return t('home.cards.profileEmpty');
  }

  const projects =
    profile.primary_projects.join(', ') || t('common.notProvided');

  return t('home.cards.profileDescription', {
    role: profile.target_role || t('common.notProvided'),
    stage: profile.current_stage || t('common.notProvided'),
    projects,
  });
}

function hasMeaningfulProfile(profile: Dashboard['profile']): boolean {
  if (!profile) {
    return false;
  }
  return Boolean(
    profile.target_role ||
    profile.target_company_type ||
    profile.current_stage ||
    profile.application_deadline ||
    profile.tech_stacks.length ||
    profile.primary_projects.length ||
    profile.self_reported_weaknesses.length,
  );
}
