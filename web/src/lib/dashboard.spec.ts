import { describeProfile, describeSession, describeWeakness } from './dashboard';
import { messages } from '../i18n/messages';

function createTranslator(locale: 'zh-CN' | 'en' = 'zh-CN') {
  return (key: string, named?: Record<string, unknown>) => {
    const value = key.split('.').reduce<unknown>((current, segment) => {
      if (!current || typeof current !== 'object') {
        return undefined;
      }

      return (current as Record<string, unknown>)[segment];
    }, messages[locale]);

    if (typeof value !== 'string') {
      return key;
    }

    return value.replace(/\{(\w+)\}/g, (_, token) => String(named?.[token] ?? ''));
  };
}

describe('dashboard helpers', () => {
  const t = createTranslator();

  it('describes missing dashboard data with empty-state copy', () => {
    expect(describeWeakness(null, t)).toContain('还没有薄弱点记录');
    expect(describeSession(null, t)).toContain('还没有历史训练记录');
    expect(describeProfile(null, t)).toContain('画像还没初始化');
  });

  it('describes populated dashboard data', () => {
    const dashboard = {
      profile: {
        id: 1,
        target_role: 'Go 后端',
        target_company_type: '创业团队',
        current_stage: '实习前',
        application_deadline: '2026-04-01',
        tech_stacks: ['Go'],
        primary_projects: ['Mirror'],
        self_reported_weaknesses: ['Kafka'],
      },
      weaknesses: [
        {
          id: 'weak_1',
          kind: 'topic',
          label: 'redis',
          severity: 0.8,
          frequency: 2,
          last_seen_at: '2026-03-19T00:00:00Z',
          evidence_session_id: 'sess_1',
        },
      ],
      recent_sessions: [
        {
          id: 'sess_1',
          mode: 'basics',
          topic: 'redis',
          project_name: '',
          status: 'completed',
          total_score: 68,
          updated_at: '2026-03-19T00:00:00Z',
        },
      ],
      today_focus: '先补 Redis',
      recommended_track: 'Redis 专项',
      days_until_deadline: 13,
    };

    expect(describeWeakness(dashboard, t)).toContain('知识点');
    expect(describeWeakness(dashboard, t)).toContain('redis');
    expect(describeSession(dashboard, t)).toContain('已完成');
    expect(describeProfile(dashboard, t)).toContain('Mirror');
  });
});
