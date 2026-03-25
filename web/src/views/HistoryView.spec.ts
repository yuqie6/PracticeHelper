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
  DeleteSessionsResult,
  PaginatedList,
  TrainingSessionSummary,
} from '../api/client';
import HistoryView from './HistoryView.vue';

const apiMocks = vi.hoisted(() => ({
  listSessions: vi.fn(),
  deleteSessions: vi.fn(),
  downloadSessionBatchExport: vi.fn(),
}));

vi.mock('../api/client', async () => {
  const actual =
    await vi.importActual<typeof import('../api/client')>('../api/client');
  return {
    ...actual,
    ...apiMocks,
  };
});

function buildSessionSummary(
  overrides: Partial<TrainingSessionSummary> = {},
): TrainingSessionSummary {
  return {
    id: 'sess_1',
    mode: 'basics',
    topic: 'redis',
    status: 'completed',
    total_score: 84,
    updated_at: '2026-03-25T10:00:00Z',
    ...overrides,
  };
}

function buildPage(
  items: TrainingSessionSummary[],
): PaginatedList<TrainingSessionSummary> {
  return {
    items,
    total: items.length,
    page: 1,
    per_page: 20,
    total_pages: 1,
  };
}

async function mountHistoryView(): Promise<VueWrapper> {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/history', component: HistoryView },
      { path: '/sessions/:id', component: { template: '<div />' } },
      { path: '/reviews/:id', component: { template: '<div />' } },
    ],
  });
  await router.push('/history');
  await router.isReady();

  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return mount(HistoryView, {
    global: {
      plugins: [router, [VueQueryPlugin, { queryClient }], i18n],
      stubs: {
        RouterLink: RouterLinkStub,
      },
    },
  });
}

beforeEach(() => {
  apiMocks.listSessions.mockReset();
  apiMocks.deleteSessions.mockReset();
  apiMocks.downloadSessionBatchExport.mockReset();

  apiMocks.listSessions.mockResolvedValue(
    buildPage([buildSessionSummary(), buildSessionSummary({ id: 'sess_2' })]),
  );
  apiMocks.deleteSessions.mockResolvedValue({
    deleted_count: 1,
    deleted_session_ids: ['sess_1'],
  } satisfies DeleteSessionsResult);
  apiMocks.downloadSessionBatchExport.mockResolvedValue({
    blob: new Blob(['zip']),
    filename: 'sessions.zip',
  });

  vi.stubGlobal(
    'confirm',
    vi.fn(() => true),
  );
});

describe('HistoryView', () => {
  it('deletes a single session after confirmation', async () => {
    const wrapper = await mountHistoryView();
    await flushPromises();

    const deleteButton = wrapper.get('.history-delete-button');
    await deleteButton.trigger('click');
    await flushPromises();

    expect(globalThis.confirm).toHaveBeenCalled();
    expect(apiMocks.deleteSessions).toHaveBeenCalledWith(['sess_1']);
  });

  it('deletes selected sessions in batch and clears the selection', async () => {
    apiMocks.deleteSessions.mockResolvedValue({
      deleted_count: 2,
      deleted_session_ids: ['sess_1', 'sess_2'],
    } satisfies DeleteSessionsResult);

    const wrapper = await mountHistoryView();
    await flushPromises();

    const selectPageButton = wrapper.get('.history-select-page-button');
    await selectPageButton.trigger('click');
    await flushPromises();

    const batchDeleteButton = wrapper.get('.history-batch-delete-button');
    await batchDeleteButton.trigger('click');
    await flushPromises();

    expect(apiMocks.deleteSessions).toHaveBeenCalledWith(['sess_1', 'sess_2']);
    expect(batchDeleteButton.attributes('disabled')).toBeDefined();
  });
});
