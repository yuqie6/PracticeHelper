<template>
  <div class="space-y-4 border-2 border-black bg-white px-4 py-4 md:border-4">
    <p v-if="description" class="neo-note">{{ description }}</p>

    <div class="space-y-2">
      <p class="neo-subheading">{{ t('jobs.fields.summary') }}</p>
      <p class="neo-note">{{ analysis.summary }}</p>
    </div>

    <div class="space-y-2">
      <p class="neo-subheading">{{ t('jobs.fields.mustHaveSkills') }}</p>
      <ul class="space-y-2">
        <li
          v-for="item in analysis.must_have_skills"
          :key="item"
          class="neo-note"
        >
          {{ item }}
        </li>
      </ul>
    </div>

    <div v-if="analysis.bonus_skills.length" class="space-y-2">
      <p class="neo-subheading">{{ t('jobs.fields.bonusSkills') }}</p>
      <ul class="space-y-2">
        <li v-for="item in analysis.bonus_skills" :key="item" class="neo-note">
          {{ item }}
        </li>
      </ul>
    </div>

    <div class="space-y-2">
      <p class="neo-subheading">{{ t('jobs.fields.responsibilities') }}</p>
      <ul class="space-y-2">
        <li
          v-for="item in analysis.responsibilities"
          :key="item"
          class="neo-note"
        >
          {{ item }}
        </li>
      </ul>
    </div>

    <div class="space-y-2">
      <p class="neo-subheading">{{ t('jobs.fields.evaluationFocus') }}</p>
      <ul class="space-y-2">
        <li
          v-for="item in analysis.evaluation_focus"
          :key="item"
          class="neo-note"
        >
          {{ item }}
        </li>
      </ul>
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
