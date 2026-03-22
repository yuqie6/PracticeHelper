<template>
  <section class="neo-page review-page space-y-6 xl:space-y-8">
    <header class="neo-panel-hero review-stage bg-[var(--neo-blue)]">
      <div class="review-stage-copy">
        <p class="neo-kicker bg-white">{{ t('review.hero.kicker') }}</p>
        <h1
          class="review-stage-title"
          :class="{ 'review-stage-title-dense': reviewHeroTitleDense }"
        >
          {{ reviewHeroTitle }}
        </h1>
        <p class="review-stage-note">
          {{ reviewStageNote }}
        </p>
      </div>

      <div class="review-stage-side">
        <article class="review-stage-stat">
          <span>{{ scoreAverageDisplay }}</span>
          <small>{{ t('review.scoreBreakdown') }}</small>
        </article>
        <article class="review-stage-stat">
          <span>{{ review?.highlights.length ?? 0 }}</span>
          <small>{{ t('review.highlights') }}</small>
        </article>
        <article class="review-stage-stat">
          <span>{{ review?.gaps.length ?? 0 }}</span>
          <small>{{ t('review.gaps') }}</small>
        </article>
        <article class="review-stage-stat">
          <span>{{ review?.next_training_focus.length ?? 0 }}</span>
          <small>{{ t('review.nextFocus') }}</small>
        </article>
      </div>
    </header>

    <NoticePanel
      v-if="loadError"
      tone="error"
      :title="t('review.loadErrorTitle')"
      :message="loadError"
    />
    <button
      v-if="loadError"
      type="button"
      class="neo-button-dark w-full sm:w-auto"
      @click="refetch()"
    >
      {{ t('common.retry') }}
    </button>

    <div v-else-if="isLoading" class="space-y-4">
      <div class="neo-skeleton h-32" />
      <div class="neo-grid lg:grid-cols-[0.9fr_1.1fr]">
        <div class="neo-skeleton h-64" />
        <div class="space-y-4">
          <div class="neo-skeleton h-28" />
          <div class="neo-skeleton h-28" />
        </div>
      </div>
    </div>

    <div v-else-if="!review" class="neo-panel space-y-2">
      <p class="neo-kicker bg-[var(--neo-yellow)]">
        {{ t('review.emptyTitle') }}
      </p>
      <p class="text-sm font-semibold">{{ t('review.emptyDescription') }}</p>
      <RouterLink to="/train" class="neo-button-dark mt-2 w-full sm:w-auto">
        {{ t('review.continueAction') }}
      </RouterLink>
    </div>

    <NoticePanel
      v-if="exportError"
      tone="error"
      :title="t('review.exportErrorTitle')"
      :message="exportError"
    />

    <div v-if="review" class="review-shell">
      <main class="review-main">
        <section class="neo-panel review-focus-panel">
          <div class="review-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-red)]">
                {{ t('review.topFixTitle') }}
              </p>
              <h2
                class="review-focus-title"
                :class="{ 'review-focus-title-dense': reviewFocusTitleDense }"
              >
                {{ reviewFocusTitle }}
              </h2>
            </div>
            <span class="neo-badge bg-white">{{ scoreAverageDisplay }}</span>
          </div>
          <p class="neo-note">
            {{ review.top_fix_reason || t('review.topFixFallbackReason') }}
          </p>
          <div
            v-if="recommendedNextLabel || review.recommended_next?.reason"
            class="review-next-box"
          >
            <p class="neo-subheading">{{ t('review.recommendedNextTitle') }}</p>
            <p class="text-base font-black">
              {{ recommendedNextLabel || t('review.continueAction') }}
            </p>
            <p v-if="review.recommended_next?.reason" class="neo-note">
              {{ review.recommended_next.reason }}
            </p>
            <RouterLink
              :to="continueTarget"
              class="neo-button-red mt-2 w-full sm:w-auto"
            >
              {{ t('review.startRecommendedAction') }}
            </RouterLink>
          </div>
        </section>

        <div class="review-grid neo-stagger-list">
          <section class="neo-panel review-list-panel">
            <p class="neo-kicker bg-[var(--neo-yellow)]">
              {{ t('review.highlights') }}
            </p>
            <ul class="review-list neo-stagger-list">
              <li
                v-for="item in review.highlights"
                :key="item"
                class="neo-note"
              >
                {{ item }}
              </li>
            </ul>
          </section>

          <section class="neo-panel review-list-panel">
            <p class="neo-kicker bg-[var(--neo-red)]">{{ t('review.gaps') }}</p>
            <ul class="review-list neo-stagger-list">
              <li v-for="item in review.gaps" :key="item" class="neo-note">
                {{ item }}
              </li>
            </ul>
          </section>

          <section class="neo-panel review-list-panel">
            <p class="neo-kicker bg-[var(--neo-green)]">
              {{ t('review.nextFocus') }}
            </p>
            <ul class="review-list neo-stagger-list">
              <li
                v-for="item in review.next_training_focus"
                :key="item"
                class="neo-note"
              >
                {{ item }}
              </li>
            </ul>
            <RouterLink
              :to="continueTarget"
              class="neo-button-red mt-4 w-full sm:w-auto"
            >
              {{ t('review.continueAction') }}
            </RouterLink>
          </section>
        </div>

        <section v-if="showAuditDetails" class="neo-panel review-audit-panel">
          <div class="review-section-head">
            <div>
              <p class="neo-kicker bg-[var(--neo-yellow)]">
                {{ t('review.auditTitle') }}
              </p>
              <p class="neo-note">
                {{ t('review.auditDescription') }}
              </p>
            </div>
          </div>

          <div v-if="isLoadingEvaluationLogs" class="neo-note">
            {{ t('common.loading') }}
          </div>

          <NoticePanel
            v-else-if="evaluationLogsError"
            tone="error"
            :title="t('review.auditErrorTitle')"
            :message="evaluationLogsError"
          />

          <section
            v-if="review.retrieval_trace && !isLoadingEvaluationLogs"
            class="review-trace-panel space-y-4"
          >
            <div class="space-y-1">
              <p class="neo-subheading">
                {{ t('review.retrievalTraceTitle') }}
              </p>
              <p class="neo-note">
                {{ t('review.retrievalTraceDescription') }}
              </p>
            </div>

            <div class="review-trace-meta">
              <span class="neo-badge bg-white">
                {{ review.retrieval_trace.topic || '—' }}
              </span>
              <span class="neo-note">
                {{
                  new Date(review.retrieval_trace.generated_at).toLocaleString()
                }}
              </span>
            </div>

            <div class="review-trace-groups">
              <article
                v-for="group in retrievalTraceGroups"
                :key="group.key"
                class="review-audit-row"
              >
                <div class="space-y-1">
                  <p class="text-base font-black">{{ group.title }}</p>
                  <p v-if="group.trace" class="neo-note">
                    {{ group.trace.strategy }} /
                    {{ group.trace.selected_count }} /
                    {{ group.trace.candidate_count }}
                  </p>
                </div>

                <p v-if="group.trace?.query" class="neo-note">
                  {{ group.trace.query }}
                </p>
                <p v-if="group.trace?.fallback_reason" class="neo-note">
                  {{ group.trace.fallback_reason }}
                </p>

                <ul
                  v-if="group.trace?.hits.length"
                  class="review-trace-hit-list neo-stagger-list"
                >
                  <li
                    v-for="(hit, index) in group.trace?.hits"
                    :key="`${group.key}-${hit.ref_id}-${index}`"
                    class="review-trace-hit"
                  >
                    <div class="flex flex-wrap items-center gap-2">
                      <span class="font-black">{{ hit.summary || '—' }}</span>
                      <code v-if="hit.ref_table" class="review-trace-code">
                        {{ hit.ref_table }}/{{ hit.ref_id || '—' }}
                      </code>
                    </div>
                    <p class="neo-note">
                      {{
                        [
                          hit.scope_type || '—',
                          hit.topic || '—',
                          formatTraceScore(hit.final_score),
                        ].join(' / ')
                      }}
                    </p>
                    <p v-if="hit.reason" class="neo-note">{{ hit.reason }}</p>
                  </li>
                </ul>

                <p v-else class="neo-note">
                  {{ t('review.retrievalTraceEmpty') }}
                </p>
              </article>
            </div>
          </section>

          <div
            v-if="evaluationLogs.length"
            class="review-audit-list neo-stagger-list"
          >
            <article
              v-for="item in evaluationLogs"
              :key="item.id"
              class="review-audit-row"
            >
              <div
                class="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between"
              >
                <div class="space-y-1">
                  <p class="text-base font-black">{{ item.flow_name }}</p>
                  <p class="neo-note">
                    {{ t('review.auditMeta') }}: {{ item.model_name || '—' }} /
                    {{ item.prompt_hash || '—' }} /
                    {{ item.latency_ms.toFixed(1) }} ms
                  </p>
                </div>
                <p class="neo-note text-xs">
                  {{ new Date(item.created_at).toLocaleString() }}
                </p>
              </div>

              <div v-if="item.runtime_trace?.entries.length" class="space-y-2">
                <p class="text-xs font-black uppercase tracking-[0.08em]">
                  {{ t('review.runtimeTraceTitle') }}
                </p>
                <RuntimeTraceList :entries="item.runtime_trace.entries" />
              </div>

              <div
                v-if="hasEvaluationRawOutput(item.raw_output)"
                class="space-y-2"
              >
                <p class="text-xs font-black uppercase tracking-[0.08em]">
                  {{ t('review.auditRawOutput') }}
                </p>
                <pre
                  class="max-h-72 overflow-auto border-2 border-black bg-[var(--neo-paper)] px-3 py-3 text-xs leading-6 md:border-4"
                ><code>{{
                  formatEvaluationRawOutput(item.raw_output)
                }}</code></pre>
              </div>
            </article>
          </div>

          <p v-else class="neo-note">
            {{ t('review.auditEmpty') }}
          </p>
        </section>
      </main>

      <aside class="review-side neo-stagger-list">
        <section class="neo-panel review-side-panel">
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
          <div class="review-side-actions">
            <RouterLink
              v-if="review.prompt_set?.id"
              :to="buildPromptExperimentLink(review.prompt_set.id)"
              class="neo-button-dark w-full"
            >
              {{ t('review.promptExperimentAction') }}
            </RouterLink>
            <button
              type="button"
              class="neo-button-dark w-full"
              @click="toggleAuditDetails"
            >
              {{
                showAuditDetails
                  ? t('review.auditHideAction')
                  : t('review.auditShowAction')
              }}
            </button>
            <button
              type="button"
              class="neo-button-dark w-full"
              :disabled="isExporting"
              :aria-busy="isExporting"
              @click="exportReport"
            >
              {{
                isExporting
                  ? t('review.exportingAction')
                  : t('review.exportAction', { format: exportFormatLabel })
              }}
            </button>
          </div>
        </section>

        <section class="neo-panel review-side-panel">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ t('review.jobTargetTitle') }}
          </p>
          <h2 class="review-section-title">
            {{ review.job_target?.title ?? t('train.genericJobTargetOption') }}
          </h2>
          <p v-if="review.job_target" class="neo-note">
            {{ t('review.jobTargetDescription') }}
          </p>
          <p v-if="review.prompt_set" class="neo-note">
            {{ review.prompt_set.label }} · {{ review.prompt_set.status }}
          </p>
        </section>

        <section class="neo-panel review-side-panel">
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('review.scoreBreakdown') }}
          </p>
          <div class="review-score-list neo-stagger-list">
            <div
              v-for="(value, key) in review.score_breakdown"
              :key="key"
              class="review-score-row"
            >
              <div class="flex items-center justify-between gap-3">
                <span class="font-black">{{ key }}</span>
                <span class="neo-badge bg-[var(--neo-yellow)]">{{
                  value
                }}</span>
              </div>
            </div>
          </div>
        </section>
      </aside>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink, useRoute } from 'vue-router';

