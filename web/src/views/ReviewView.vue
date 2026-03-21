<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-blue)]">
      <p class="neo-kicker bg-white">{{ t('review.hero.kicker') }}</p>
      <p class="text-base font-semibold">
        {{ review?.overall ?? reviewHeaderText }}
      </p>
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

    <div v-if="review" class="flex justify-end">
      <div
        class="flex w-full flex-col gap-3 sm:w-auto sm:flex-row sm:items-end"
      >
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
        <RouterLink
          v-if="review.prompt_set?.id"
          :to="buildPromptExperimentLink(review.prompt_set.id)"
          class="neo-button-dark w-full sm:w-auto"
        >
          {{ t('review.promptExperimentAction') }}
        </RouterLink>
        <button
          type="button"
          class="neo-button-dark w-full sm:w-auto"
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
          class="neo-button-dark w-full sm:w-auto"
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
    </div>

    <div v-if="review" class="neo-grid lg:grid-cols-[0.9fr_1.1fr]">
      <div class="neo-panel space-y-4">
        <div class="space-y-4">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ t('review.jobTargetTitle') }}
          </p>
          <div
            class="space-y-3 border-2 border-black bg-white px-4 py-4 md:border-4"
          >
            <p class="text-base font-black">
              {{
                review.job_target?.title ?? t('train.genericJobTargetOption')
              }}
            </p>
            <p v-if="review.job_target" class="neo-note">
              {{ t('review.jobTargetDescription') }}
            </p>
          </div>
        </div>

        <div v-if="review.prompt_set" class="space-y-4">
          <p class="neo-kicker bg-[var(--neo-blue)]">
            {{ t('review.promptSetTitle') }}
          </p>
          <div
            class="space-y-3 border-2 border-black bg-white px-4 py-4 md:border-4"
          >
            <p class="text-base font-black">
              {{ review.prompt_set.label }}
            </p>
            <p class="neo-note">
              {{
                t('review.promptSetDescription', {
                  status: review.prompt_set.status,
                })
              }}
            </p>
          </div>
        </div>

        <div class="space-y-4">
          <p class="neo-kicker bg-[var(--neo-red)]">
            {{ t('review.topFixTitle') }}
          </p>
          <div
            class="space-y-3 border-2 border-black bg-white px-4 py-4 md:border-4"
          >
            <p class="text-lg font-black leading-7">
              {{ review.top_fix || review.overall }}
            </p>
            <p class="neo-note">
              {{ review.top_fix_reason || t('review.topFixFallbackReason') }}
            </p>
          </div>
        </div>

        <div
          v-if="recommendedNextLabel || review.recommended_next?.reason"
          class="space-y-4"
        >
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('review.recommendedNextTitle') }}
          </p>
          <div
            class="space-y-3 border-2 border-black bg-white px-4 py-4 md:border-4"
          >
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
        </div>

        <div class="space-y-4">
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('review.scoreBreakdown') }}
          </p>
          <div class="space-y-3">
            <div
              v-for="(value, key) in review.score_breakdown"
              :key="key"
              class="border-2 border-black bg-white px-4 py-3 md:border-4"
            >
              <div class="flex items-center justify-between gap-3">
                <span class="font-black">{{ key }}</span>
                <span class="neo-badge bg-[var(--neo-yellow)]">{{
                  value
                }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="neo-grid">
        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ t('review.highlights') }}
          </p>
          <ul class="space-y-2">
            <li v-for="item in review.highlights" :key="item" class="neo-note">
              {{ item }}
            </li>
          </ul>
        </div>

        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-red)]">{{ t('review.gaps') }}</p>
          <ul class="space-y-2">
            <li v-for="item in review.gaps" :key="item" class="neo-note">
              {{ item }}
            </li>
          </ul>
        </div>

        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('review.nextFocus') }}
          </p>
          <ul class="space-y-2">
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
        </div>
      </div>
    </div>

    <div v-if="review && showAuditDetails" class="neo-panel space-y-4">
      <div class="flex items-center justify-between gap-3">
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

      <div v-else-if="evaluationLogs.length" class="neo-stagger-list space-y-3">
        <article
          v-for="item in evaluationLogs"
          :key="item.id"
          class="space-y-3 border-2 border-black bg-white px-4 py-4 md:border-4"
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

          <div v-if="hasEvaluationRawOutput(item.raw_output)" class="space-y-2">
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
} from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
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
</script>
