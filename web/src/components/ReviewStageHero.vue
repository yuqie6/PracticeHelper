<template>
  <header class="neo-panel-hero review-stage bg-[var(--neo-blue)]">
    <div class="review-stage-copy">
      <p class="neo-kicker bg-white">{{ t('review.hero.kicker') }}</p>
      <h1
        class="review-stage-title"
        :class="{ 'review-stage-title-dense': titleDense }"
      >
        {{ title }}
      </h1>
      <p class="review-stage-note">
        {{ note }}
      </p>
    </div>

    <div class="review-stage-side">
      <article v-for="item in stats" :key="item.label" class="review-stage-stat">
        <span>{{ item.value }}</span>
        <small>{{ item.label }}</small>
      </article>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

defineProps<{
  title: string;
  note: string;
  titleDense: boolean;
  stats: Array<{
    label: string;
    value: string | number;
  }>;
}>();

const { t } = useI18n();
</script>

<style scoped>
.review-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-blue) 84%, white) 0%,
    color-mix(in srgb, var(--neo-blue) 58%, var(--neo-green)) 100%
  );
}

.review-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.review-stage-copy,
.review-stage-side {
  position: relative;
  z-index: 1;
}

.review-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.review-stage-title {
  font-size: clamp(2rem, 5vw, 4.2rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 1;
  margin: 0;
  max-width: 14ch;
  text-wrap: balance;
}

.review-stage-title-dense {
  font-size: clamp(1.45rem, 2.8vw, 2.4rem);
  letter-spacing: -0.03em;
  line-height: 1.25;
  max-width: 28ch;
  text-wrap: pretty;
}

.review-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 40rem;
  text-wrap: pretty;
}

.review-stage-side {
  display: grid;
  gap: 0.75rem;
}

.review-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.review-stage-stat span {
  font-size: clamp(2.2rem, 7vw, 3.8rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.review-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

@media (min-width: 768px) {
  .review-stage-side {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .review-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.1fr) minmax(18rem, 0.9fr);
  }
}
</style>
