<template>
  <header class="neo-panel-hero session-stage bg-[var(--neo-yellow)]">
    <div class="session-stage-copy">
      <p class="neo-kicker bg-white">{{ t('session.hero.kicker') }}</p>
      <h1
        class="session-stage-title"
        :class="{ 'session-stage-title-dense': titleDense }"
      >
        {{ title }}
      </h1>
      <p class="session-stage-note">
        {{ note }}
      </p>
    </div>

    <div class="session-stage-stats">
      <article v-for="item in stats" :key="item.label" class="session-stage-stat">
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
  titleDense: boolean;
  note: string;
  stats: Array<{
    value: string;
    label: string;
  }>;
}>();

const { t } = useI18n();
</script>

<style scoped>
.session-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-yellow) 88%, white) 0%,
    color-mix(in srgb, var(--neo-yellow) 60%, var(--neo-red)) 100%
  );
}

.session-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.session-stage-copy,
.session-stage-stats {
  position: relative;
  z-index: 1;
}

.session-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.session-stage-title {
  font-size: clamp(1.8rem, 5vw, 3.8rem);
  font-weight: 900;
  letter-spacing: -0.05em;
  line-height: 1;
  margin: 0;
  max-width: 16ch;
  text-wrap: balance;
}

.session-stage-title-dense {
  font-size: clamp(1.35rem, 2.8vw, 2.3rem);
  letter-spacing: -0.03em;
  line-height: 1.24;
  max-width: 28ch;
  text-wrap: pretty;
}

.session-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 40rem;
  text-wrap: pretty;
}

.session-stage-stats {
  display: grid;
  gap: 0.75rem;
}

.session-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.session-stage-stat span {
  font-size: clamp(2rem, 7vw, 3.4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.session-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  line-height: 1.4;
  text-transform: uppercase;
}

@media (min-width: 768px) {
  .session-stage-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .session-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.15fr) minmax(18rem, 0.85fr);
  }
}
</style>