import {
  ApiError,
  downloadSessionExport,
  getReview,
  listSessionEvaluationLogs,
  type MemoryRetrievalTrace,
} from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import RuntimeTraceList from '../components/RuntimeTraceList.vue';
import {
  formatEvaluationRawOutput,
  hasEvaluationRawOutput,
} from '../lib/evaluationLogs';
import {
  SESSION_EXPORT_FORMAT,
  SESSION_EXPORT_FORMATS,
  triggerFileDownload,
  type SessionExportFormat,
} from '../lib/export';
import { formatModeLabel, formatTopicLabel } from '../lib/labels';
import { buildPromptExperimentLink } from '../lib/promptExperiments';
import { useToast } from '../lib/useToast';

const route = useRoute();
const reviewId = computed(() => route.params.id as string);
const { t } = useI18n();
const { show: showToast } = useToast();
const exportError = ref('');
const isExporting = ref(false);
const exportFormat = ref<SessionExportFormat>(SESSION_EXPORT_FORMAT);
const showAuditDetails = ref(false);

const { data, error, isLoading, refetch } = useQuery({
  queryKey: ['review', reviewId],
  queryFn: () => getReview(reviewId.value),
});

const review = computed(() => data.value ?? null);
const sessionId = computed(() => review.value?.session_id ?? '');
const loadError = computed(() =>
  error.value instanceof Error ? error.value.message : '',
);
const reviewHeaderText = computed(() => {
  if (loadError.value) {
    return t('review.headerError');
  }
  if (isLoading.value) {
    return t('review.hero.loading');
  }
  return t('review.emptyDescription');
});
const reviewHeroTitle = computed(
  () => review.value?.overall ?? reviewHeaderText.value,
);
const reviewStageNote = computed(() => {
  if (review.value) {
    return review.value.top_fix_reason || t('review.topFixFallbackReason');
  }
  return reviewHeaderText.value;
});
const reviewHeroTitleDense = computed(() => reviewHeroTitle.value.length > 26);
const reviewFocusTitle = computed(
  () => review.value?.top_fix || review.value?.overall || '',
);
const reviewFocusTitleDense = computed(
  () => reviewFocusTitle.value.length > 26,
);

