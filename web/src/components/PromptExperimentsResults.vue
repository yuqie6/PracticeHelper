<template>
  <main class="prompt-main">
    <div v-if="isLoadingComparison" class="neo-panel bg-white">
      <p class="neo-note">{{ t('common.loading') }}</p>
    </div>

    <div v-else-if="report" class="prompt-compare-grid neo-stagger-list">
      <article
        v-for="item in [report.left, report.right]"
        :key="item.prompt_set.id"
        class="neo-panel prompt-compare-panel"
      >
        <div class="prompt-section-head">
          <div class="space-y-2">
            <p class="neo-kicker bg-[var(--neo-yellow)]">{{ item.prompt_set.status }}</p>
            <h2 class="prompt-section-title">{{ item.prompt_set.label }}</h2>
          </div>
        </div>

        <p v-if="item.prompt_set.description" class="neo-note">
          {{ item.prompt_set.description }}
        </p>

        <div class="prompt-metric-grid">
          <div class="prompt-metric-card">
            <p class="neo-note">{{ t('promptExperiments.metrics.sessionCount') }}</p>
            <p class="text-lg font-black">{{ item.session_count }}</p>
          </div>
          <div class="prompt-metric-card">
            <p class="neo-note">{{ t('promptExperiments.metrics.completedCount') }}</p>
            <p class="text-lg font-black">{{ item.completed_count }}</p>
          </div>
          <div class="prompt-metric-card">
            <p class="neo-note">{{ t('promptExperiments.metrics.avgTotalScore') }}</p>
            <p class="text-lg font-black">{{ item.avg_total_score.toFixed(1) }}</p>
          </div>
          <div class="prompt-metric-card">
            <p class="neo-note">{{ t('promptExperiments.metrics.avgQuestionLatency') }}</p>
            <p class="text-lg font-black">{{ item.avg_generate_question_latency_ms.toFixed(1) }} ms</p>
          </div>
          <div class="prompt-metric-card">
            <p class="neo-note">{{ t('promptExperiments.metrics.avgAnswerLatency') }}</p>
            <p class="text-lg font-black">{{ item.avg_evaluate_answer_latency_ms.toFixed(1) }} ms</p>
          </div>
          <div class="prompt-metric-card">
            <p class="neo-note">{{ t('promptExperiments.metrics.avgReviewLatency') }}</p>
            <p class="text-lg font-black">{{ item.avg_generate_review_latency_ms.toFixed(1) }} ms</p>
          </div>
        </div>
      </article>
    </div>

    <section class="neo-panel prompt-samples-panel">
      <div class="prompt-section-head">
        <div>
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('promptExperiments.samples.kicker') }}
          </p>
          <h2 class="prompt-section-title">{{ t('promptExperiments.samples.title') }}</h2>
        </div>
        <span class="neo-badge bg-white">{{ report?.recent_samples.length ?? 0 }}</span>
      </div>

      <div v-if="!report?.recent_samples.length" class="neo-note">
        {{ t('promptExperiments.samples.empty') }}
      </div>

      <div v-else class="prompt-sample-list neo-stagger-list">
        <article
          v-for="sample in report.recent_samples"
          :key="sample.session_id"
          class="prompt-sample-row"
        >
          <div class="prompt-sample-top">
            <div class="space-y-2">
              <p class="text-base font-black">
                {{ formatModeLabel(t, sample.mode) }}
                <template v-if="sample.topic"> · {{ formatTopicLabel(t, sample.topic) }}</template>
              </p>
              <p class="neo-note">
                {{ sample.prompt_set.label }} · {{ formatStatusLabel(t, sample.status) }}
              </p>
              <p class="neo-note text-xs">{{ new Date(sample.updated_at).toLocaleString() }}</p>
            </div>
            <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
              <span class="text-lg font-black">{{ sample.total_score > 0 ? sample.total_score : '—' }}</span>
              <button
                type="button"
                class="neo-button-dark w-full sm:w-auto"
                @click="emit('toggle-sample', sample.session_id)"
              >
                {{
                  expandedSessionId === sample.session_id
                    ? t('promptExperiments.samples.hideLogs')
                    : t('promptExperiments.samples.showLogs')
                }}
              </button>
            </div>
          </div>

          <div v-if="expandedSessionId === sample.session_id" class="prompt-sample-logs">
            <div v-if="isLoadingLogs" class="neo-note">{{ t('common.loading') }}</div>
            <NoticePanel
              v-else-if="logsError"
              tone="error"
              :title="t('promptExperiments.logs.errorTitle')"
              :message="logsError"
            />
            <div v-else-if="sampleLogs.length" class="prompt-log-list neo-stagger-list">
              <div v-for="log in sampleLogs" :key="log.id" class="prompt-log-row">
                <p class="font-black">{{ log.flow_name }}</p>
                <p class="neo-note">{{ t('promptExperiments.logs.modelName') }}: {{ log.model_name || '—' }}</p>
                <p class="neo-note">{{ t('promptExperiments.logs.promptHash') }}: {{ log.prompt_hash || '—' }}</p>
                <p class="neo-note">{{ t('promptExperiments.logs.latency') }}: {{ log.latency_ms.toFixed(1) }} ms</p>
              </div>
            </div>
            <p v-else class="neo-note">{{ t('promptExperiments.logs.empty') }}</p>
          </div>
        </article>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import type { EvaluationLogEntry, PromptExperimentReport } from '../api/client';
import NoticePanel from './NoticePanel.vue';
import { formatModeLabel, formatStatusLabel, formatTopicLabel } from '../lib/labels';

defineProps<{
  report: PromptExperimentReport | undefined;
  isLoadingComparison: boolean;
  expandedSessionId: string;
  sampleLogs: EvaluationLogEntry[];
  isLoadingLogs: boolean;
  logsError: string;
}>();

const emit = defineEmits<{
  (event: 'toggle-sample', sessionId: string): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
.prompt-compare-panel,
.prompt-samples-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.prompt-section-head {
  align-items: end;
  border-bottom: 2px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.prompt-section-title {
  font-size: 1.3rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.prompt-compare-grid,
.prompt-metric-grid,
.prompt-sample-list,
.prompt-log-list {
  display: grid;
  gap: 1rem;
}

.prompt-metric-card,
.prompt-sample-row,
.prompt-log-row {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
}

.prompt-sample-top {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.prompt-sample-logs {
  display: grid;
  gap: 0.75rem;
}

@media (min-width: 768px) {
  .prompt-metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .prompt-sample-top {
    align-items: center;
    flex-direction: row;
    justify-content: space-between;
  }
}

@media (min-width: 1280px) {
  .prompt-compare-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
