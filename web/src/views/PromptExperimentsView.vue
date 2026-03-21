<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-blue)]">
      <p class="neo-kicker bg-white">
        {{ t('promptExperiments.hero.kicker') }}
      </p>
      <h1 class="text-xl font-black md:text-2xl">
        {{ t('promptExperiments.hero.title') }}
      </h1>
      <p class="neo-note mt-2">
        {{ t('promptExperiments.hero.description') }}
      </p>
    </header>

    <NoticePanel
      v-if="promptSetsError"
      tone="error"
      :title="t('promptExperiments.loadErrorTitle')"
      :message="promptSetsError"
    />

    <div class="neo-panel space-y-4">
      <div class="neo-grid md:grid-cols-2">
        <label class="space-y-2">
          <span class="neo-subheading">{{
            t('promptExperiments.filters.left')
          }}</span>
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
          <span class="neo-subheading">{{
            t('promptExperiments.filters.right')
          }}</span>
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
          <span class="neo-subheading">{{
            t('promptExperiments.filters.mode')
          }}</span>
          <select v-model="filters.mode" class="neo-select">
            <option value="">{{ t('promptExperiments.filters.allModes') }}</option>
            <option value="basics">{{ formatModeLabel(t, 'basics') }}</option>
            <option value="project">{{ formatModeLabel(t, 'project') }}</option>
          </select>
        </label>
        <label class="space-y-2">
          <span class="neo-subheading">{{
            t('promptExperiments.filters.topic')
          }}</span>
          <select v-model="filters.topic" class="neo-select">
            <option value="">{{ t('promptExperiments.filters.allTopics') }}</option>
            <option v-for="topic in availableTopics" :key="topic" :value="topic">
              {{ formatTopicLabel(t, topic) }}
            </option>
          </select>
        </label>
      </div>
    </div>

    <NoticePanel
      v-if="comparisonError"
      tone="error"
      :title="t('promptExperiments.compareErrorTitle')"
      :message="comparisonError"
    />

    <div v-if="isLoadingComparison" class="neo-panel bg-white">
      <p class="neo-note">{{ t('common.loading') }}</p>
    </div>

    <div
      v-else-if="report"
      class="neo-grid lg:grid-cols-2"
    >
      <article
        v-for="item in [report.left, report.right]"
        :key="item.prompt_set.id"
        class="neo-panel space-y-4"
      >
        <div class="space-y-2">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ item.prompt_set.status }}
          </p>
          <h2 class="text-xl font-black">{{ item.prompt_set.label }}</h2>
          <p class="neo-note">{{ item.prompt_set.description }}</p>
        </div>
        <div class="neo-grid md:grid-cols-2">
          <div class="border-2 border-black bg-white px-4 py-3 md:border-4">
            <p class="neo-note">
              {{ t('promptExperiments.metrics.sessionCount') }}
            </p>
            <p class="text-lg font-black">{{ item.session_count }}</p>
          </div>
          <div class="border-2 border-black bg-white px-4 py-3 md:border-4">
            <p class="neo-note">
              {{ t('promptExperiments.metrics.completedCount') }}
            </p>
            <p class="text-lg font-black">{{ item.completed_count }}</p>
          </div>
          <div class="border-2 border-black bg-white px-4 py-3 md:border-4">
            <p class="neo-note">
              {{ t('promptExperiments.metrics.avgTotalScore') }}
            </p>
            <p class="text-lg font-black">
              {{ item.avg_total_score.toFixed(1) }}
            </p>
          </div>
          <div class="border-2 border-black bg-white px-4 py-3 md:border-4">
            <p class="neo-note">
              {{ t('promptExperiments.metrics.avgQuestionLatency') }}
            </p>
            <p class="text-lg font-black">
              {{ item.avg_generate_question_latency_ms.toFixed(1) }} ms
            </p>
          </div>
          <div class="border-2 border-black bg-white px-4 py-3 md:border-4">
            <p class="neo-note">
              {{ t('promptExperiments.metrics.avgAnswerLatency') }}
            </p>
            <p class="text-lg font-black">
              {{ item.avg_evaluate_answer_latency_ms.toFixed(1) }} ms
            </p>
          </div>
          <div class="border-2 border-black bg-white px-4 py-3 md:border-4">
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

    <section class="neo-panel space-y-4">
      <div class="flex items-center justify-between gap-3">
        <div>
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('promptExperiments.samples.kicker') }}
          </p>
          <h2 class="text-xl font-black">
            {{ t('promptExperiments.samples.title') }}
          </h2>
        </div>
        <span class="neo-note">
          {{ report?.recent_samples.length ?? 0 }}
        </span>
      </div>

      <div v-if="!report?.recent_samples.length" class="neo-note">
        {{ t('promptExperiments.samples.empty') }}
      </div>

      <div v-else class="space-y-3">
        <article
          v-for="sample in report.recent_samples"
          :key="sample.session_id"
          class="border-2 border-black bg-white p-4 md:border-4"
        >
          <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div class="space-y-2">
              <p class="text-base font-black">
                {{ formatModeLabel(t, sample.mode) }}
                <template v-if="sample.topic">
                  · {{ formatTopicLabel(t, sample.topic) }}
                </template>
              </p>
              <p class="neo-note">
                {{ sample.prompt_set.label }} · {{ formatStatusLabel(t, sample.status) }}
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

          <div v-if="expandedSessionId === sample.session_id" class="mt-4 space-y-3">
            <div v-if="isLoadingLogs" class="neo-note">
              {{ t('common.loading') }}
            </div>
            <NoticePanel
              v-else-if="logsError"
              tone="error"
              :title="t('promptExperiments.logs.errorTitle')"
              :message="logsError"
            />
            <div v-else-if="sampleLogs.length" class="space-y-3">
              <div
                v-for="log in sampleLogs"
                :key="log.id"
                class="border-2 border-black bg-[var(--neo-paper)] px-4 py-3 md:border-4"
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
  </section>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute } from 'vue-router';

import {
  getPromptExperiment,
  listPromptSets,
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

const {
  data: promptSets,
  error: promptSetsQueryError,
} = useQuery({
  queryKey: ['prompt-sets'],
  queryFn: listPromptSets,
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

const comparisonEnabled = computed(
  () => Boolean(filters.left && filters.right && filters.left !== filters.right),
);

const {
  data: report,
  error: comparisonQueryError,
  isLoading: isLoadingComparison,
} = useQuery({
  queryKey: ['prompt-experiment', filters.left, filters.right, filters.mode, filters.topic],
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
  promptSetsQueryError.value instanceof Error ? promptSetsQueryError.value.message : '',
);
const comparisonError = computed(() =>
  comparisonQueryError.value instanceof Error ? comparisonQueryError.value.message : '',
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