const continueTarget = computed(() => {
  const recommended = review.value?.recommended_next;
  if (!recommended) {
    return '/train';
  }

  const params = new URLSearchParams();
  params.set('mode', recommended.mode);
  if (recommended.topic) {
    params.set('topic', recommended.topic);
  }
  if (recommended.project_id) {
    params.set('project_id', recommended.project_id);
  }
  if (review.value?.job_target_id) {
    params.set('job_target_id', review.value.job_target_id);
  }

  const query = params.toString();
  return query ? `/train?${query}` : '/train';
});

const recommendedNextLabel = computed(() => {
  const recommended = review.value?.recommended_next;
  if (!recommended) {
    return '';
  }

  if (recommended.mode === 'basics' && recommended.topic) {
    return t('review.recommendedNextBasics', {
      topic: formatTopicLabel(t, recommended.topic),
    });
  }

  return t('review.recommendedNextMode', {
    mode: formatModeLabel(t, recommended.mode),
  });
});
const scoreAverageDisplay = computed(() => {
  const entries = Object.values(review.value?.score_breakdown ?? {});
  if (!entries.length) {
    return '--';
  }
  const total = entries.reduce((sum, value) => sum + Number(value || 0), 0);
  return (total / entries.length).toFixed(1);
});

const exportFormatOptions = computed(() =>
  SESSION_EXPORT_FORMATS.map((item) => ({
    value: item,
    label: t(`common.exportFormats.${item}`),
  })),
);

