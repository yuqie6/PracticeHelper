<template>
  <section class="neo-page history-page space-y-6 xl:space-y-8">
    <HistoryStageHero
      :export-format-label="exportFormatLabel"
      :stats="historyStageStats"
    />

    <div class="history-shell">
      <HistoryFiltersPanel
        :filters="filters"
        :available-topics="availableTopics"
        :selected-count="selectedCount"
        :all-selected-on-page="allSelectedOnPage"
        :page-count="sessions.length"
        :export-format="exportFormat"
        :export-format-label="exportFormatLabel"
        :export-format-options="exportFormatOptions"
        :export-error="exportError"
        :delete-error="deleteError"
        :is-exporting="isExporting"
        :is-deleting="isDeleting"
        @update:filter="updateFilter"
        @update:export-format="updateExportFormat"
        @toggle-select-all="toggleSelectAll"
        @clear-selection="clearSelection"
        @delete="deleteSelected"
        @export="exportSelected"
        @dismiss-export-error="exportError = ''"
        @dismiss-delete-error="deleteError = ''"
      />

      <HistoryResultsPanel
        :is-loading="isLoading"
        :sessions="sessions"
        :current-page="currentPage"
        :total-pages="totalPages"
        :selected-session-ids="selectedSessionIds"
        :deleting-session-ids="deletingSessionIds"
        :resolve-session-link="resolveSessionLink"
        @toggle-selected="toggleSelected"
        @delete="deleteSingle"
      />
    </div>

    <div
      v-if="totalPages > 1"
      class="flex flex-col items-stretch gap-3 sm:flex-row sm:items-center sm:justify-center"
    >
      <button
        class="neo-button-dark w-full sm:w-auto"
        :disabled="currentPage <= 1"
        @click="currentPage--"
      >
        {{ t('history.prev') }}
      </button>
      <span class="text-center text-sm font-semibold">
        {{ currentPage }} / {{ totalPages }}
      </span>
      <button
        class="neo-button-dark w-full sm:w-auto"
        :disabled="currentPage >= totalPages"
        @click="currentPage++"
      >
        {{ t('history.next') }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  ApiError,
  deleteSessions,
  downloadSessionBatchExport,
  listSessions,
  type TrainingSessionSummary,
} from '../api/client';
import HistoryFiltersPanel from '../components/HistoryFiltersPanel.vue';
import HistoryResultsPanel from '../components/HistoryResultsPanel.vue';
import HistoryStageHero from '../components/HistoryStageHero.vue';
import {
  SESSION_EXPORT_FORMAT,
  SESSION_EXPORT_FORMATS,
  triggerFileDownload,
  type SessionExportFormat,
} from '../lib/export';
import { useToast } from '../lib/useToast';

const { t } = useI18n();
const { show: showToast } = useToast();
const queryClient = useQueryClient();
const currentPage = ref(1);
const selectedSessionIds = ref<string[]>([]);
const exportError = ref('');
const deleteError = ref('');
const isExporting = ref(false);
const deletingSessionIds = ref<string[]>([]);
const exportFormat = ref<SessionExportFormat>(SESSION_EXPORT_FORMAT);
const filters = reactive({
  mode: '',
  topic: '',
  status: '',
});

const availableTopics = [
  'mixed',
  'go',
  'redis',
  'kafka',
  'mysql',
  'system_design',
  'distributed',
  'network',
  'microservice',
  'os',
  'docker_k8s',
];

watch(filters, () => {
  currentPage.value = 1;
});

const { data, isLoading } = useQuery({
  queryKey: ['sessions', currentPage, filters],
  queryFn: () =>
    listSessions({
      page: currentPage.value,
      per_page: 20,
      mode: filters.mode || undefined,
      topic: filters.topic || undefined,
      status: filters.status || undefined,
    }),
});

const sessions = computed(() => data.value?.items ?? []);
const totalPages = computed(() => data.value?.total_pages ?? 1);
const selectedCount = computed(() => selectedSessionIds.value.length);
const isDeleting = computed(() => deleteMutation.isPending.value);
const allSelectedOnPage = computed(
  () =>
    sessions.value.length > 0 &&
    sessions.value.every((item) => selectedSessionIds.value.includes(item.id)),
);
const historyStageStats = computed(() => [
  { value: sessions.value.length, label: t('history.openAction') },
  { value: selectedCount.value, label: t('history.batch.kicker') },
  { value: totalPages.value, label: t('history.hero.kicker') },
]);
const exportFormatOptions = computed(() =>
  SESSION_EXPORT_FORMATS.map((item) => ({
    value: item,
    label: t(`common.exportFormats.${item}`),
  })),
);
const exportFormatLabel = computed(() =>
  t(`common.exportFormats.${exportFormat.value}`),
);

