<template>
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

    <div v-if="jobTargets.length" class="jobs-list neo-stagger-list">
      <button
        v-for="target in jobTargets"
        :key="target.id"
        type="button"
        class="jobs-list-row"
        :class="{ 'jobs-list-row-active': selectedJobTargetId === target.id }"
        @click="emit('select', target.id)"
      >
        <div class="flex items-start justify-between gap-3">
          <div class="space-y-1">
            <p class="text-sm font-black uppercase tracking-[0.08em]">
              {{ target.title }}
            </p>
            <p v-if="target.company_name" class="break-all text-sm font-semibold">
              {{ target.company_name }}
            </p>
          </div>
          <span class="neo-badge bg-[var(--neo-green)]">
            {{ formatJobTargetAnalysisStatusLabel(t, target.latest_analysis_status) }}
          </span>
        </div>
        <p
          v-if="activeJobTargetId === target.id"
          class="text-xs font-black uppercase tracking-[0.08em]"
        >
          {{ t('jobs.activeBadge') }}
        </p>
        <p class="break-all text-xs font-semibold text-black/80">
          {{ t('common.lastUpdated', { value: formatDateTime(target.updated_at) }) }}
        </p>
      </button>
    </div>
    <p v-else class="neo-note">{{ t('jobs.emptyList') }}</p>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { JobTarget } from '../api/client';
import { formatJobTargetAnalysisStatusLabel } from '../lib/labels';

defineProps<{
  jobTargets: JobTarget[];
  selectedJobTargetId: string;
  activeJobTargetId: string;
  formatDateTime: (value?: string) => string;
}>();

const emit = defineEmits<{
  (event: 'select', id: string): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
.jobs-list-panel {
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

.jobs-list {
  display: grid;
  gap: 0.85rem;
}

.jobs-list-row {
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

.jobs-list-row:hover {
  box-shadow: 8px 8px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(var(--motion-lift-md), var(--motion-lift-md));
}

.jobs-list-row-active {
  background: color-mix(in srgb, var(--neo-yellow) 72%, white);
}

@media (prefers-reduced-motion: reduce) {
  .jobs-list-row {
    transition: none;
  }

  .jobs-list-row:hover {
    box-shadow: inherit;
    transform: none;
  }
}
</style>
