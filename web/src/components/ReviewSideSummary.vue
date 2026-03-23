<template>
  <aside class="review-side neo-stagger-list">
    <section class="neo-panel review-side-panel">
      <label class="space-y-2">
        <span class="text-xs font-black uppercase tracking-[0.08em]">
          {{ t('common.exportFormatLabel') }}
        </span>
        <select
          :value="exportFormat"
          class="neo-select"
          @change="emit('update:exportFormat', ($event.target as HTMLSelectElement).value)"
        >
          <option
            v-for="item in exportFormatOptions"
            :key="item.value"
            :value="item.value"
          >
            {{ item.label }}
          </option>
        </select>
      </label>
      <div class="review-side-actions">
        <RouterLink
          v-if="promptSetId"
          :to="promptExperimentLink"
          class="neo-button-dark w-full"
        >
          {{ t('review.promptExperimentAction') }}
        </RouterLink>
        <button
          type="button"
          class="neo-button-dark w-full"
          @click="emit('toggle-audit')"
        >
          {{ showAuditDetails ? t('review.auditHideAction') : t('review.auditShowAction') }}
        </button>
        <button
          type="button"
          class="neo-button-dark w-full"
          :disabled="isExporting"
          :aria-busy="isExporting"
          @click="emit('export')"
        >
          {{
            isExporting
              ? t('review.exportingAction')
              : t('review.exportAction', { format: exportFormatLabel })
          }}
        </button>
      </div>
    </section>

    <section class="neo-panel review-side-panel">
      <p class="neo-kicker bg-[var(--neo-yellow)]">
        {{ t('review.jobTargetTitle') }}
      </p>
      <h2 class="review-section-title">
        {{ jobTargetTitle }}
      </h2>
      <p v-if="showJobTargetDescription" class="neo-note">
        {{ t('review.jobTargetDescription') }}
      </p>
      <p v-if="promptSetSummary" class="neo-note">
        {{ promptSetSummary }}
      </p>
      <p v-if="promptOverlaySummary" class="neo-note">
        {{ t('review.promptOverlaySummary', { summary: promptOverlaySummary }) }}
      </p>
      <p v-if="hasPromptOverlay" class="neo-note">
        {{ t('review.promptOverlayReminder') }}
      </p>
    </section>

    <section class="neo-panel review-side-panel">
      <p class="neo-kicker bg-[var(--neo-green)]">
        {{ t('review.scoreBreakdown') }}
      </p>
      <div class="review-score-list neo-stagger-list">
        <div
          v-for="(value, key) in scoreBreakdown"
          :key="key"
          class="review-score-row"
        >
          <div class="flex items-center justify-between gap-3">
            <span class="font-black">{{ key }}</span>
            <span class="neo-badge bg-[var(--neo-yellow)]">{{ value }}</span>
          </div>
        </div>
      </div>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

defineProps<{
  exportFormat: string;
  exportFormatOptions: Array<{ value: string; label: string }>;
  exportFormatLabel: string;
  isExporting: boolean;
  showAuditDetails: boolean;
  promptSetId: string;
  promptExperimentLink: string;
  jobTargetTitle: string;
  showJobTargetDescription: boolean;
  promptSetSummary: string;
  promptOverlaySummary: string;
  hasPromptOverlay: boolean;
  scoreBreakdown: Record<string, number>;
}>();

const emit = defineEmits<{
  (event: 'update:exportFormat', value: string): void;
  (event: 'toggle-audit'): void;
  (event: 'export'): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
.review-side {
  display: grid;
  gap: 1rem;
}

.review-side-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.review-side-actions,
.review-score-list {
  display: grid;
  gap: 1rem;
}

.review-score-row {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
}

.review-section-title {
  font-size: 1.3rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

@media (min-width: 1280px) {
  .review-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
