<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-blue)]">
      <p class="neo-kicker bg-white">{{ t('jobs.hero.kicker') }}</p>
      <p class="text-base font-semibold">
        {{ t('jobs.hero.description') }}
      </p>
    </header>

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

    <div class="neo-grid lg:grid-cols-[0.78fr_1.22fr]">
      <div class="neo-panel space-y-4">
        <div class="flex items-center justify-between gap-3">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ t('jobs.listTitle') }}
          </p>
          <button type="button" class="neo-button-red" @click="startCreate">
            {{ t('jobs.createAction') }}
          </button>
        </div>

        <div v-if="jobTargets.length" class="neo-stagger-list space-y-3">
          <button
            v-for="target in jobTargets"
            :key="target.id"
            type="button"
            class="w-full border-2 border-black bg-white px-4 py-3 text-left shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] md:border-4"
            :class="{
              'bg-[var(--neo-yellow)]': selectedJobTargetId === target.id,
            }"
            @click="selectTarget(target.id)"
          >
            <div class="flex items-center justify-between gap-3">
              <div class="space-y-1">
                <p class="text-sm font-black uppercase tracking-[0.08em]">
                  {{ target.title }}
                </p>
                <p
                  v-if="target.company_name"
                  class="break-all text-sm font-semibold"
                >
                  {{ target.company_name }}
                </p>
              </div>
              <span class="neo-badge bg-[var(--neo-green)]">
                {{
                  formatJobTargetAnalysisStatusLabel(
                    t,
                    target.latest_analysis_status,
                  )
                }}
              </span>
            </div>
            <p
              v-if="activeJobTargetId === target.id"
              class="mt-2 text-xs font-black uppercase tracking-[0.08em]"
            >
              {{ t('jobs.activeBadge') }}
            </p>
            <p class="mt-2 break-all text-xs font-semibold text-black/80">
              {{
                t('common.lastUpdated', {
                  value: formatDateTime(target.updated_at),
                })
              }}
            </p>
          </button>
        </div>
        <p v-else class="neo-note">{{ t('jobs.emptyList') }}</p>
      </div>

      <div v-if="showEditor" class="space-y-6">
        <div class="neo-panel space-y-4">
          <div class="flex items-center justify-between gap-3">
            <p class="neo-kicker bg-[var(--neo-red)]">
              {{
                selectedJobTarget
                  ? t('jobs.editorTitle')
                  : t('jobs.createTitle')
              }}
            </p>
            <div
              v-if="selectedJobTarget"
              class="flex flex-col gap-3 sm:flex-row sm:flex-wrap"
            >
              <button
                type="button"
                class="neo-button w-full bg-white sm:w-auto"
                :disabled="isActivating"
                @click="toggleActiveJobTarget"
              >
                {{
                  isActiveSelection
                    ? t('jobs.clearActiveAction')
                    : t('jobs.activateAction')
                }}
              </button>
              <button
                type="button"
                class="neo-button-dark w-full sm:w-auto"
                :disabled="isAnalyzing"
                @click="runAnalysis"
              >
                {{
                  isAnalyzing
                    ? t('jobs.analyzing')
                    : selectedLatestAnalysis
                      ? t('jobs.reanalyzeAction')
                      : t('jobs.analyzeAction')
                }}
              </button>
            </div>
          </div>

          <p v-if="selectedJobTarget && isActiveSelection" class="neo-note">
            {{ activeSelectionDescription }}
          </p>

          <form class="space-y-4" @submit.prevent="submit">
            <label class="space-y-2">
              <span class="neo-subheading">{{ t('jobs.fields.title') }}</span>
              <input
                v-model="editor.title"
                class="neo-input"
                :placeholder="t('jobs.placeholders.title')"
              />
            </label>

            <label class="space-y-2">
              <span class="neo-subheading">
                {{ t('jobs.fields.companyName') }}
              </span>
              <input
                v-model="editor.company_name"
                class="neo-input"
                :placeholder="t('jobs.placeholders.companyName')"
              />
            </label>

            <label class="space-y-2">
              <span class="neo-subheading">
                {{ t('jobs.fields.sourceText') }}
              </span>
              <textarea
                v-model="editor.source_text"
                class="neo-textarea min-h-[240px]"
                :placeholder="t('jobs.placeholders.sourceText')"
              />
            </label>

            <button
              type="submit"
              class="neo-button-dark w-full sm:w-auto"
              :disabled="isSaving || isCreating"
            >
              {{
                isSaving || isCreating
                  ? t('common.saving')
                  : t('jobs.saveAction')
              }}
            </button>
          </form>
        </div>

        <div v-if="selectedJobTarget" class="neo-panel space-y-4">
          <p class="neo-kicker bg-[var(--neo-blue)]">
            {{ t('jobs.readinessTitle') }}
          </p>
          <div
            class="space-y-3 border-2 border-black bg-white px-4 py-4 md:border-4"
          >
            <p class="text-base font-black">
              {{
                formatJobTargetAnalysisStatusLabel(
                  t,
                  selectedJobTarget.latest_analysis_status,
                )
              }}
            </p>
            <p class="neo-note">{{ selectedJobTargetReadinessDescription }}</p>
          </div>
        </div>

        <div class="neo-panel space-y-4">
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('jobs.latestAnalysisTitle') }}
          </p>
          <template v-if="selectedLatestAnalysis">
            <div
              class="space-y-4 border-2 border-black bg-white px-4 py-4 md:border-4"
            >
              <p class="neo-note">{{ latestSnapshotDescription }}</p>

              <div class="space-y-2">
                <p class="neo-subheading">{{ t('jobs.fields.summary') }}</p>
                <p class="neo-note">{{ selectedLatestAnalysis.summary }}</p>
              </div>

              <div class="space-y-2">
                <p class="neo-subheading">
                  {{ t('jobs.fields.mustHaveSkills') }}
                </p>
                <ul class="space-y-2">
                  <li
                    v-for="item in selectedLatestAnalysis.must_have_skills"
                    :key="item"
                    class="neo-note"
                  >
                    {{ item }}
                  </li>
                </ul>
              </div>

              <div
                v-if="selectedLatestAnalysis.bonus_skills.length"
                class="space-y-2"
              >
                <p class="neo-subheading">{{ t('jobs.fields.bonusSkills') }}</p>
                <ul class="space-y-2">
                  <li
                    v-for="item in selectedLatestAnalysis.bonus_skills"
                    :key="item"
                    class="neo-note"
                  >
                    {{ item }}
                  </li>
                </ul>
              </div>

              <div class="space-y-2">
                <p class="neo-subheading">
                  {{ t('jobs.fields.responsibilities') }}
                </p>
                <ul class="space-y-2">
                  <li
                    v-for="item in selectedLatestAnalysis.responsibilities"
                    :key="item"
                    class="neo-note"
                  >
                    {{ item }}
                  </li>
                </ul>
              </div>

              <div class="space-y-2">
                <p class="neo-subheading">
                  {{ t('jobs.fields.evaluationFocus') }}
                </p>
                <ul class="space-y-2">
                  <li
                    v-for="item in selectedLatestAnalysis.evaluation_focus"
                    :key="item"
                    class="neo-note"
                  >
                    {{ item }}
                  </li>
                </ul>
              </div>
            </div>
          </template>
          <p v-else class="neo-note">{{ t('jobs.noLatestAnalysis') }}</p>
        </div>

        <div class="neo-panel space-y-4">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ t('jobs.historyTitle') }}
          </p>
          <div v-if="analysisRuns.length" class="neo-stagger-list space-y-3">
            <article
              v-for="run in analysisRuns"
              :key="run.id"
              class="space-y-2 border-2 border-black bg-white px-4 py-4 md:border-4"
            >
              <div class="flex items-center justify-between gap-3">
                <p class="text-sm font-black uppercase tracking-[0.08em]">
                  {{
                    t('jobs.runStatus', {
                      status: formatJobTargetAnalysisStatusLabel(t, run.status),
                    })
                  }}
                </p>
                <span class="neo-badge bg-[var(--neo-yellow)]">
                  {{ formatDateTime(run.created_at) }}
                </span>
              </div>
              <p v-if="run.error_message" class="neo-note">
                {{ t('jobs.runError', { message: run.error_message }) }}
              </p>
              <p
                v-else-if="run.summary"
                class="break-all text-sm font-semibold text-black/80"
              >
                {{ run.summary }}
              </p>
            </article>
          </div>
          <p v-else class="neo-note">{{ t('jobs.historyEmpty') }}</p>
        </div>
      </div>

      <div v-else class="neo-panel space-y-3">
        <p class="neo-kicker bg-[var(--neo-yellow)]">
          {{ t('jobs.noSelectionTitle') }}
        </p>
        <p class="text-sm font-semibold">
          {{ t('jobs.noSelectionDescription') }}
        </p>
      </div>
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
import NoticePanel from '../components/NoticePanel.vue';
import { describeJobTargetStatus } from '../lib/jobTargetStatus';
import { formatJobTargetAnalysisStatusLabel } from '../lib/labels';

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
const activeJobTargetId = computed(
  () => profileData.value?.active_job_target_id ?? '',
);

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
