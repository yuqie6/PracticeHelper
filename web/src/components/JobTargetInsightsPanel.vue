<template>
  <div class="jobs-detail">
    <template v-if="showEditor">
      <JobTargetEditorPanel
        :selected-job-target="selectedJobTarget"
        :selected-latest-analysis="selectedLatestAnalysis"
        :editor="editor"
        :is-active-selection="isActiveSelection"
        :is-creating="isCreating"
        :is-saving="isSaving"
        :is-analyzing="isAnalyzing"
        :is-activating="isActivating"
        :active-selection-description="activeSelectionDescription"
        @submit="emit('submit')"
        @run-analysis="emit('run-analysis')"
        @toggle-active="emit('toggle-active')"
        @update:field="emit('update:field', $event)"
      />

      <div v-if="selectedJobTarget" class="jobs-insight-grid">
        <section class="neo-panel-soft jobs-info-panel jobs-readiness-panel">
          <p class="neo-kicker bg-[var(--neo-blue)]">
            {{ t('jobs.readinessTitle') }}
          </p>
          <div class="jobs-readiness-box">
            <p class="text-base font-black">
              {{ formatJobTargetAnalysisStatusLabel(t, selectedJobTarget.latest_analysis_status) }}
            </p>
            <p class="neo-note">
              {{ selectedJobTargetReadinessDescription }}
            </p>
          </div>
        </section>

        <section class="neo-panel jobs-info-panel jobs-analysis-panel">
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

      <section v-if="selectedJobTarget" class="neo-panel jobs-history-panel">
        <div class="jobs-section-head">
          <div class="space-y-2">
            <p class="neo-kicker bg-[var(--neo-yellow)]">
              {{ t('jobs.historyTitle') }}
            </p>
            <h2 class="jobs-section-title">{{ t('jobs.historyTitle') }}</h2>
          </div>
          <span class="neo-badge bg-white">{{ analysisRuns.length }}</span>
        </div>

        <div v-if="analysisRuns.length" class="jobs-history-list neo-stagger-list">
          <article
            v-for="run in analysisRuns"
            :key="run.id"
            class="jobs-history-row"
          >
            <div class="flex items-center justify-between gap-3">
              <p class="text-sm font-black uppercase tracking-[0.08em]">
                {{ t('jobs.runStatus', { status: formatJobTargetAnalysisStatusLabel(t, run.status) }) }}
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
    </template>

    <section v-else class="neo-panel jobs-empty-panel">
      <p class="neo-kicker bg-[var(--neo-yellow)]">
        {{ t('jobs.noSelectionTitle') }}
      </p>
      <h2 class="jobs-section-title">{{ t('app.nav.jobs') }}</h2>
      <p class="neo-note">{{ t('jobs.noSelectionDescription') }}</p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { JobTarget, JobTargetAnalysisRun } from '../api/client';
import AnalysisResultPanel from './AnalysisResultPanel.vue';
import JobTargetEditorPanel from './JobTargetEditorPanel.vue';
import { formatJobTargetAnalysisStatusLabel } from '../lib/labels';

defineProps<{
  showEditor: boolean;
  selectedJobTarget: JobTarget | null;
  selectedLatestAnalysis: JobTarget['latest_successful_analysis'] | null;
  editor: {
    title: string;
    company_name: string;
    source_text: string;
  };
  isActiveSelection: boolean;
  isCreating: boolean;
  isSaving: boolean;
  isAnalyzing: boolean;
  isActivating: boolean;
  activeSelectionDescription: string;
  selectedJobTargetReadinessDescription: string;
  latestSnapshotDescription: string;
  analysisRuns: JobTargetAnalysisRun[];
  formatDateTime: (value?: string) => string;
}>();

const emit = defineEmits<{
  (event: 'submit'): void;
  (event: 'run-analysis'): void;
  (event: 'toggle-active'): void;
  (event: 'update:field', payload: { field: string; value: string }): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
.jobs-detail {
  display: grid;
  gap: 1rem;
}

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

.jobs-insight-grid {
  display: grid;
  gap: 1rem;
}

.jobs-readiness-panel {
  gap: 1rem;
}

.jobs-analysis-panel {
  min-width: 0;
}

.jobs-history-list {
  display: grid;
  gap: 0.85rem;
}

.jobs-history-row,
.jobs-readiness-box {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
  transition:
    transform var(--motion-duration-base) var(--motion-ease-standard),
    box-shadow var(--motion-duration-base) var(--motion-ease-standard),
    background-color var(--motion-duration-fast) var(--motion-ease-soft);
}

.jobs-history-row:hover {
  box-shadow: 8px 8px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(var(--motion-lift-md), var(--motion-lift-md));
}

.jobs-readiness-box {
  align-items: start;
}

@media (prefers-reduced-motion: reduce) {
  .jobs-history-row {
    transition: none;
  }

  .jobs-history-row:hover {
    box-shadow: inherit;
    transform: none;
  }
}
</style>
