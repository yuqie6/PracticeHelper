import { describe, expect, it } from 'vitest';

import type { Dashboard, ProjectProfile } from '../api/client';
import {
  buildOnboardingSnapshot,
  buildOnboardingTarget,
  resolveOnboardingSteps,
  shouldShowOnboarding,
} from './onboarding';

function makeDashboard(partial?: Partial<Dashboard>): Dashboard {
  return {
    profile: null,
    weaknesses: [],
    recent_sessions: [],
    current_session: null,
    today_focus: '',
    recommended_track: '',
    recommendation_scope: 'generic',
    ...partial,
  };
}

function makeProject(partial?: Partial<ProjectProfile>): ProjectProfile {
  return {
    id: 'proj_1',
    name: 'Mirror',
    repo_url: '',
    default_branch: 'main',
    import_commit: '',
    summary: 'Agent workflow',
    tech_stack: [],
    highlights: [],
    challenges: [],
    tradeoffs: [],
    ownership_points: [],
    followup_points: [],
    import_status: 'completed',
    ...partial,
  };
}

describe('onboarding helpers', () => {
  it('marks profile as current when profile is missing', () => {
    const snapshot = buildOnboardingSnapshot(makeDashboard(), []);

    expect(shouldShowOnboarding(snapshot)).toBe(true);
    expect(resolveOnboardingSteps(snapshot)).toEqual([
      { key: 'profile', state: 'current' },
      { key: 'projects', state: 'next' },
      { key: 'train', state: 'next' },
    ]);
  });

  it('marks train as current after profile and projects are ready', () => {
    const snapshot = buildOnboardingSnapshot(
      makeDashboard({
        profile: {
          id: 1,
          target_role: '后端工程师',
          target_company_type: '互联网',
          current_stage: '在职',
          tech_stacks: [],
          primary_projects: [],
          self_reported_weaknesses: [],
        },
      }),
      [makeProject()],
    );

    expect(resolveOnboardingSteps(snapshot)).toEqual([
      { key: 'profile', state: 'done' },
      { key: 'projects', state: 'done' },
      { key: 'train', state: 'current' },
    ]);
  });

  it('hides onboarding after a session already exists', () => {
    const snapshot = buildOnboardingSnapshot(
      makeDashboard({
        profile: {
          id: 1,
          target_role: '后端工程师',
          target_company_type: '互联网',
          current_stage: '在职',
          tech_stacks: [],
          primary_projects: [],
          self_reported_weaknesses: [],
        },
        recent_sessions: [
          {
            id: 'sess_1',
            mode: 'basics',
            status: 'completed',
            total_score: 82,
            updated_at: '2026-03-22T00:00:00Z',
          },
        ],
      }),
      [],
    );

    expect(shouldShowOnboarding(snapshot)).toBe(false);
  });

  it('builds onboarding targets with query params', () => {
    expect(buildOnboardingTarget('profile')).toBe('/profile?onboarding=1');
    expect(buildOnboardingTarget('projects')).toBe('/projects?onboarding=1');
    expect(buildOnboardingTarget('train', { skipProjects: true })).toBe(
      '/train?onboarding=1&skip_projects=1',
    );
  });
});
