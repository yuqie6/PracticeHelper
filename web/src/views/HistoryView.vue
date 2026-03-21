<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-yellow)]">
      <p class="neo-kicker bg-white">{{ t('history.hero.kicker') }}</p>
      <h1 class="text-xl font-black md:text-2xl">
        {{ t('history.hero.title') }}
      </h1>
    </header>

    <div class="grid gap-3 md:flex md:flex-wrap">
      <select v-model="filters.mode" class="neo-select w-full md:w-auto">
        <option value="">{{ t('history.filters.allModes') }}</option>
        <option value="basics">{{ formatModeLabel(t, 'basics') }}</option>
        <option value="project">{{ formatModeLabel(t, 'project') }}</option>
      </select>
      <select v-model="filters.topic" class="neo-select w-full md:w-auto">
        <option value="">{{ t('history.filters.allTopics') }}</option>
        <option v-for="topic in availableTopics" :key="topic" :value="topic">
          {{ formatTopicLabel(t, topic) }}
        </option>
      </select>
      <select v-model="filters.status" class="neo-select w-full md:w-auto">
        <option value="">{{ t('history.filters.allStatuses') }}</option>
        <option value="completed">
          {{ formatStatusLabel(t, 'completed') }}
        </option>
        <option value="waiting_answer">
          {{ formatStatusLabel(t, 'waiting_answer') }}
        </option>
        <option value="review_pending">
          {{ formatStatusLabel(t, 'review_pending') }}
        </option>
      </select>
    </div>

    <div v-if="isLoading" class="space-y-3">
      <div v-for="n in 5" :key="n" class="neo-skeleton h-24" />
    </div>

    <div v-else-if="!sessions.length" class="neo-panel bg-white">
      <p class="neo-note">{{ t('history.empty') }}</p>
    </div>

    <NoticePanel
      v-if="exportError"
      tone="error"
      dismissible
      :title="t('history.exportErrorTitle')"
      :message="exportError"
      @dismiss="exportError = ''"
    />

    <div v-else class="space-y-3">
      <div class="neo-panel-soft sticky top-4 z-10 space-y-3">
        <div
          class="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between"
        >
          <div class="space-y-2">
            <p class="neo-kicker bg-[var(--neo-green)]">
              {{ t('history.batch.kicker') }}
            </p>
            <p class="text-base font-black">
              {{ t('history.batch.selectedCount', { count: selectedCount }) }}
            </p>
            <p class="neo-note">
              {{
                t('history.batch.description', {
                  format: exportFormatLabel,
                })
              }}
            </p>
          </div>
          <div class="flex flex-col gap-3 lg:items-end">
            <label class="w-full space-y-2 sm:w-44">
              <span class="text-xs font-black uppercase tracking-[0.08em]">
                {{ t('common.exportFormatLabel') }}
              </span>
              <select v-model="exportFormat" class="neo-select">
                <option
                  v-for="item in exportFormatOptions"
                  :key="item.value"
                  :value="item.value"
                >
                  {{ item.label }}
                </option>
              </select>
            </label>
            <div class="flex flex-col gap-3 sm:flex-row">
              <button
                type="button"
                class="neo-button w-full bg-white sm:w-auto"
                @click="toggleSelectAll"
              >
                {{
                  allSelectedOnPage
                    ? t('history.batch.clearPageAction', {
                        count: sessions.length,
                      })
                    : t('history.batch.selectPageAction', {
                        count: sessions.length,
                      })
                }}
              </button>
              <button
                type="button"
                class="neo-button w-full bg-white sm:w-auto"
                :disabled="selectedCount === 0"
                @click="clearSelection"
              >
                {{ t('history.batch.clearAllAction') }}
              </button>
              <button
                type="button"
                class="neo-button-dark w-full sm:w-auto"
                :disabled="selectedCount === 0 || isExporting"
                @click="exportSelected"
              >
                {{
                  isExporting
                    ? t('history.batch.exportingAction')
                    : t('history.batch.exportAction', {
                        count: selectedCount,
                        format: exportFormatLabel,
                      })
                }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <article
        v-for="item in sessions"
        :key="item.id"
        class="neo-panel bg-white"
      >
        <div class="flex items-start gap-3">
          <label class="flex shrink-0 items-center pt-1">
            <input
              class="neo-checkbox"
              type="checkbox"
              :checked="selectedSessionIds.includes(item.id)"
              @change="toggleSelected(item.id)"
            />
          </label>

          <RouterLink
            :to="resolveSessionLink(item)"
            class="block flex-1 space-y-3 transition-transform hover:-translate-y-0.5"
          >
            <div
              class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"
            >
              <div class="space-y-2">
                <span class="block text-base font-black">
                  {{ formatModeLabel(t, item.mode) }}
                  <template v-if="item.topic">
                    · {{ formatTopicLabel(t, item.topic) }}</template
                  >
                  <template v-if="item.project_name">
                    · {{ item.project_name }}</template
                  >
                </span>
                <span
                  v-if="item.prompt_set"
                  class="neo-badge bg-[var(--neo-blue)]"
                >
                  {{
                    t('history.promptSetBadge', { name: item.prompt_set.label })
                  }}
                </span>
              </div>
              <span class="text-sm font-semibold">
                {{ formatStatusLabel(t, item.status) }}
              </span>
            </div>
            <div
              class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"
            >
              <span v-if="item.job_target" class="neo-note">
                {{ item.job_target.title }}
              </span>
              <span v-else class="neo-note">{{
                t('history.noJobTarget')
              }}</span>
              <span class="text-sm font-black">
                {{ item.total_score > 0 ? item.total_score : '—' }}
              </span>
            </div>
            <div
              class="flex flex-col gap-2 text-xs font-semibold sm:flex-row sm:items-center sm:justify-between"
            >
              <span class="neo-note">
                {{ new Date(item.updated_at).toLocaleString() }}
              </span>
              <span class="neo-badge bg-[var(--neo-yellow)]">
                {{ t('history.openAction') }}
              </span>
            </div>
          </RouterLink>
        </div>
      </article>
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
import { useQuery } from '@tanstack/vue-query';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  ApiError,
  downloadSessionBatchExport,
  listSessions,
  type TrainingSessionSummary,
} from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import {
  SESSION_EXPORT_FORMAT,
  SESSION_EXPORT_FORMATS,
  triggerFileDownload,
  type SessionExportFormat,
} from '../lib/export';
import {
  formatModeLabel,
  formatStatusLabel,
  formatTopicLabel,
} from '../lib/labels';
import { useToast } from '../lib/useToast';

const { t } = useI18n();
const { show: showToast } = useToast();
const currentPage = ref(1);
const selectedSessionIds = ref<string[]>([]);
const exportError = ref('');
const isExporting = ref(false);
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
const allSelectedOnPage = computed(
  () =>
    sessions.value.length > 0 &&
    sessions.value.every((item) => selectedSessionIds.value.includes(item.id)),
);
const exportFormatOptions = computed(() =>
  SESSION_EXPORT_FORMATS.map((item) => ({
    value: item,
    label: t(`common.exportFormats.${item}`),
  })),
);
const exportFormatLabel = computed(() =>
  t(`common.exportFormats.${exportFormat.value}`),
);

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

async function exportSelected() {
  if (selectedCount.value === 0 || isExporting.value) {
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
</script>
