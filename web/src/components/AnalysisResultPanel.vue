<template>
  <div class="analysis-panel">
    <p v-if="description" class="neo-note">{{ description }}</p>

    <section class="analysis-block">
      <p class="neo-kicker bg-white">{{ t('jobs.fields.summary') }}</p>
      <p class="analysis-summary">{{ analysis.summary }}</p>
    </section>

    <div class="analysis-grid">
      <section class="analysis-block">
        <p class="neo-subheading">{{ t('jobs.fields.mustHaveSkills') }}</p>
        <ul class="analysis-list">
          <li
            v-for="item in analysis.must_have_skills"
            :key="item"
            class="analysis-list-item"
          >
            {{ item }}
          </li>
        </ul>
      </section>

      <section v-if="analysis.bonus_skills.length" class="analysis-block">
        <p class="neo-subheading">{{ t('jobs.fields.bonusSkills') }}</p>
        <ul class="analysis-list">
          <li
            v-for="item in analysis.bonus_skills"
            :key="item"
            class="analysis-list-item"
          >
            {{ item }}
          </li>
        </ul>
      </section>

      <section class="analysis-block">
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

      <section class="analysis-block">
        <p class="neo-subheading">{{ t('jobs.fields.evaluationFocus') }}</p>
        <ul class="analysis-list">
          <li
            v-for="item in analysis.evaluation_focus"
            :key="item"
            class="analysis-list-item"
          >
            {{ item }}
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { JobTargetAnalysisRun } from '../api/client';

defineProps<{
  analysis: JobTargetAnalysisRun;
  description?: string;
}>();

const { t } = useI18n();
</script>

<style scoped>
.analysis-panel {
  display: grid;
  gap: 1rem;
}

.analysis-block {
  display: grid;
  gap: 0.7rem;
}

.analysis-summary {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.8;
  margin: 0;
}

.analysis-grid {
  display: grid;
  gap: 1rem;
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
  .analysis-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
