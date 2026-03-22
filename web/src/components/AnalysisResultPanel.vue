<template>
  <div class="analysis-panel">
    <p v-if="description" class="neo-note">{{ description }}</p>

    <section class="analysis-block analysis-block-surface">
      <p class="neo-kicker bg-white">{{ t('jobs.fields.summary') }}</p>
      <p class="analysis-summary">{{ analysis.summary }}</p>
    </section>

    <div v-if="primarySections.length" class="analysis-primary-grid">
      <section
        v-for="section in primarySections"
        :key="section.key"
        class="analysis-block analysis-block-surface"
      >
        <p class="neo-subheading">{{ section.title }}</p>
        <div class="analysis-chip-cloud">
          <span v-for="item in section.items" :key="item" class="analysis-chip">
            {{ item }}
          </span>
        </div>
      </section>
    </div>

    <div v-if="secondarySections.length" class="analysis-secondary-stack">
      <section
        v-for="section in secondarySections"
        :key="section.key"
        class="analysis-block analysis-block-surface"
      >
        <p class="neo-subheading">{{ section.title }}</p>
        <div class="analysis-chip-cloud">
          <span v-for="item in section.items" :key="item" class="analysis-chip">
            {{ item }}
          </span>
        </div>
      </section>
    </div>

    <section
      v-if="analysis.responsibilities.length"
      class="analysis-block analysis-block-surface"
    >
      <p class="neo-subheading">{{ t('jobs.fields.responsibilities') }}</p>
      <ul class="analysis-list">
        <li
          v-for="item in analysis.responsibilities"
          :key="item"
          class="analysis-list-item"
        >
          {{ item }}
        </li>
      </ul>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';

import type { JobTargetAnalysisRun } from '../api/client';

const props = defineProps<{
  analysis: JobTargetAnalysisRun;
  description?: string;
}>();

const { t } = useI18n();

const primarySections = computed(() =>
  [
    {
      key: 'must-have',
      title: t('jobs.fields.mustHaveSkills'),
      items: props.analysis.must_have_skills,
    },
    {
      key: 'focus',
      title: t('jobs.fields.evaluationFocus'),
      items: props.analysis.evaluation_focus,
    },
  ].filter((section) => section.items.length > 0),
);

const secondarySections = computed(() =>
  [
    {
      key: 'bonus',
      title: t('jobs.fields.bonusSkills'),
      items: props.analysis.bonus_skills,
    },
  ].filter((section) => section.items.length > 0),
);
</script>

<style scoped>
.analysis-panel {
  display: grid;
  gap: 1rem;
}

.analysis-block {
  align-content: start;
  display: grid;
  gap: 0.7rem;
}

.analysis-block-surface {
  background: color-mix(in srgb, var(--neo-surface) 86%, transparent);
  border: 2px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  padding: 1rem;
}

.analysis-summary {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.8;
  margin: 0;
  text-wrap: pretty;
}

.analysis-primary-grid,
.analysis-secondary-stack {
  display: grid;
  gap: 1rem;
}

.analysis-chip-cloud {
  display: flex;
  flex-wrap: wrap;
  gap: 0.65rem;
}

.analysis-chip {
  background: color-mix(in srgb, var(--neo-paper) 84%, transparent);
  border: 2px solid var(--neo-border);
  display: inline-flex;
  font-size: 0.9rem;
  font-weight: 700;
  line-height: 1.5;
  padding: 0.55rem 0.8rem;
}

.analysis-list {
  display: grid;
  gap: 0.7rem;
  margin: 0;
  padding: 0;
}

.analysis-list-item {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  list-style: none;
  padding: 0.85rem 1rem;
}

@media (min-width: 768px) {
  .analysis-primary-grid {
    grid-template-columns: repeat(auto-fit, minmax(14rem, 1fr));
  }
}
</style>
