<template>
  <main class="history-main">
    <div v-if="isLoading" class="space-y-3">
      <div v-for="n in 5" :key="n" class="neo-skeleton h-24" />
    </div>

    <section v-else-if="!sessions.length" class="neo-panel history-empty-panel">
      <p class="neo-note">{{ t('history.empty') }}</p>
    </section>

    <section v-else class="neo-panel history-results-panel neo-stagger-list">
      <div class="history-section-head">
        <div class="space-y-2">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ t('history.openAction') }}
          </p>
          <h2 class="history-section-title">{{ t('history.hero.title') }}</h2>
        </div>
        <span class="neo-badge bg-white"
          >{{ currentPage }} / {{ totalPages }}</span
        >
      </div>

      <article v-for="item in sessions" :key="item.id" class="history-row">
        <div class="history-row-actions">
          <label class="flex shrink-0 items-center pt-1">
            <input
              class="neo-checkbox"
              type="checkbox"
              :checked="selectedSessionIds.includes(item.id)"
              :disabled="deletingSessionIds.includes(item.id)"
              @change="emit('toggle-selected', item.id)"
            />
          </label>
          <button
            type="button"
            class="neo-button history-delete-button"
            :disabled="deletingSessionIds.includes(item.id)"
            @click="emit('delete', item.id)"
          >
            {{
              deletingSessionIds.includes(item.id)
                ? t('history.deletingAction')
                : t('history.deleteAction')
            }}
          </button>
        </div>

        <RouterLink :to="resolveSessionLink(item)" class="history-row-link">
          <div class="history-row-top">
            <div class="space-y-2">
              <span class="block text-base font-black">
                {{ formatModeLabel(t, item.mode) }}
                <template v-if="item.topic">
                  · {{ formatTopicLabel(t, item.topic) }}</template
                >
                <template v-if="item.project_name">
                  · {{ item.project_name }}</template
                >
              </span>
              <span
                v-if="item.prompt_set"
                class="neo-badge bg-[var(--neo-blue)]"
              >
                {{
                  t('history.promptSetBadge', { name: item.prompt_set.label })
                }}
              </span>
              <span
                v-if="item.prompt_overlay_hash"
                class="neo-badge bg-[var(--neo-green)]"
              >
                {{ t('history.promptOverlayBadge') }}
              </span>
            </div>
            <span class="text-sm font-semibold">
              {{ formatStatusLabel(t, item.status) }}
            </span>
          </div>

          <div class="history-row-middle">
            <span v-if="item.job_target" class="neo-note">{{
              item.job_target.title
            }}</span>
            <span v-else class="neo-note">{{ t('history.noJobTarget') }}</span>
            <span class="text-sm font-black">{{
              item.total_score > 0 ? item.total_score : '—'
            }}</span>
          </div>

          <div class="history-row-bottom">
            <span class="neo-note">{{
              new Date(item.updated_at).toLocaleString()
            }}</span>
            <span class="neo-badge bg-[var(--neo-yellow)]">{{
              t('history.openAction')
            }}</span>
          </div>
        </RouterLink>
      </article>
    </section>
  </main>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

import type { TrainingSessionSummary } from '../api/client';
import {
  formatModeLabel,
  formatStatusLabel,
  formatTopicLabel,
} from '../lib/labels';

defineProps<{
  isLoading: boolean;
  sessions: TrainingSessionSummary[];
  currentPage: number;
  totalPages: number;
  selectedSessionIds: string[];
  deletingSessionIds: string[];
  resolveSessionLink: (item: TrainingSessionSummary) => string;
}>();

const emit = defineEmits<{
  (event: 'toggle-selected', id: string): void;
  (event: 'delete', id: string): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
.history-main {
  min-width: 0;
}

.history-results-panel,
.history-empty-panel {
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

.history-row {
  border-top: 1px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: grid;
  gap: 1rem;
  grid-template-columns: auto minmax(0, 1fr);
  padding-top: 1rem;
}

.history-row:first-of-type {
  border-top: 0;
  padding-top: 0;
}

.history-row-actions {
  align-items: stretch;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.history-row-link {
  display: grid;
  gap: 0.85rem;
  transition: transform var(--motion-duration-base) var(--motion-ease-standard);
}

.history-row-link:hover {
  transform: translateX(4px);
}

.history-row-top,
.history-row-middle,
.history-row-bottom {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.history-delete-button {
  min-width: 5.5rem;
}

@media (min-width: 768px) {
  .history-row-top,
  .history-row-middle,
  .history-row-bottom {
    align-items: center;
    flex-direction: row;
    justify-content: space-between;
  }

  .history-row-actions {
    align-items: start;
  }
}
</style>
