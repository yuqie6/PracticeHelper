<template>
  <section class="neo-panel projects-jobs-panel">
    <div class="projects-section-head">
      <div class="space-y-2">
        <p class="neo-kicker bg-[var(--neo-blue)]">
          {{ t('projects.jobsTitle') }}
        </p>
        <h2 class="projects-section-title">
          {{ t('projects.jobsTitle') }}
        </h2>
      </div>
      <span class="neo-badge bg-white">
        {{ jobs.length }}
      </span>
    </div>

    <div v-if="jobs.length" class="projects-job-list neo-stagger-list">
      <article
        v-for="job in jobs"
        :key="job.id"
        class="projects-job-row"
      >
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div class="space-y-1">
            <p class="text-sm font-black uppercase tracking-[0.08em]">
              {{ formatImportJobStatusLabel(t, job.status) }}
            </p>
            <p class="text-base font-semibold">{{ job.message }}</p>
          </div>
          <span class="neo-badge bg-[var(--neo-yellow)]">
            {{ formatImportJobStageLabel(t, job.stage) }}
          </span>
        </div>

        <div
          class="space-y-1 break-all text-sm font-semibold text-black/80"
        >
          <p>{{ t('projects.jobRepo') }}: {{ job.repo_url }}</p>
          <p v-if="job.error_message">{{ job.error_message }}</p>
          <p v-if="job.project_name">
            {{ t('projects.jobResult') }}: {{ job.project_name }}
          </p>
        </div>

        <div class="projects-job-actions">
          <button
            v-if="job.project_id"
            type="button"
            class="neo-button-dark w-full"
            @click="emit('open-project', job.project_id)"
          >
            {{ t('projects.openProject') }}
          </button>
          <button
            v-if="job.status === 'failed'"
            type="button"
            class="neo-button-red w-full"
            :disabled="isRetrying"
            @click="emit('retry-job', job.id)"
          >
            {{ isRetrying ? t('common.starting') : t('projects.retryAction') }}
          </button>
        </div>
      </article>
    </div>
    <p v-else class="neo-note">{{ t('projects.jobsEmpty') }}</p>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { ProjectImportJob } from '../api/client';
import {
  formatImportJobStageLabel,
  formatImportJobStatusLabel,
} from '../lib/labels';

defineProps<{
  jobs: ProjectImportJob[];
  isRetrying: boolean;
}>();

const emit = defineEmits<{
  (event: 'open-project', projectId: string): void;
  (event: 'retry-job', jobId: string): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
.projects-jobs-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.projects-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.projects-section-title {
  font-size: 1.4rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.projects-job-list {
  display: grid;
  gap: 0.85rem;
}

.projects-job-row {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.85rem;
  padding: 1rem;
  transition:
    transform var(--motion-duration-base) var(--motion-ease-standard),
    box-shadow var(--motion-duration-base) var(--motion-ease-standard),
    background-color var(--motion-duration-fast) var(--motion-ease-soft);
}

.projects-job-row:hover {
  box-shadow: 8px 8px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(var(--motion-lift-md), var(--motion-lift-md));
}

.projects-job-actions {
  display: grid;
  gap: 0.65rem;
}

@media (min-width: 1280px) {
  .projects-jobs-panel {
    position: sticky;
    top: 1.5rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .projects-job-row {
    transition: none;
  }

  .projects-job-row:hover {
    box-shadow: inherit;
    transform: none;
  }
}
</style>