watch(totalPages, (value) => {
  if (value > 0 && currentPage.value > value) {
    currentPage.value = value;
  }
});

const deleteMutation = useMutation({
  mutationFn: (sessionIds: string[]) => deleteSessions(sessionIds),
  onMutate: (sessionIds) => {
    deleteError.value = '';
    deletingSessionIds.value = [...sessionIds];
  },
  onSuccess: async (result) => {
    selectedSessionIds.value = selectedSessionIds.value.filter(
      (id) => !result.deleted_session_ids.includes(id),
    );
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ['sessions'] }),
      queryClient.invalidateQueries({ queryKey: ['session'] }),
      queryClient.invalidateQueries({ queryKey: ['review'] }),
      queryClient.invalidateQueries({ queryKey: ['dashboard'] }),
      queryClient.invalidateQueries({ queryKey: ['weaknesses'] }),
      queryClient.invalidateQueries({ queryKey: ['weakness-trends'] }),
      queryClient.invalidateQueries({ queryKey: ['due-reviews'] }),
      queryClient.invalidateQueries({ queryKey: ['prompt-experiment'] }),
      queryClient.invalidateQueries({ queryKey: ['session-evaluation-logs'] }),
      queryClient.invalidateQueries({ queryKey: ['review-evaluation-logs'] }),
    ]);
    showToast(
      t('history.deleteSuccess', { count: result.deleted_count }),
      'success',
    );
  },
  onError: (error) => {
    if (error instanceof ApiError) {
      deleteError.value = error.message;
      return;
    }
    if (error instanceof Error) {
      deleteError.value = error.message;
      return;
    }
    deleteError.value = t('common.requestFailed');
  },
  onSettled: () => {
    deletingSessionIds.value = [];
  },
});

function updateFilter(payload: { field: string; value: string }) {
  if (payload.field === 'mode') {
    filters.mode = payload.value;
    return;
  }
  if (payload.field === 'topic') {
    filters.topic = payload.value;
    return;
  }
  if (payload.field === 'status') {
    filters.status = payload.value;
  }
}

function updateExportFormat(value: string) {
  exportFormat.value = value as SessionExportFormat;
}

function toggleSelected(sessionId: string) {
  if (selectedSessionIds.value.includes(sessionId)) {
    selectedSessionIds.value = selectedSessionIds.value.filter(
      (item) => item !== sessionId,
    );
    return;
  }
  selectedSessionIds.value = [...selectedSessionIds.value, sessionId];
}

function toggleSelectAll() {
  const pageIds = sessions.value.map((item) => item.id);
  if (allSelectedOnPage.value) {
    selectedSessionIds.value = selectedSessionIds.value.filter(
      (id) => !pageIds.includes(id),
    );
    return;
  }
  selectedSessionIds.value = Array.from(
    new Set([...selectedSessionIds.value, ...pageIds]),
  );
}

function clearSelection() {
  selectedSessionIds.value = [];
}

function deleteSelected() {
  confirmDelete(selectedSessionIds.value);
}

function deleteSingle(sessionId: string) {
  confirmDelete([sessionId]);
}

async function exportSelected() {
  if (selectedCount.value === 0 || isExporting.value || isDeleting.value) {
    return;
  }

  exportError.value = '';
  isExporting.value = true;

  try {
    const { blob, filename } = await downloadSessionBatchExport(
      selectedSessionIds.value,
      exportFormat.value,
    );
    triggerFileDownload(blob, filename);
    showToast(t('common.exportSuccess'), 'success');
  } catch (error) {
    if (error instanceof ApiError) {
      exportError.value = error.message;
    } else if (error instanceof Error) {
      exportError.value = error.message;
    } else {
      exportError.value = t('common.requestFailed');
    }
  } finally {
    isExporting.value = false;
  }
}

function resolveSessionLink(item: TrainingSessionSummary) {
  return item.review_id ? `/reviews/${item.review_id}` : `/sessions/${item.id}`;
}

function confirmDelete(sessionIds: string[]) {
  if (sessionIds.length === 0 || isDeleting.value) {
    return;
  }
  const confirmed = window.confirm(
    t('history.deleteConfirm', { count: sessionIds.length }),
  );
  if (!confirmed) {
    return;
  }
  deleteMutation.mutate(sessionIds);
}
</script>

<style scoped>
.history-page {
  position: relative;
}

.history-shell {
  display: grid;
  gap: 1rem;
}

@media (min-width: 1280px) {
  .history-shell {
    align-items: start;
    grid-template-columns: minmax(18rem, 21rem) minmax(0, 1fr);
  }
}
</style>
