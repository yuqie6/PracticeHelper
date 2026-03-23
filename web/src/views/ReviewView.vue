<template>
  <section class="neo-page review-page space-y-6 xl:space-y-8">
    <ReviewStageHero
      :title="reviewHeroTitle"
      :note="reviewStageNote"
      :title-dense="reviewHeroTitleDense"
      :stats="reviewStageStats"
    />

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
      <div class="review-main">
        <ReviewSummaryMain
          :focus-title="reviewFocusTitle"
          :focus-title-dense="reviewFocusTitleDense"
          :focus-reason="review.top_fix_reason || t('review.topFixFallbackReason')"
          :score-average-display="scoreAverageDisplay"
          :recommended-next-label="recommendedNextLabel"
          :recommended-next-reason="review.recommended_next?.reason || ''"
          :continue-target="continueTarget"
          :highlights="review.highlights"
          :gaps="review.gaps"
          :next-training-focus="review.next_training_focus"
        />

        <ReviewAuditPanel
          v-if="showAuditDetails"
          :retrieval-trace="review.retrieval_trace"
          :retrieval-trace-groups="retrievalTraceGroups"
          :evaluation-logs="evaluationLogs"
          :is-loading-evaluation-logs="isLoadingEvaluationLogs"
          :evaluation-logs-error="evaluationLogsError"
          :format-trace-score="formatTraceScore"
        />
      </div>

      <ReviewSideSummary
        :export-format="exportFormat"
        :export-format-options="exportFormatOptions"
        :export-format-label="exportFormatLabel"
        :is-exporting="isExporting"
        :show-audit-details="showAuditDetails"
        :prompt-set-id="review.prompt_set?.id || ''"
        :prompt-experiment-link="review.prompt_set?.id ? buildPromptExperimentLink(review.prompt_set.id) : ''"
        :job-target-title="review.job_target?.title ?? t('train.genericJobTargetOption')"
        :show-job-target-description="Boolean(review.job_target)"
        :prompt-set-summary="review.prompt_set ? `${review.prompt_set.label} · ${review.prompt_set.status}` : ''"
        :prompt-overlay-summary="reviewPromptOverlaySummary"
        :has-prompt-overlay="hasReviewPromptOverlay"
        :score-breakdown="review.score_breakdown"
        @update:export-format="updateExportFormat"
        @toggle-audit="toggleAuditDetails"
        @export="exportReport"
      />
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
import ReviewAuditPanel from '../components/ReviewAuditPanel.vue';
import ReviewSideSummary from '../components/ReviewSideSummary.vue';
import ReviewStageHero from '../components/ReviewStageHero.vue';
import ReviewSummaryMain from '../components/ReviewSummaryMain.vue';
import {
  SESSION_EXPORT_FORMAT,
  SESSION_EXPORT_FORMATS,
  triggerFileDownload,
  type SessionExportFormat,
} from '../lib/export';
import { formatModeLabel, formatTopicLabel } from '../lib/labels';
import {
  formatPromptOverlaySummary,
  hasPromptOverlay,
} from '../lib/promptOverlay';
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
const reviewStageStats = computed(() => [
  {
    value: scoreAverageDisplay.value,
    label: t('review.scoreBreakdown'),
  },
  {
    value: review.value?.highlights.length ?? 0,
    label: t('review.highlights'),
  },
  {
    value: review.value?.gaps.length ?? 0,
    label: t('review.gaps'),
  },
  {
    value: review.value?.next_training_focus.length ?? 0,
    label: t('review.nextFocus'),
  },
]);
const reviewFocusTitle = computed(
  () => review.value?.top_fix || review.value?.overall || '',
);
const reviewFocusTitleDense = computed(
  () => reviewFocusTitle.value.length > 26,
);
const reviewPromptOverlaySummary = computed(() =>
  formatPromptOverlaySummary(
    t,
    review.value?.prompt_overlay,
    review.value?.prompt_overlay_hash,
  ),
);
const hasReviewPromptOverlay = computed(() =>
  hasPromptOverlay(
    review.value?.prompt_overlay,
    review.value?.prompt_overlay_hash,
  ),
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

function updateExportFormat(value: string) {
  exportFormat.value = value as SessionExportFormat;
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

.review-shell {
  display: grid;
  gap: 1rem;
}

.review-main {
  display: grid;
  gap: 1rem;
}

@media (min-width: 1280px) {
  .review-shell {
    align-items: start;
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 22rem);
  }
}
</style>
