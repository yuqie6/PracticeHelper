<template>
  <section class="neo-page prompt-page space-y-6 xl:space-y-8">
    <header class="neo-panel-hero prompt-stage bg-[var(--neo-blue)]">
      <div class="prompt-stage-copy">
        <p class="neo-kicker bg-white">
          {{ t('promptExperiments.hero.kicker') }}
        </p>
        <h1 class="prompt-stage-title">
          {{ t('promptExperiments.hero.title') }}
        </h1>
        <p class="prompt-stage-note">
          {{ t('promptExperiments.hero.description') }}
        </p>
      </div>

      <div class="prompt-stage-stats">
        <article class="prompt-stage-stat">
          <span>{{ promptSets?.length ?? 0 }}</span>
          <small>{{ t('train.fields.promptSet') }}</small>
        </article>
        <article class="prompt-stage-stat">
          <span>{{ report?.recent_samples.length ?? 0 }}</span>
          <small>{{ t('promptExperiments.samples.title') }}</small>
        </article>
      </div>
    </header>

    <NoticePanel
      v-if="promptSetsError"
      tone="error"
      :title="t('promptExperiments.loadErrorTitle')"
      :message="promptSetsError"
    />

    <NoticePanel
      v-if="comparisonError"
      tone="error"
      :title="t('promptExperiments.compareErrorTitle')"
      :message="comparisonError"
    />

    <div class="prompt-shell">
      <aside class="prompt-side">
        <section class="neo-panel prompt-filter-panel">
          <div class="prompt-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-yellow)]">
                {{ t('promptExperiments.hero.kicker') }}
              </p>
              <h2 class="prompt-section-title">
                {{ t('promptExperiments.hero.title') }}
              </h2>
            </div>
          </div>

          <p class="neo-note">
            {{ t('promptExperiments.overlayNotice') }}
          </p>

          <div class="prompt-filter-grid">
            <label class="space-y-2">
              <span class="neo-subheading">
                {{ t('promptExperiments.filters.left') }}
              </span>
              <select v-model="filters.left" class="neo-select">
                <option
                  v-for="item in promptSets ?? []"
                  :key="item.id"
                  :value="item.id"
                >
                  {{ item.label }}
                </option>
              </select>
            </label>
            <label class="space-y-2">
              <span class="neo-subheading">
                {{ t('promptExperiments.filters.right') }}
              </span>
              <select v-model="filters.right" class="neo-select">
                <option
                  v-for="item in promptSets ?? []"
                  :key="item.id"
                  :value="item.id"
                >
                  {{ item.label }}
                </option>
              </select>
            </label>
            <label class="space-y-2">
              <span class="neo-subheading">
                {{ t('promptExperiments.filters.mode') }}
              </span>
              <select v-model="filters.mode" class="neo-select">
                <option value="">
                  {{ t('promptExperiments.filters.allModes') }}
                </option>
                <option value="basics">
                  {{ formatModeLabel(t, 'basics') }}
                </option>
                <option value="project">
                  {{ formatModeLabel(t, 'project') }}
                </option>
              </select>
            </label>
            <label class="space-y-2">
              <span class="neo-subheading">
                {{ t('promptExperiments.filters.topic') }}
              </span>
              <select v-model="filters.topic" class="neo-select">
                <option value="">
                  {{ t('promptExperiments.filters.allTopics') }}
                </option>
                <option
                  v-for="topic in availableTopics"
                  :key="topic"
                  :value="topic"
                >
                  {{ formatTopicLabel(t, topic) }}
                </option>
              </select>
            </label>
          </div>
        </section>
      </aside>

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
                <p class="neo-kicker bg-[var(--neo-yellow)]">
                  {{ item.prompt_set.status }}
                </p>
                <h2 class="prompt-section-title">
                  {{ item.prompt_set.label }}
                </h2>
              </div>
            </div>

            <p v-if="item.prompt_set.description" class="neo-note">
              {{ item.prompt_set.description }}
            </p>

            <div class="prompt-metric-grid">
              <div class="prompt-metric-card">
                <p class="neo-note">
                  {{ t('promptExperiments.metrics.sessionCount') }}
                </p>
                <p class="text-lg font-black">{{ item.session_count }}</p>
              </div>
              <div class="prompt-metric-card">
                <p class="neo-note">
                  {{ t('promptExperiments.metrics.completedCount') }}
                </p>
                <p class="text-lg font-black">{{ item.completed_count }}</p>
              </div>
              <div class="prompt-metric-card">
                <p class="neo-note">
                  {{ t('promptExperiments.metrics.avgTotalScore') }}
                </p>
                <p class="text-lg font-black">
                  {{ item.avg_total_score.toFixed(1) }}
                </p>
              </div>
              <div class="prompt-metric-card">
                <p class="neo-note">
                  {{ t('promptExperiments.metrics.avgQuestionLatency') }}
                </p>
                <p class="text-lg font-black">
                  {{ item.avg_generate_question_latency_ms.toFixed(1) }} ms
                </p>
              </div>
              <div class="prompt-metric-card">
                <p class="neo-note">
                  {{ t('promptExperiments.metrics.avgAnswerLatency') }}
                </p>
                <p class="text-lg font-black">
                  {{ item.avg_evaluate_answer_latency_ms.toFixed(1) }} ms
                </p>
              </div>
              <div class="prompt-metric-card">
                <p class="neo-note">
                  {{ t('promptExperiments.metrics.avgReviewLatency') }}
                </p>
                <p class="text-lg font-black">
                  {{ item.avg_generate_review_latency_ms.toFixed(1) }} ms
                </p>
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
              <h2 class="prompt-section-title">
                {{ t('promptExperiments.samples.title') }}
              </h2>
            </div>
            <span class="neo-badge bg-white">
              {{ report?.recent_samples.length ?? 0 }}
            </span>
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
                    <template v-if="sample.topic">
                      · {{ formatTopicLabel(t, sample.topic) }}
                    </template>
                  </p>
                  <p class="neo-note">
                    {{ sample.prompt_set.label }} ·
                    {{ formatStatusLabel(t, sample.status) }}
                  </p>
                  <p class="neo-note text-xs">
                    {{ new Date(sample.updated_at).toLocaleString() }}
                  </p>
                </div>
                <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
                  <span class="text-lg font-black">
                    {{ sample.total_score > 0 ? sample.total_score : '—' }}
                  </span>
                  <button
                    type="button"
                    class="neo-button-dark w-full sm:w-auto"
                    @click="toggleSample(sample.session_id)"
                  >
                    {{
                      expandedSessionId === sample.session_id
                        ? t('promptExperiments.samples.hideLogs')
                        : t('promptExperiments.samples.showLogs')
                    }}
                  </button>
                </div>
              </div>

              <div
                v-if="expandedSessionId === sample.session_id"
                class="prompt-sample-logs"
              >
                <div v-if="isLoadingLogs" class="neo-note">
                  {{ t('common.loading') }}
                </div>
                <NoticePanel
                  v-else-if="logsError"
                  tone="error"
                  :title="t('promptExperiments.logs.errorTitle')"
                  :message="logsError"
                />
                <div
                  v-else-if="sampleLogs.length"
                  class="prompt-log-list neo-stagger-list"
                >
                  <div
                    v-for="log in sampleLogs"
                    :key="log.id"
                    class="prompt-log-row"
                  >
                    <p class="font-black">{{ log.flow_name }}</p>
                    <p class="neo-note">
                      {{ t('promptExperiments.logs.modelName') }}:
                      {{ log.model_name || '—' }}
                    </p>
                    <p class="neo-note">
                      {{ t('promptExperiments.logs.promptHash') }}:
                      {{ log.prompt_hash || '—' }}
                    </p>
                    <p class="neo-note">
                      {{ t('promptExperiments.logs.latency') }}:
                      {{ log.latency_ms.toFixed(1) }} ms
                    </p>
                  </div>
                </div>
                <p v-else class="neo-note">
                  {{ t('promptExperiments.logs.empty') }}
                </p>
              </div>
            </article>
          </div>
        </section>
      </main>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import {
  getPromptExperiment,
  listPromptExperimentPromptSets,
  listSessionEvaluationLogs,
} from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import {
  formatModeLabel,
  formatStatusLabel,
  formatTopicLabel,
} from '../lib/labels';
import { resolvePromptExperimentSelection } from '../lib/promptExperiments';

