<template>
  <header class="neo-panel-hero history-stage bg-[var(--neo-yellow)]">
    <div class="history-stage-copy">
      <p class="neo-kicker bg-white">{{ t('history.hero.kicker') }}</p>
      <h1 class="history-stage-title">{{ t('history.hero.title') }}</h1>
      <p class="history-stage-note">
        {{ t('history.batch.description', { format: exportFormatLabel }) }}
      </p>
    </div>

    <div class="history-stage-stats">
      <article v-for="item in stats" :key="item.label" class="history-stage-stat">
        <span>{{ item.value }}</span>
        <small>{{ item.label }}</small>
      </article>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

defineProps<{
  exportFormatLabel: string;
  stats: Array<{ label: string; value: string | number }>;
}>();

const { t } = useI18n();
</script>

<style scoped>
.history-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-yellow) 88%, white) 0%,
    color-mix(in srgb, var(--neo-yellow) 60%, var(--neo-green)) 100%
  );
}

.history-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.history-stage-copy,
.history-stage-stats {
  position: relative;
  z-index: 1;
}

.history-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.history-stage-title {
  font-size: clamp(2.1rem, 6vw, 4.5rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
  max-width: 11ch;
  text-transform: uppercase;
}

.history-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 38rem;
}

.history-stage-stats {
  display: grid;
  gap: 0.75rem;
}

.history-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.history-stage-stat span {
  font-size: clamp(2.4rem, 8vw, 4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.history-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

@media (min-width: 768px) {
  .history-stage-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .history-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.15fr) minmax(18rem, 0.85fr);
  }
}
</style>
