<template>
  <section class="neo-page jobs-page space-y-6 xl:space-y-8">
    <header class="neo-panel-hero jobs-stage bg-[var(--neo-blue)]">
      <div class="jobs-stage-copy">
        <p class="neo-kicker bg-white">{{ t('jobs.hero.kicker') }}</p>
        <h1 class="jobs-stage-title">{{ t('app.nav.jobs') }}</h1>
        <p class="jobs-stage-note">{{ t('jobs.hero.description') }}</p>
      </div>

      <div class="jobs-stage-side">
        <article class="jobs-stage-stat">
          <span>{{ jobTargets.length }}</span>
          <small>{{ t('jobs.listTitle') }}</small>
        </article>
        <article class="jobs-stage-stat">
          <span>{{ readyJobTargetCount }}</span>
          <small>{{ t('jobs.readinessTitle') }}</small>
        </article>
        <article class="jobs-stage-stat">
          <span>{{ analysisRuns.length }}</span>
          <small>{{ t('jobs.historyTitle') }}</small>
        </article>
        <button
          type="button"
          class="neo-button-red w-full"
          @click="startCreate"
        >
          {{ t('jobs.createAction') }}
        </button>
      </div>
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

    <div class="jobs-shell">
      <aside class="jobs-list-wrap">
        <section class="neo-panel jobs-list-panel">
          <div class="jobs-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-yellow)]">
                {{ t('jobs.listTitle') }}
              </p>
              <h2 class="jobs-section-title">{{ t('jobs.listTitle') }}</h2>
            </div>
            <span class="neo-badge bg-white">{{ jobTargets.length }}</span>
          </div>

          <div v-if="jobTargets.length" class="jobs-list">
            <button
              v-for="target in jobTargets"
              :key="target.id"
              type="button"
              class="jobs-list-row"
              :class="{
                'jobs-list-row-active': selectedJobTargetId === target.id,
              }"
              @click="selectTarget(target.id)"
            >
              <div class="flex items-start justify-between gap-3">
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
                class="text-xs font-black uppercase tracking-[0.08em]"
              >
                {{ t('jobs.activeBadge') }}
              </p>
              <p class="break-all text-xs font-semibold text-black/80">
                {{
                  t('common.lastUpdated', {
                    value: formatDateTime(target.updated_at),
                  })
                }}
              </p>
            </button>
          </div>
          <p v-else class="neo-note">{{ t('jobs.emptyList') }}</p>
        </section>
      </aside>

      <div v-if="showEditor" class="jobs-detail">
        <section class="neo-panel jobs-editor-panel">
          <div class="jobs-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-red)]">
                {{
                  selectedJobTarget
                    ? t('jobs.editorTitle')
                    : t('jobs.createTitle')
                }}
              </p>
              <h2 class="jobs-section-title">
                {{ selectedJobTarget?.title || t('jobs.createTitle') }}
              </h2>
            </div>
            <div v-if="selectedJobTarget" class="jobs-editor-actions">
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

          <form class="jobs-editor-form" @submit.prevent="submit">
            <div class="jobs-editor-grid">
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
            </div>

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
        </section>

        <div v-if="selectedJobTarget" class="jobs-insight-grid">
          <section class="neo-panel jobs-info-panel">
            <p class="neo-kicker bg-[var(--neo-blue)]">
              {{ t('jobs.readinessTitle') }}
            </p>
            <div class="jobs-readiness-box">
              <p class="text-base font-black">
                {{
                  formatJobTargetAnalysisStatusLabel(
                    t,
                    selectedJobTarget.latest_analysis_status,
                  )
                }}
              </p>
              <p class="neo-note">
                {{ selectedJobTargetReadinessDescription }}
              </p>
            </div>
          </section>

          <section class="neo-panel jobs-info-panel">
            <p class="neo-kicker bg-[var(--neo-green)]">
              {{ t('jobs.latestAnalysisTitle') }}
            </p>
            <AnalysisResultPanel
              v-if="selectedLatestAnalysis"
              :analysis="selectedLatestAnalysis"
              :description="latestSnapshotDescription"
            />
            <p v-else class="neo-note">{{ t('jobs.noLatestAnalysis') }}</p>
          </section>
        </div>

        <section class="neo-panel jobs-history-panel">
          <div class="jobs-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-yellow)]">
                {{ t('jobs.historyTitle') }}
              </p>
              <h2 class="jobs-section-title">{{ t('jobs.historyTitle') }}</h2>
            </div>
            <span class="neo-badge bg-white">{{ analysisRuns.length }}</span>
          </div>

          <div v-if="analysisRuns.length" class="jobs-history-list">
            <article
              v-for="run in analysisRuns"
              :key="run.id"
              class="jobs-history-row"
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
        </section>
      </div>

      <section v-else class="neo-panel jobs-empty-panel">
        <p class="neo-kicker bg-[var(--neo-yellow)]">
          {{ t('jobs.noSelectionTitle') }}
        </p>
        <h2 class="jobs-section-title">{{ t('app.nav.jobs') }}</h2>
        <p class="neo-note">{{ t('jobs.noSelectionDescription') }}</p>
      </section>
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
import AnalysisResultPanel from '../components/AnalysisResultPanel.vue';
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
const readyJobTargetCount = computed(
  () =>
    jobTargets.value.filter(
      (target) => target.latest_analysis_status === 'succeeded',
    ).length,
);
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

