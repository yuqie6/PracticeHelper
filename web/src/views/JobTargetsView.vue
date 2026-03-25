<template>
  <section class="neo-page jobs-page space-y-6 xl:space-y-8">
    <JobTargetsStageHero :stats="stageStats" @create="startCreate" />

    <NoticePanel
      v-if="createError"
      tone="error"
      :title="t('jobs.createErrorTitle')"
      :message="createError"
    />
    <NoticePanel
      v-if="saveError"
      tone="error"
      :title="t('jobs.saveErrorTitle')"
      :message="saveError"
    />
    <NoticePanel
      v-if="analyzeError"
      tone="error"
      :title="t('jobs.analyzeErrorTitle')"
      :message="analyzeError"
    />
    <NoticePanel
      v-if="activeError"
      tone="error"
      :title="t('jobs.activeErrorTitle')"
      :message="activeError"
    />

    <div class="jobs-shell">
      <aside class="jobs-list-wrap">
        <JobTargetsListPanel
          :job-targets="jobTargets"
          :selected-job-target-id="selectedJobTargetId"
          :active-job-target-id="activeJobTargetId"
          :format-date-time="formatDateTime"
          @select="selectTarget"
        />
      </aside>

      <JobTargetInsightsPanel
        :show-editor="showEditor"
        :selected-job-target="selectedJobTarget"
        :selected-latest-analysis="selectedLatestAnalysis"
        :editor="editor"
        :is-active-selection="isActiveSelection"
        :is-creating="isCreating"
        :is-saving="isSaving"
        :is-analyzing="isAnalyzing"
        :is-activating="isActivating"
        :active-selection-description="activeSelectionDescription"
        :selected-job-target-readiness-description="selectedJobTargetReadinessDescription"
        :latest-snapshot-description="latestSnapshotDescription"
        :analysis-runs="analysisRuns"
        :format-date-time="formatDateTime"
        @submit="submit"
        @run-analysis="runAnalysis"
        @toggle-active="toggleActiveJobTarget"
        @update:field="updateEditorField"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import {
  activateJobTarget,
  analyzeJobTarget,
  clearActiveJobTarget,
  createJobTarget,
  getJobTarget,
  getProfile,
  listJobTargetAnalysisRuns,
  listJobTargets,
  updateJobTarget,
} from '../api/client';
import JobTargetInsightsPanel from '../components/JobTargetInsightsPanel.vue';
import JobTargetsListPanel from '../components/JobTargetsListPanel.vue';
import JobTargetsStageHero from '../components/JobTargetsStageHero.vue';
import NoticePanel from '../components/NoticePanel.vue';
import { describeJobTargetStatus } from '../lib/jobTargetStatus';

const queryClient = useQueryClient();
const { t } = useI18n();

const selectedJobTargetId = ref('');
const isCreatingNew = ref(false);
const createError = ref('');
const saveError = ref('');
const analyzeError = ref('');
const activeError = ref('');

const editor = reactive({
  title: '',
  company_name: '',
  source_text: '',
});

const { data: jobTargetsData } = useQuery({
  queryKey: ['job-targets'],
  queryFn: listJobTargets,
});

const { data: profileData } = useQuery({
  queryKey: ['profile'],
  queryFn: getProfile,
});

const jobTargets = computed(() => jobTargetsData.value ?? []);
const readyJobTargetCount = computed(
  () =>
    jobTargets.value.filter(
      (target) => target.latest_analysis_status === 'succeeded',
    ).length,
);
const activeJobTargetId = computed(
  () => profileData.value?.active_job_target_id ?? '',
);
const stageStats = computed(() => [
  { value: jobTargets.value.length, label: t('jobs.listTitle') },
  { value: readyJobTargetCount.value, label: t('jobs.readinessTitle') },
  { value: analysisRuns.value.length, label: t('jobs.historyTitle') },
]);

const { data: selectedJobTargetData } = useQuery({
  queryKey: ['job-targets', selectedJobTargetId],
  queryFn: () => getJobTarget(selectedJobTargetId.value),
  enabled: computed(() => Boolean(selectedJobTargetId.value)),
});

const { data: analysisRunsData } = useQuery({
  queryKey: ['job-target-analysis-runs', selectedJobTargetId],
  queryFn: () => listJobTargetAnalysisRuns(selectedJobTargetId.value),
  enabled: computed(() => Boolean(selectedJobTargetId.value)),
});

const selectedJobTarget = computed(() => selectedJobTargetData.value ?? null);
const selectedLatestAnalysis = computed(
  () => selectedJobTarget.value?.latest_successful_analysis ?? null,
);
const isActiveSelection = computed(
  () =>
    Boolean(selectedJobTarget.value) &&
    activeJobTargetId.value === selectedJobTarget.value?.id,
);
const analysisRuns = computed(() => analysisRunsData.value ?? []);
const showEditor = computed(
  () => Boolean(selectedJobTarget.value) || isCreatingNew.value,
);
const selectedJobTargetReadinessDescription = computed(() => {
  if (!selectedJobTarget.value) {
    return '';
  }
  return describeJobTargetStatus(
    t,
    'jobsReadiness',
    selectedJobTarget.value.latest_analysis_status,
  );
});
const latestSnapshotDescription = computed(() => {
  if (!selectedJobTarget.value || !selectedLatestAnalysis.value) {
    return '';
  }
  return describeJobTargetStatus(
    t,
    'jobsSnapshot',
    selectedJobTarget.value.latest_analysis_status,
  );
});
const activeSelectionDescription = computed(() =>
  isActiveSelection.value
    ? selectedJobTarget.value?.latest_analysis_status === 'succeeded'
      ? t('jobs.activeReadyDescription')
      : t('jobs.activeNotReadyDescription')
    : '',
);