const exportFormatLabel = computed(() =>
  t(`common.exportFormats.${exportFormat.value}`),
);
const {
  data: evaluationLogsData,
  error: evaluationLogsQueryError,
  isLoading: isLoadingEvaluationLogs,
} = useQuery({
  queryKey: ['review-evaluation-logs', sessionId],
  enabled: computed(() => showAuditDetails.value && Boolean(sessionId.value)),
  queryFn: () => listSessionEvaluationLogs(sessionId.value),
});
const evaluationLogs = computed(() => evaluationLogsData.value ?? []);
const evaluationLogsError = computed(() =>
  evaluationLogsQueryError.value instanceof Error
    ? evaluationLogsQueryError.value.message
    : '',
);
const retrievalTraceGroups = computed(() => {
  const trace = review.value?.retrieval_trace;
  if (!trace) {
    return [];
  }

  return [
    {
      key: 'observations',
      title: t('review.retrievalTraceObservations'),
      trace: trace.observations ?? null,
    },
    {
      key: 'session-summaries',
      title: t('review.retrievalTraceSummaries'),
      trace: trace.session_summaries ?? null,
    },
  ] as Array<{
    key: string;
    title: string;
    trace: MemoryRetrievalTrace | null;
  }>;
});

async function exportReport() {
  if (!review.value?.session_id || isExporting.value) {
    return;
  }

  exportError.value = '';
  isExporting.value = true;

  try {
    const { blob, filename } = await downloadSessionExport(
      review.value.session_id,
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

function toggleAuditDetails() {
  showAuditDetails.value = !showAuditDetails.value;
}

function formatTraceScore(value?: number): string {
  if (value == null || Number.isNaN(value) || value === 0) {
    return '—';
  }
  return value.toFixed(3);
}
</script>

<style scoped>
.review-page {
  position: relative;
}

.review-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-blue) 84%, white) 0%,
    color-mix(in srgb, var(--neo-blue) 58%, var(--neo-green)) 100%
  );
}

