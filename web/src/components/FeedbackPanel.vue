<template>
  <div class="space-y-4">
    <div
      class="flex items-center gap-4 border-l-8 bg-white px-4 py-4"
      :style="{ borderColor: `var(${scoreColor(evaluation.score)})` }"
    >
      <span class="text-5xl font-black">{{ evaluation.score }}</span>
      <div>
        <p class="text-sm font-bold uppercase tracking-[0.08em]">/10</p>
        <p class="text-base font-semibold leading-7">
          {{ evaluation.headline || t('session.feedbackHeadlineFallback') }}
        </p>
      </div>
    </div>

    <details v-if="evaluation.strengths.length" class="space-y-2">
      <summary class="neo-subheading cursor-pointer">
        {{ t('session.strengths') }}
      </summary>
      <ul class="mt-3 space-y-2">
        <li v-for="item in evaluation.strengths" :key="item" class="neo-note">
          {{ item }}
        </li>
      </ul>
    </details>

    <details v-if="evaluation.gaps.length" class="space-y-2" open>
      <summary class="neo-subheading cursor-pointer">
        {{ t('session.gaps') }}
      </summary>
      <ul class="mt-3 space-y-2">
        <li v-for="item in evaluation.gaps" :key="item" class="neo-note">
          {{ item }}
        </li>
      </ul>
    </details>

    <div class="space-y-2">
      <p class="neo-subheading">{{ t('session.suggestionTitle') }}</p>
      <p class="neo-note">
        {{ evaluation.suggestion || t('session.suggestionFallback') }}
      </p>
    </div>

    <details
      v-if="Object.keys(evaluation.score_breakdown ?? {}).length"
      class="space-y-2"
    >
      <summary class="neo-subheading cursor-pointer">
        {{ t('session.scoreBreakdownTitle') }}
      </summary>
      <ul class="mt-3 space-y-2">
        <li
          v-for="(score, label) in evaluation.score_breakdown"
          :key="label"
          class="border-2 border-black bg-white px-3 py-2 md:border-4"
        >
          <div class="flex items-center justify-between text-sm font-semibold">
            <span>{{ label }}</span>
            <span>{{ score }}/10</span>
          </div>
          <div
            class="neo-score-bar mt-1"
            :style="{
              width: `${Number(score) * 10}%`,
              background: `var(${scoreColor(Number(score))})`,
            }"
          />
        </li>
      </ul>
    </details>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

import type { TrainingEvaluation } from '../api/client';

defineProps<{
  evaluation: TrainingEvaluation;
}>();

const { t } = useI18n();

function scoreColor(score: number): string {
  if (score >= 8) return '--neo-green';
  if (score >= 5) return '--neo-yellow';
  return '--neo-red';
}
</script>
