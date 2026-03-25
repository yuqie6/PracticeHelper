<template>
  <aside class="prompt-side">
    <section class="neo-panel prompt-filter-panel">
      <div class="prompt-section-head">
        <div class="space-y-2">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ t('promptExperiments.hero.kicker') }}
          </p>
          <h2 class="prompt-section-title">{{ t('promptExperiments.hero.title') }}</h2>
        </div>
      </div>

      <p class="neo-note">{{ t('promptExperiments.overlayNotice') }}</p>

      <div class="prompt-filter-grid">
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('promptExperiments.filters.left') }}</span>
          <select :value="filters.left" class="neo-select" @change="emitFilter('left', $event)">
            <option v-for="item in promptSets" :key="item.id" :value="item.id">{{ item.label }}</option>
          </select>
        </label>
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('promptExperiments.filters.right') }}</span>
          <select :value="filters.right" class="neo-select" @change="emitFilter('right', $event)">
            <option v-for="item in promptSets" :key="item.id" :value="item.id">{{ item.label }}</option>
          </select>
        </label>
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('promptExperiments.filters.mode') }}</span>
          <select :value="filters.mode" class="neo-select" @change="emitFilter('mode', $event)">
            <option value="">{{ t('promptExperiments.filters.allModes') }}</option>
            <option value="basics">{{ formatModeLabel(t, 'basics') }}</option>
            <option value="project">{{ formatModeLabel(t, 'project') }}</option>
          </select>
        </label>
        <label class="space-y-2">
          <span class="neo-subheading">{{ t('promptExperiments.filters.topic') }}</span>
          <select :value="filters.topic" class="neo-select" @change="emitFilter('topic', $event)">
            <option value="">{{ t('promptExperiments.filters.allTopics') }}</option>
            <option v-for="topic in availableTopics" :key="topic" :value="topic">
              {{ formatTopicLabel(t, topic) }}
            </option>
          </select>
        </label>
      </div>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import type { PromptSetSummary } from '../api/client';
import { formatModeLabel, formatTopicLabel } from '../lib/labels';

defineProps<{
  promptSets: PromptSetSummary[];
  filters: { left: string; right: string; mode: string; topic: string };
  availableTopics: string[];
}>();

const emit = defineEmits<{
  (event: 'update:filter', payload: { field: string; value: string }): void;
}>();

const { t } = useI18n();

function emitFilter(field: string, event: Event) {
  emit('update:filter', { field, value: (event.target as HTMLSelectElement).value });
}
</script>

<style scoped>
.prompt-filter-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.prompt-section-head {
  align-items: end;
  border-bottom: 2px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.prompt-section-title {
  font-size: 1.3rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.prompt-filter-grid {
  display: grid;
  gap: 1rem;
}

@media (min-width: 768px) {
  .prompt-filter-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .prompt-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
