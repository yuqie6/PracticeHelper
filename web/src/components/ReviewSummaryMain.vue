<template>
  <main class="review-main">
    <section class="neo-panel review-focus-panel">
      <div class="review-section-head">
        <div class="space-y-2">
          <p class="neo-kicker bg-[var(--neo-red)]">
            {{ t('review.topFixTitle') }}
          </p>
          <h2
            class="review-focus-title"
            :class="{ 'review-focus-title-dense': focusTitleDense }"
          >
            {{ focusTitle }}
          </h2>
        </div>
        <span class="neo-badge bg-white">{{ scoreAverageDisplay }}</span>
      </div>
      <p class="neo-note">
        {{ focusReason }}
      </p>
      <div
        v-if="recommendedNextLabel || recommendedNextReason"
        class="review-next-box"
      >
        <p class="neo-subheading">{{ t('review.recommendedNextTitle') }}</p>
        <p class="text-base font-black">
          {{ recommendedNextLabel || t('review.continueAction') }}
        </p>
        <p v-if="recommendedNextReason" class="neo-note">
          {{ recommendedNextReason }}
        </p>
        <RouterLink :to="continueTarget" class="neo-button-red mt-2 w-full sm:w-auto">
          {{ t('review.startRecommendedAction') }}
        </RouterLink>
      </div>
    </section>

    <div class="review-grid neo-stagger-list">
      <section class="neo-panel review-list-panel">
        <p class="neo-kicker bg-[var(--neo-yellow)]">
          {{ t('review.highlights') }}
        </p>
        <ul class="review-list neo-stagger-list">
          <li v-for="item in highlights" :key="item" class="neo-note">
            {{ item }}
          </li>
        </ul>
      </section>

      <section class="neo-panel review-list-panel">
        <p class="neo-kicker bg-[var(--neo-red)]">{{ t('review.gaps') }}</p>
        <ul class="review-list neo-stagger-list">
          <li v-for="item in gaps" :key="item" class="neo-note">
            {{ item }}
          </li>
        </ul>
      </section>

      <section class="neo-panel review-list-panel">
        <p class="neo-kicker bg-[var(--neo-green)]">
          {{ t('review.nextFocus') }}
        </p>
        <ul class="review-list neo-stagger-list">
          <li v-for="item in nextTrainingFocus" :key="item" class="neo-note">
            {{ item }}
          </li>
        </ul>
        <RouterLink :to="continueTarget" class="neo-button-red mt-4 w-full sm:w-auto">
          {{ t('review.continueAction') }}
        </RouterLink>
      </section>
    </div>
  </main>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

defineProps<{
  focusTitle: string;
  focusTitleDense: boolean;
  focusReason: string;
  scoreAverageDisplay: string;
  recommendedNextLabel: string;
  recommendedNextReason: string;
  continueTarget: string;
  highlights: string[];
  gaps: string[];
  nextTrainingFocus: string[];
}>();

const { t } = useI18n();
</script>

<style scoped>
.review-main {
  display: grid;
  gap: 1rem;
}

.review-focus-panel,
.review-list-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.review-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.review-focus-title {
  font-size: clamp(1.2rem, 2.4vw, 2rem);
  font-weight: 900;
  letter-spacing: -0.03em;
  line-height: 1.12;
  margin: 0;
  max-width: 24ch;
  text-wrap: balance;
}

.review-focus-title-dense {
  font-size: clamp(1.05rem, 2vw, 1.5rem);
  line-height: 1.35;
  max-width: 34ch;
  text-wrap: pretty;
}

.review-next-box {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
}

.review-grid {
  display: grid;
  gap: 1rem;
}

.review-list {
  display: grid;
  gap: 0.75rem;
  margin: 0;
  padding-left: 1rem;
}

@media (min-width: 768px) {
  .review-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .review-grid > :last-child {
    grid-column: 1 / -1;
  }
}
</style>
