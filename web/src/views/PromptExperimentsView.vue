<template>
  <section class="neo-page prompt-page space-y-6 xl:space-y-8">
    <PromptExperimentsStageHero :stats="promptStageStats" />

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
      <PromptExperimentsFilters
        :prompt-sets="promptSets ?? []"
        :filters="filters"
        :available-topics="availableTopics"
        @update:filter="updateFilter"
      />

      <PromptExperimentsResults
        :report="report"
        :is-loading-comparison="isLoadingComparison"
        :expanded-session-id="expandedSessionId"
        :sample-logs="sampleLogs"
        :is-loading-logs="isLoadingLogs"
        :logs-error="logsError"
        @toggle-sample="toggleSample"
      />
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
import PromptExperimentsFilters from '../components/PromptExperimentsFilters.vue';
import PromptExperimentsResults from '../components/PromptExperimentsResults.vue';
import PromptExperimentsStageHero from '../components/PromptExperimentsStageHero.vue';
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
const promptStageStats = computed(() => [
  { value: promptSets.value?.length ?? 0, label: t('train.fields.promptSet') },
  {
    value: report.value?.recent_samples.length ?? 0,
    label: t('promptExperiments.samples.title'),
  },
]);

function updateFilter(payload: { field: string; value: string }) {
  if (payload.field === 'left') {
    filters.left = payload.value;
    return;
  }
  if (payload.field === 'right') {
    filters.right = payload.value;
    return;
  }
  if (payload.field === 'mode') {
    filters.mode = payload.value;
    return;
  }
  if (payload.field === 'topic') {
    filters.topic = payload.value;
  }
}

function toggleSample(sessionId: string) {
  expandedSessionId.value =
    expandedSessionId.value === sessionId ? '' : sessionId;
}
</script>

<style scoped>
.prompt-page {
  position: relative;
}

.prompt-shell {
  display: grid;
  gap: 1rem;
}

@media (min-width: 1280px) {
  .prompt-shell {
    align-items: start;
    grid-template-columns: minmax(18rem, 21rem) minmax(0, 1fr);
  }
}
</style>