<style scoped>
.jobs-page {
  position: relative;
}

.jobs-stage {
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

.jobs-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.jobs-stage-copy,
.jobs-stage-side {
  position: relative;
  z-index: 1;
}

.jobs-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.jobs-stage-title {
  font-size: clamp(2.1rem, 6vw, 4.6rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
  max-width: 10ch;
  text-transform: uppercase;
}

.jobs-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 38rem;
}

.jobs-stage-side {
  display: grid;
  gap: 0.75rem;
}

.jobs-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.jobs-stage-stat span {
  font-size: clamp(2.4rem, 8vw, 4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.jobs-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.jobs-shell {
  display: grid;
  gap: 1rem;
}

.jobs-list-panel,
.jobs-editor-panel,
.jobs-info-panel,
.jobs-history-panel,
.jobs-empty-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.jobs-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.jobs-section-title {
  font-size: 1.35rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.jobs-list,
.jobs-history-list {
  display: grid;
  gap: 0.85rem;
}

.jobs-list-row,
.jobs-history-row,
.jobs-readiness-box {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
  transition:
    transform 180ms ease,
    box-shadow 180ms ease,
    background-color 180ms ease;
}

.jobs-list-row:hover,
.jobs-history-row:hover {
  box-shadow: 8px 8px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(-2px, -2px);
}

.jobs-list-row-active {
  background: color-mix(in srgb, var(--neo-yellow) 72%, white);
}

.jobs-detail {
  display: grid;
  gap: 1rem;
}

.jobs-editor-actions,
.jobs-editor-form,
.jobs-editor-grid {
  display: grid;
  gap: 1rem;
}

.jobs-insight-grid {
  display: grid;
  gap: 1rem;
}

@media (min-width: 768px) {
  .jobs-stage-side {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .jobs-editor-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .jobs-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.15fr) minmax(18rem, 0.85fr);
  }

  .jobs-shell {
    align-items: start;
    grid-template-columns: minmax(18rem, 21rem) minmax(0, 1fr);
  }

  .jobs-list-wrap {
    position: sticky;
    top: 1.5rem;
  }

  .jobs-insight-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (prefers-reduced-motion: reduce) {
  .jobs-list-row,
  .jobs-history-row {
    transition: none;
  }

  .jobs-list-row:hover,
  .jobs-history-row:hover {
    box-shadow: inherit;
    transform: none;
  }
}
</style>