watch(
  jobTargets,
  (items) => {
    if (items.length === 0) {
      isCreatingNew.value = true;
      return;
    }

    if (!selectedJobTargetId.value && !isCreatingNew.value) {
      selectedJobTargetId.value = items[0].id;
    }
  },
  { immediate: true },
);

watch(selectedJobTarget, (target) => {
  if (!target) {
    return;
  }
  isCreatingNew.value = false;
  editor.title = target.title;
  editor.company_name = target.company_name ?? '';
  editor.source_text = target.source_text;
});

const createMutation = useMutation({
  mutationFn: createJobTarget,
  onSuccess: async (target) => {
    createError.value = '';
    isCreatingNew.value = false;
    selectedJobTargetId.value = target.id;
    await queryClient.invalidateQueries({ queryKey: ['job-targets'] });
    await queryClient.invalidateQueries({
      queryKey: ['job-targets', selectedJobTargetId],
    });
  },
  onError: (error) => {
    createError.value =
      error instanceof Error ? error.message : t('common.requestFailed');
  },
});

const updateMutation = useMutation({
  mutationFn: ({
    jobTargetId,
    payload,
  }: {
    jobTargetId: string;
    payload: { title: string; company_name?: string; source_text: string };
  }) => updateJobTarget(jobTargetId, payload),
  onSuccess: async () => {
    saveError.value = '';
    await queryClient.invalidateQueries({ queryKey: ['job-targets'] });
    await queryClient.invalidateQueries({
      queryKey: ['job-targets', selectedJobTargetId],
    });
  },
  onError: (error) => {
    saveError.value =
      error instanceof Error ? error.message : t('common.requestFailed');
  },
});

const analyzeMutation = useMutation({
  mutationFn: (jobTargetId: string) => analyzeJobTarget(jobTargetId),
  onSuccess: async () => {
    analyzeError.value = '';
    await queryClient.invalidateQueries({ queryKey: ['job-targets'] });
    await queryClient.invalidateQueries({
      queryKey: ['job-targets', selectedJobTargetId],
    });
    await queryClient.invalidateQueries({
      queryKey: ['job-target-analysis-runs', selectedJobTargetId],
    });
  },
  onError: (error) => {
    analyzeError.value =
      error instanceof Error ? error.message : t('common.requestFailed');
  },
});

const activateMutation = useMutation({
  mutationFn: (jobTargetId: string) => activateJobTarget(jobTargetId),
  onSuccess: async (profile) => {
    activeError.value = '';
    queryClient.setQueryData(['profile'], profile);
    await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    await queryClient.invalidateQueries({ queryKey: ['job-targets'] });
  },
  onError: (error) => {
    activeError.value =
      error instanceof Error ? error.message : t('common.requestFailed');
  },
});

const clearActiveMutation = useMutation({
  mutationFn: clearActiveJobTarget,
  onSuccess: async (profile) => {
    activeError.value = '';
    queryClient.setQueryData(['profile'], profile);
    await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
  },
  onError: (error) => {
    activeError.value =
      error instanceof Error ? error.message : t('common.requestFailed');
  },
});

const isCreating = computed(() => createMutation.isPending.value);
const isSaving = computed(() => updateMutation.isPending.value);
const isAnalyzing = computed(() => analyzeMutation.isPending.value);
const isActivating = computed(
  () => activateMutation.isPending.value || clearActiveMutation.isPending.value,
);

function updateEditorField(payload: { field: string; value: string }) {
  if (payload.field === 'title') {
    editor.title = payload.value;
    return;
  }
  if (payload.field === 'company_name') {
    editor.company_name = payload.value;
    return;
  }
  if (payload.field === 'source_text') {
    editor.source_text = payload.value;
  }
}

function startCreate() {
  isCreatingNew.value = true;
  selectedJobTargetId.value = '';
  createError.value = '';
  saveError.value = '';
  analyzeError.value = '';
  activeError.value = '';
  editor.title = '';
  editor.company_name = '';
  editor.source_text = '';
}

function selectTarget(jobTargetId: string) {
  selectedJobTargetId.value = jobTargetId;
  createError.value = '';
  saveError.value = '';
  analyzeError.value = '';
  activeError.value = '';
}

function submit() {
  const payload = {
    title: editor.title.trim(),
    company_name: editor.company_name.trim(),
    source_text: editor.source_text.trim(),
  };
  if (!payload.title || !payload.source_text) {
    return;
  }

  if (selectedJobTarget.value) {
    updateMutation.mutate({
      jobTargetId: selectedJobTarget.value.id,
      payload,
    });
    return;
  }

  createMutation.mutate(payload);
}

function runAnalysis() {
  if (!selectedJobTarget.value) {
    return;
  }
  analyzeError.value = '';
  analyzeMutation.mutate(selectedJobTarget.value.id);
}

function toggleActiveJobTarget() {
  if (!selectedJobTarget.value) {
    return;
  }
  activeError.value = '';
  if (isActiveSelection.value) {
    clearActiveMutation.mutate();
    return;
  }
  activateMutation.mutate(selectedJobTarget.value.id);
}

function formatDateTime(value?: string) {
  if (!value) {
    return '';
  }
  return new Date(value).toLocaleString();
}
</script>

<style scoped>
.jobs-page {
  position: relative;
}

.jobs-shell {
  display: grid;
  gap: 1rem;
}

@media (min-width: 1280px) {
  .jobs-shell {
    align-items: start;
    grid-template-columns: minmax(18rem, 21rem) minmax(0, 1fr);
  }

  .jobs-list-wrap {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