const { t } = useI18n();
const route = useRoute();
const expandedSessionId = ref('');
const filters = reactive({
  left: '',
  right: '',
  mode: '',
  topic: '',
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

const { data: promptSets, error: promptSetsQueryError } = useQuery({
  queryKey: ['prompt-experiment-prompt-sets'],
  queryFn: listPromptExperimentPromptSets,
});

watch(
  [promptSets, () => route.query.left, () => route.query.right],
  ([items, left, right]) => {
    const resolved = resolvePromptExperimentSelection(
      items ?? [],
      typeof left === 'string' ? left : undefined,
      typeof right === 'string' ? right : undefined,
    );
    filters.left = resolved.left;
    filters.right = resolved.right;
  },
  { immediate: true },
);

watch(
  () => route.query,
  () => {
    const mode = route.query.mode;
    const topic = route.query.topic;
    filters.mode = typeof mode === 'string' ? mode : '';
    filters.topic = typeof topic === 'string' ? topic : '';
  },
  { immediate: true },
);

const comparisonEnabled = computed(() =>
  Boolean(filters.left && filters.right && filters.left !== filters.right),
);

const {
  data: report,
  error: comparisonQueryError,
  isLoading: isLoadingComparison,
} = useQuery({
  queryKey: [
    'prompt-experiment',
    filters.left,
    filters.right,
    filters.mode,
    filters.topic,
  ],
  enabled: comparisonEnabled,
  queryFn: () =>
    getPromptExperiment({
      left: filters.left,
      right: filters.right,
      mode: filters.mode || undefined,
      topic: filters.topic || undefined,
      limit: 12,
    }),
});

const {
  data: sampleLogsData,
  error: sampleLogsError,
  isLoading: isLoadingLogs,
} = useQuery({
  queryKey: ['session-evaluation-logs', expandedSessionId],
  enabled: computed(() => Boolean(expandedSessionId.value)),
  queryFn: () => listSessionEvaluationLogs(expandedSessionId.value),
});

const promptSetsError = computed(() =>
  promptSetsQueryError.value instanceof Error
    ? promptSetsQueryError.value.message
    : '',
);
const comparisonError = computed(() =>
  comparisonQueryError.value instanceof Error
    ? comparisonQueryError.value.message
    : '',
);
const logsError = computed(() =>
  sampleLogsError.value instanceof Error ? sampleLogsError.value.message : '',
);
const sampleLogs = computed(() => sampleLogsData.value ?? []);

function toggleSample(sessionId: string) {
  expandedSessionId.value =
    expandedSessionId.value === sessionId ? '' : sessionId;
}
</script>

<style scoped>
.prompt-page {
  position: relative;
}

.prompt-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-blue) 84%, white) 0%,
    color-mix(in srgb, var(--neo-blue) 58%, var(--neo-yellow)) 100%
  );
}

.prompt-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.prompt-stage-copy,
.prompt-stage-stats {
  position: relative;
  z-index: 1;
}

.prompt-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.prompt-stage-title {
  font-size: clamp(2rem, 5vw, 4.2rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 1;
  margin: 0;
  max-width: 12ch;
}

.prompt-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 38rem;
}

.prompt-stage-stats {
  display: grid;
  gap: 0.75rem;
}

.prompt-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.prompt-stage-stat span {
  font-size: clamp(2.2rem, 7vw, 3.8rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.prompt-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.prompt-shell {
  display: grid;
  gap: 1rem;
}

.prompt-filter-panel,
.prompt-compare-panel,
.prompt-samples-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.prompt-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
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

.prompt-filter-grid,
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
  .prompt-stage-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .prompt-filter-grid,
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
  .prompt-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.1fr) minmax(18rem, 0.9fr);
  }

  .prompt-shell {
    align-items: start;
    grid-template-columns: minmax(18rem, 21rem) minmax(0, 1fr);
  }

  .prompt-side {
    position: sticky;
    top: 1.5rem;
  }

  .prompt-compare-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