.review-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.review-stage-copy,
.review-stage-side {
  position: relative;
  z-index: 1;
}

.review-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.review-stage-title {
  font-size: clamp(2rem, 5vw, 4.2rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 1;
  margin: 0;
  max-width: 14ch;
  text-wrap: balance;
}

.review-stage-title-dense {
  font-size: clamp(1.45rem, 2.8vw, 2.4rem);
  letter-spacing: -0.03em;
  line-height: 1.25;
  max-width: 28ch;
  text-wrap: pretty;
}

.review-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 40rem;
  text-wrap: pretty;
}

.review-stage-side {
  display: grid;
  gap: 0.75rem;
}

.review-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.review-stage-stat span {
  font-size: clamp(2.2rem, 7vw, 3.8rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.review-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.review-shell {
  display: grid;
  gap: 1rem;
}

.review-main,
.review-side {
  display: grid;
  gap: 1rem;
}

.review-focus-panel,
.review-list-panel,
.review-side-panel,
.review-audit-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.review-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.review-section-title {
  font-size: 1.3rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.review-focus-title {
  font-size: clamp(1.2rem, 2.4vw, 2rem);
  font-weight: 900;
  letter-spacing: -0.03em;
  line-height: 1.12;
  margin: 0;
  max-width: 24ch;
  text-wrap: balance;
}

.review-focus-title-dense {
  font-size: clamp(1.05rem, 2vw, 1.5rem);
  line-height: 1.35;
  max-width: 34ch;
  text-wrap: pretty;
}

.review-next-box,
.review-score-row,
.review-audit-row {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
}

.review-grid,
.review-side-actions,
.review-score-list,
.review-trace-groups,
.review-audit-list {
  display: grid;
  gap: 1rem;
}

.review-trace-panel,
.review-trace-hit-list {
  display: grid;
  gap: 1rem;
}

.review-trace-meta {
  align-items: center;
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
  justify-content: space-between;
}

.review-trace-hit {
  background: color-mix(in srgb, var(--neo-paper) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.5rem;
  padding: 0.9rem 1rem;
}

.review-trace-code {
  background: color-mix(in srgb, var(--neo-surface) 92%, transparent);
  border: 2px solid var(--neo-border);
  font-family: var(--font-mono);
  font-size: 0.75rem;
  font-weight: 800;
  padding: 0.2rem 0.45rem;
}

.review-list {
  display: grid;
  gap: 0.75rem;
  margin: 0;
  padding-left: 1rem;
}

@media (min-width: 768px) {
  .review-stage-side {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .review-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .review-grid > :last-child {
    grid-column: 1 / -1;
  }
}

@media (min-width: 1280px) {
  .review-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.1fr) minmax(18rem, 0.9fr);
  }

  .review-shell {
    align-items: start;
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 22rem);
  }

  .review-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
