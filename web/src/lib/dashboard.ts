import type { Dashboard } from '../api/client';

export function describeWeakness(dashboard: Dashboard | null): string {
  const weakness = dashboard?.weaknesses?.[0];
  if (!weakness) {
    return '还没有薄弱点画像，说明你还没开始被系统追着练。';
  }
  return `当前最该补的是 ${weakness.kind} / ${weakness.label}，系统会继续围着它出题。`;
}

export function describeSession(dashboard: Dashboard | null): string {
  const session = dashboard?.recent_sessions?.[0];
  if (!session) {
    return '还没有历史训练记录，做完第一轮后这里会显示最近一次打分和状态。';
  }
  return `最近一轮是 ${session.project_name || session.topic || session.mode}，状态 ${session.status}。`;
}

export function describeProfile(dashboard: Dashboard | null): string {
  const profile = dashboard?.profile;
  if (!profile) {
    return '画像还没初始化，系统现在不知道你要投什么岗、该优先练什么。';
  }

  const projects = profile.primary_projects.join('、') || '暂未填写';
  return `${profile.target_role} / ${profile.current_stage}，主讲项目 ${projects}。`;
}
