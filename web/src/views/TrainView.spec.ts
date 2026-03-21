import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query';
import {
  RouterLinkStub,
  flushPromises,
  mount,
  type VueWrapper,
} from '@vue/test-utils';
import { createMemoryHistory, createRouter } from 'vue-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { i18n } from '../i18n';
import type {
  Dashboard,
  JobTarget,
  PromptSetSummary,
  TrainingSession,
} from '../api/client';
import TrainView from './TrainView.vue';

const apiMocks = vi.hoisted(() => ({
  createSessionStream: vi.fn(),
  getDashboard: vi.fn(),
  listJobTargets: vi.fn(),
  listProjects: vi.fn(),
  listPromptSets: vi.fn(),
}));

vi.mock('../api/client', async () => {
  const actual =
    await vi.importActual<typeof import('../api/client')>('../api/client');
  return {
    ...actual,
    ...apiMocks,
  };
});

function buildDashboard(overrides: Partial<Dashboard> = {}): Dashboard {
  return {
    profile: null,
    weaknesses: [],
    recent_sessions: [],
    current_session: null,
    today_focus: '',
    recommended_track: '',
    recommendation_scope: 'generic',
    ...overrides,
  };
}

function buildJobTarget(overrides: Partial<JobTarget> = {}): JobTarget {
  return {
    id: 'jt_1',
    title: '后端工程师',
    company_name: 'Example',
    source_text: '要求 Go 和 Redis。',
    latest_analysis_status: 'succeeded',
    latest_analysis_id: 'run_1',
    created_at: '2026-03-22T00:00:00Z',
    updated_at: '2026-03-22T00:00:00Z',
    ...overrides,
  };
}

function buildPromptSet(
  overrides: Partial<PromptSetSummary> = {},
): PromptSetSummary {
  return {
    id: 'stable-v1',
    label: 'Stable v1',
    status: 'stable',
    is_default: true,
    ...overrides,
  };
}

function buildSession(
  overrides: Partial<TrainingSession> = {},
): TrainingSession {
  return {
    id: 'sess_1',
    mode: 'basics',
    topic: 'go',
    intensity: 'standard',
    status: 'waiting_answer',
    max_turns: 2,
    total_score: 0,
    turns: [],
    ...overrides,
  };
}

async function mountTrainView(initialPath = '/train'): Promise<VueWrapper> {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/train', component: TrainView },
      { path: '/sessions/:id', component: { template: '<div />' } },
      { path: '/reviews/:id', component: { template: '<div />' } },
    ],
  });
  await router.push(initialPath);
  await router.isReady();

  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return mount(TrainView, {
    global: {
      plugins: [router, [VueQueryPlugin, { queryClient }], i18n],
      stubs: {
        RouterLink: RouterLinkStub,
      },
    },
  });
}

beforeEach(() => {
  apiMocks.getDashboard.mockReset();
  apiMocks.listJobTargets.mockReset();
  apiMocks.listProjects.mockReset();
  apiMocks.listPromptSets.mockReset();
  apiMocks.createSessionStream.mockReset();

  apiMocks.getDashboard.mockResolvedValue(buildDashboard());
  apiMocks.listJobTargets.mockResolvedValue([]);
  apiMocks.listProjects.mockResolvedValue([]);
  apiMocks.listPromptSets.mockResolvedValue([buildPromptSet()]);
  apiMocks.createSessionStream.mockResolvedValue(buildSession());
});

describe('TrainView', () => {
  it('submits with the active succeeded job target and default prompt set', async () => {
    apiMocks.getDashboard.mockResolvedValue(
      buildDashboard({
        recommendation_scope: 'job_target',
        active_job_target: {
          id: 'jt_ready',
          title: '后端主程',
          latest_analysis_status: 'succeeded',
        },
      }),
    );
    apiMocks.listJobTargets.mockResolvedValue([
      buildJobTarget({
        id: 'jt_ready',
        title: '后端主程',
        latest_analysis_status: 'succeeded',
      }),
    ]);
    apiMocks.createSessionStream.mockResolvedValue(
      buildSession({ id: 'sess_bound' }),
    );

    const wrapper = await mountTrainView();
    await flushPromises();

    await wrapper.get('form').trigger('submit.prevent');
    await flushPromises();

    expect(apiMocks.createSessionStream).toHaveBeenCalledWith(
      expect.objectContaining({
        mode: 'basics',
        topic: 'go',
        job_target_id: 'jt_ready',
        prompt_set_id: 'stable-v1',
        ignore_active_job_target: false,
      }),
      expect.any(Function),
    );
  });

  it('disables submit when the selected job target is stale', async () => {
    apiMocks.listJobTargets.mockResolvedValue([
      buildJobTarget({
        id: 'jt_stale',
        latest_analysis_status: 'stale',
      }),
    ]);

    const wrapper = await mountTrainView('/train?job_target_id=jt_stale');
    await flushPromises();

    const submitButton = wrapper.get('button[type="submit"]');
    expect(submitButton.attributes('disabled')).toBeDefined();
    expect(apiMocks.createSessionStream).not.toHaveBeenCalled();
  });

  it('falls back to generic mode when the active job target is not ready', async () => {
    apiMocks.getDashboard.mockResolvedValue(
      buildDashboard({
        recommendation_scope: 'generic',
        active_job_target: {
          id: 'jt_fallback',
          title: '平台工程师',
          latest_analysis_status: 'stale',
        },
      }),
    );
    apiMocks.listJobTargets.mockResolvedValue([
      buildJobTarget({
        id: 'jt_fallback',
        title: '平台工程师',
        latest_analysis_status: 'stale',
      }),
    ]);

    const wrapper = await mountTrainView();
    await flushPromises();

    await wrapper.get('form').trigger('submit.prevent');
    await flushPromises();

    expect(apiMocks.createSessionStream).toHaveBeenCalledWith(
      expect.objectContaining({
        job_target_id: undefined,
        ignore_active_job_target: true,
      }),
      expect.any(Function),
    );
  });
});
