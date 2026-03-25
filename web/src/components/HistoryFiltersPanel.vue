<template>
  <aside class="history-side neo-stagger-list">
    <section class="neo-panel history-filter-panel">
      <div class="history-section-head">
        <div class="space-y-2">
          <p class="neo-kicker bg-[var(--neo-blue)]">
            {{ t('history.filters.allModes') }}
          </p>
          <h2 class="history-section-title">{{ t('history.hero.kicker') }}</h2>
        </div>
      </div>

      <div class="history-filter-grid">
        <select
          :value="filters.mode"
          class="neo-select w-full"
          @change="emitFilter('mode', $event)"
        >
          <option value="">{{ t('history.filters.allModes') }}</option>
          <option value="basics">{{ formatModeLabel(t, 'basics') }}</option>
          <option value="project">{{ formatModeLabel(t, 'project') }}</option>
        </select>
        <select
          :value="filters.topic"
          class="neo-select w-full"
          @change="emitFilter('topic', $event)"
        >
          <option value="">{{ t('history.filters.allTopics') }}</option>
          <option v-for="topic in availableTopics" :key="topic" :value="topic">
            {{ formatTopicLabel(t, topic) }}
          </option>
        </select>
        <select
          :value="filters.status"
          class="neo-select w-full"
          @change="emitFilter('status', $event)"
        >
          <option value="">{{ t('history.filters.allStatuses') }}</option>
          <option value="completed">
            {{ formatStatusLabel(t, 'completed') }}
          </option>
          <option value="waiting_answer">
            {{ formatStatusLabel(t, 'waiting_answer') }}
          </option>
          <option value="review_pending">
            {{ formatStatusLabel(t, 'review_pending') }}
          </option>
        </select>
      </div>
    </section>

    <section class="neo-panel-soft history-batch-panel">
      <div class="space-y-2">
        <p class="neo-kicker bg-[var(--neo-green)]">
          {{ t('history.batch.kicker') }}
        </p>
        <p class="text-base font-black">
          {{ t('history.batch.selectedCount', { count: selectedCount }) }}
        </p>
        <p class="neo-note">
          {{ t('history.batch.description', { format: exportFormatLabel }) }}
        </p>
      </div>

      <label class="space-y-2">
        <span class="text-xs font-black uppercase tracking-[0.08em]">
          {{ t('common.exportFormatLabel') }}
        </span>
        <select
          :value="exportFormat"
          class="neo-select"
          @change="
            emit(
              'update:exportFormat',
              ($event.target as HTMLSelectElement).value,
            )
          "
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

      <div class="history-batch-actions">
        <button
          type="button"
          class="neo-button history-select-page-button w-full bg-white"
          @click="emit('toggle-select-all')"
        >
          {{
            allSelectedOnPage
              ? t('history.batch.clearPageAction', { count: pageCount })
              : t('history.batch.selectPageAction', { count: pageCount })
          }}
        </button>
        <button
          type="button"
          class="neo-button w-full bg-white"
          :disabled="selectedCount === 0"
          @click="emit('clear-selection')"
        >
          {{ t('history.batch.clearAllAction') }}
        </button>
        <button
          type="button"
          class="neo-button-dark history-batch-delete-button w-full"
          :disabled="selectedCount === 0 || isDeleting"
          @click="emit('delete')"
        >
          {{
            isDeleting
              ? t('history.batch.deletingAction')
              : t('history.batch.deleteAction', { count: selectedCount })
          }}
        </button>
        <button
          type="button"
          class="neo-button-dark w-full"
          :disabled="selectedCount === 0 || isExporting || isDeleting"
          @click="emit('export')"
        >
          {{
            isExporting
              ? t('history.batch.exportingAction')
              : t('history.batch.exportAction', {
                  count: selectedCount,
                  format: exportFormatLabel,
                })
          }}
        </button>
      </div>
    </section>

    <NoticePanel
      v-if="exportError"
      tone="error"
      dismissible
      :title="t('history.exportErrorTitle')"
      :message="exportError"
      @dismiss="emit('dismiss-export-error')"
    />

    <NoticePanel
      v-if="deleteError"
      tone="error"
      dismissible
      :title="t('history.deleteErrorTitle')"
      :message="deleteError"
      @dismiss="emit('dismiss-delete-error')"
    />
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import NoticePanel from './NoticePanel.vue';
import {
  formatModeLabel,
  formatStatusLabel,
  formatTopicLabel,
} from '../lib/labels';

defineProps<{
  filters: { mode: string; topic: string; status: string };
  availableTopics: string[];
  selectedCount: number;
  allSelectedOnPage: boolean;
  pageCount: number;
  exportFormat: string;
  exportFormatLabel: string;
  exportFormatOptions: Array<{ value: string; label: string }>;
  exportError: string;
  deleteError: string;
  isExporting: boolean;
  isDeleting: boolean;
}>();

const emit = defineEmits<{
  (event: 'update:filter', payload: { field: string; value: string }): void;
  (event: 'update:exportFormat', value: string): void;
  (event: 'toggle-select-all'): void;
  (event: 'clear-selection'): void;
  (event: 'delete'): void;
  (event: 'export'): void;
  (event: 'dismiss-export-error'): void;
  (event: 'dismiss-delete-error'): void;
}>();

const { t } = useI18n();

function emitFilter(field: string, event: Event) {
  emit('update:filter', {
    field,
    value: (event.target as HTMLSelectElement).value,
  });
}
</script>

<style scoped>
.history-side {
  min-width: 0;
}

.history-filter-panel,
.history-batch-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.history-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.history-section-title {
  font-size: 1.35rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.history-filter-grid,
.history-batch-actions {
  display: grid;
  gap: 0.75rem;
}

@media (min-width: 1280px) {
  .history-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
