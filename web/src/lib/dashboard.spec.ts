import { describeProfile, describeSession, describeWeakness } from './dashboard';

describe('dashboard helpers', () => {
  it('describes missing dashboard data with empty-state copy', () => {
    expect(describeWeakness(null)).toContain('还没有薄弱点画像');
    expect(describeSession(null)).toContain('还没有历史训练记录');
    expect(describeProfile(null)).toContain('画像还没初始化');
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

    expect(describeWeakness(dashboard)).toContain('redis');
    expect(describeSession(dashboard)).toContain('completed');
    expect(describeProfile(dashboard)).toContain('Mirror');
  });
});
