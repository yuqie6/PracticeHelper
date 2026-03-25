<template>
  <header class="neo-panel-hero prompt-stage bg-[var(--neo-blue)]">
    <div class="prompt-stage-copy">
      <p class="neo-kicker bg-white">{{ t('promptExperiments.hero.kicker') }}</p>
      <h1 class="prompt-stage-title">{{ t('promptExperiments.hero.title') }}</h1>
      <p class="prompt-stage-note">{{ t('promptExperiments.hero.description') }}</p>
    </div>

    <div class="prompt-stage-stats">
      <article v-for="item in stats" :key="item.label" class="prompt-stage-stat">
        <span>{{ item.value }}</span>
        <small>{{ item.label }}</small>
      </article>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
defineProps<{ stats: Array<{ label: string; value: string | number }> }>();
const { t } = useI18n();
</script>

<style scoped>
.prompt-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-blue) 84%, white) 0%,
    color-mix(in srgb, var(--neo-blue) 58%, var(--neo-yellow)) 100%
  );
}

.prompt-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.prompt-stage-copy,
.prompt-stage-stats {
  position: relative;
  z-index: 1;
}

.prompt-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.prompt-stage-title {
  font-size: clamp(2rem, 5vw, 4.2rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 1;
  margin: 0;
  max-width: 12ch;
}

.prompt-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 38rem;
}

.prompt-stage-stats {
  display: grid;
  gap: 0.75rem;
}

.prompt-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.prompt-stage-stat span {
  font-size: clamp(2.2rem, 7vw, 3.8rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.prompt-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

@media (min-width: 768px) {
  .prompt-stage-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .prompt-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.1fr) minmax(18rem, 0.9fr);
  }
}
</style>
