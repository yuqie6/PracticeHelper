<template>
  <section class="neo-page history-page space-y-6 xl:space-y-8">
    <header class="neo-panel-hero history-stage bg-[var(--neo-yellow)]">
      <div class="history-stage-copy">
        <p class="neo-kicker bg-white">{{ t('history.hero.kicker') }}</p>
        <h1 class="history-stage-title">{{ t('history.hero.title') }}</h1>
        <p class="history-stage-note">
          {{
            t('history.batch.description', {
              format: exportFormatLabel,
            })
          }}
        </p>
      </div>

      <div class="history-stage-stats">
        <article class="history-stage-stat">
          <span>{{ sessions.length }}</span>
          <small>{{ t('history.openAction') }}</small>
        </article>
        <article class="history-stage-stat">
          <span>{{ selectedCount }}</span>
          <small>{{ t('history.batch.kicker') }}</small>
        </article>
        <article class="history-stage-stat">
          <span>{{ totalPages }}</span>
          <small>{{ t('history.hero.kicker') }}</small>
        </article>
      </div>
    </header>

    <div class="history-shell">
      <aside class="history-side">
        <section class="neo-panel history-filter-panel">
          <div class="history-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-blue)]">
                {{ t('history.filters.allModes') }}
              </p>
              <h2 class="history-section-title">
                {{ t('history.hero.kicker') }}
              </h2>
            </div>
          </div>

          <div class="history-filter-grid">
            <select v-model="filters.mode" class="neo-select w-full">
              <option value="">{{ t('history.filters.allModes') }}</option>
              <option value="basics">{{ formatModeLabel(t, 'basics') }}</option>
              <option value="project">
                {{ formatModeLabel(t, 'project') }}
              </option>
            </select>
            <select v-model="filters.topic" class="neo-select w-full">
              <option value="">{{ t('history.filters.allTopics') }}</option>
              <option
                v-for="topic in availableTopics"
                :key="topic"
                :value="topic"
              >
                {{ formatTopicLabel(t, topic) }}
              </option>
            </select>
            <select v-model="filters.status" class="neo-select w-full">
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
        </section>

        <section class="neo-panel-soft history-batch-panel">
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

          <label class="space-y-2">
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

          <div class="history-batch-actions">
            <button
              type="button"
              class="neo-button w-full bg-white"
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
              class="neo-button w-full bg-white"
              :disabled="selectedCount === 0"
              @click="clearSelection"
            >
              {{ t('history.batch.clearAllAction') }}
            </button>
            <button
              type="button"
              class="neo-button-dark w-full"
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
        </section>

        <NoticePanel
          v-if="exportError"
          tone="error"
          dismissible
          :title="t('history.exportErrorTitle')"
          :message="exportError"
          @dismiss="exportError = ''"
        />
      </aside>

      <main class="history-main">
        <div v-if="isLoading" class="space-y-3">
          <div v-for="n in 5" :key="n" class="neo-skeleton h-24" />
        </div>

        <section
          v-else-if="!sessions.length"
          class="neo-panel history-empty-panel"
        >
          <p class="neo-note">{{ t('history.empty') }}</p>
        </section>

        <section v-else class="neo-panel history-results-panel">
          <div class="history-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-yellow)]">
                {{ t('history.openAction') }}
              </p>
              <h2 class="history-section-title">
                {{ t('history.hero.title') }}
              </h2>
            </div>
            <span class="neo-badge bg-white">
              {{ currentPage }} / {{ totalPages }}
            </span>
          </div>

          <article v-for="item in sessions" :key="item.id" class="history-row">
            <label class="flex shrink-0 items-center pt-1">
              <input
                class="neo-checkbox"
                type="checkbox"
                :checked="selectedSessionIds.includes(item.id)"
                @change="toggleSelected(item.id)"
              />
            </label>

            <RouterLink :to="resolveSessionLink(item)" class="history-row-link">
              <div class="history-row-top">
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
                      t('history.promptSetBadge', {
                        name: item.prompt_set.label,
                      })
                    }}
                  </span>
                </div>
                <span class="text-sm font-semibold">
                  {{ formatStatusLabel(t, item.status) }}
                </span>
              </div>

              <div class="history-row-middle">
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

              <div class="history-row-bottom">
                <span class="neo-note">
                  {{ new Date(item.updated_at).toLocaleString() }}
                </span>
                <span class="neo-badge bg-[var(--neo-yellow)]">
                  {{ t('history.openAction') }}
                </span>
              </div>
            </RouterLink>
          </article>
        </section>
      </main>
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

<style scoped>
.history-page {
  position: relative;
}

.history-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-yellow) 88%, white) 0%,
    color-mix(in srgb, var(--neo-yellow) 60%, var(--neo-green)) 100%
  );
}

.history-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.history-stage-copy,
.history-stage-stats {
  position: relative;
  z-index: 1;
}

.history-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.history-stage-title {
  font-size: clamp(2.1rem, 6vw, 4.5rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
  max-width: 11ch;
  text-transform: uppercase;
}

.history-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 38rem;
}

.history-stage-stats {
  display: grid;
  gap: 0.75rem;
}

.history-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.history-stage-stat span {
  font-size: clamp(2.4rem, 8vw, 4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.history-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.history-shell {
  display: grid;
  gap: 1rem;
}

.history-side,
.history-main {
  min-width: 0;
}

.history-filter-panel,
.history-results-panel,
.history-empty-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.history-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.history-section-title {
  font-size: 1.35rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.history-filter-grid,
.history-batch-actions {
  display: grid;
  gap: 0.75rem;
}

.history-row {
  border-top: 1px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: grid;
  gap: 1rem;
  grid-template-columns: auto minmax(0, 1fr);
  padding-top: 1rem;
}

.history-row:first-of-type {
  border-top: 0;
  padding-top: 0;
}

.history-row-link {
  display: grid;
  gap: 0.85rem;
  transition: transform 180ms ease;
}

.history-row-link:hover {
  transform: translateX(4px);
}

.history-row-top,
.history-row-middle,
.history-row-bottom {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

@media (min-width: 768px) {
  .history-stage-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .history-row-top,
  .history-row-middle,
  .history-row-bottom {
    align-items: center;
    flex-direction: row;
    justify-content: space-between;
  }
}

@media (min-width: 1280px) {
  .history-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.15fr) minmax(18rem, 0.85fr);
  }

  .history-shell {
    align-items: start;
    grid-template-columns: minmax(18rem, 21rem) minmax(0, 1fr);
  }

  .history-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
