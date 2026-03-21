<template>
  <div class="feedback-panel">
    <section
      class="feedback-score"
      :style="{ '--feedback-accent': `var(${scoreColor(evaluation.score)})` }"
    >
      <div class="feedback-score-value">
        <span>{{ evaluation.score }}</span>
        <small>/10</small>
      </div>
      <div class="feedback-score-copy">
        <p class="neo-kicker bg-white">{{ t('session.feedback') }}</p>
        <h3 class="feedback-score-headline">
          {{ evaluation.headline || t('session.feedbackHeadlineFallback') }}
        </h3>
      </div>
    </section>

    <div
      v-if="evaluation.strengths.length || evaluation.gaps.length"
      class="feedback-grid"
    >
      <details v-if="evaluation.strengths.length" class="feedback-detail" open>
        <summary class="feedback-detail-summary">
          {{ t('session.strengths') }}
        </summary>
        <ul class="feedback-list">
          <li
            v-for="item in evaluation.strengths"
            :key="item"
            class="feedback-list-item"
          >
            {{ item }}
          </li>
        </ul>
      </details>

      <details v-if="evaluation.gaps.length" class="feedback-detail" open>
        <summary class="feedback-detail-summary">
          {{ t('session.gaps') }}
        </summary>
        <ul class="feedback-list">
          <li
            v-for="item in evaluation.gaps"
            :key="item"
            class="feedback-list-item"
          >
            {{ item }}
          </li>
        </ul>
      </details>
    </div>

    <section class="feedback-suggestion">
      <p class="neo-subheading">{{ t('session.suggestionTitle') }}</p>
      <p class="neo-note">
        {{ evaluation.suggestion || t('session.suggestionFallback') }}
      </p>
    </section>

    <details
      v-if="Object.keys(evaluation.score_breakdown ?? {}).length"
      class="feedback-detail"
    >
      <summary class="feedback-detail-summary">
        {{ t('session.scoreBreakdownTitle') }}
      </summary>
      <ul class="feedback-breakdown">
        <li
          v-for="(score, label) in evaluation.score_breakdown"
          :key="label"
          class="feedback-breakdown-row"
        >
          <div
            class="flex items-center justify-between gap-3 text-sm font-semibold"
          >
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

<style scoped>
.feedback-panel {
  display: grid;
  gap: 1rem;
}

.feedback-score {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  border-left: 10px solid var(--feedback-accent);
  display: grid;
  gap: 1rem;
  padding: 1rem;
}

.feedback-score-value {
  align-items: end;
  display: flex;
  gap: 0.45rem;
}

.feedback-score-value span {
  font-size: clamp(2.8rem, 8vw, 4.4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.85;
}

.feedback-score-value small {
  font-size: 0.8rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.feedback-score-copy {
  display: grid;
  gap: 0.6rem;
}

.feedback-score-headline {
  font-size: 1rem;
  font-weight: 800;
  line-height: 1.7;
  margin: 0;
}

.feedback-grid {
  display: grid;
  gap: 1rem;
}

.feedback-detail {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  padding: 1rem;
}

.feedback-detail-summary {
  cursor: pointer;
  font-size: 1rem;
  font-weight: 900;
  letter-spacing: 0.04em;
  list-style: none;
  text-transform: uppercase;
}

.feedback-detail-summary::-webkit-details-marker {
  display: none;
}

.feedback-list,
.feedback-breakdown {
  display: grid;
  gap: 0.75rem;
  margin: 0.85rem 0 0;
  padding: 0;
}

.feedback-list-item,
.feedback-breakdown-row {
  background: color-mix(in srgb, var(--neo-paper) 86%, transparent);
  border: 2px solid var(--neo-border);
  list-style: none;
  padding: 0.85rem 1rem;
}

.feedback-suggestion {
  display: grid;
  gap: 0.6rem;
}

@media (min-width: 768px) {
  .feedback-score {
    align-items: center;
    grid-template-columns: auto minmax(0, 1fr);
  }

  .feedback-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
</style>
