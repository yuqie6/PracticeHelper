<template>
  <header class="neo-panel-hero jobs-stage bg-[var(--neo-blue)]">
    <div class="jobs-stage-copy">
      <p class="neo-kicker bg-white">{{ t('jobs.hero.kicker') }}</p>
      <h1 class="jobs-stage-title">{{ t('app.nav.jobs') }}</h1>
      <p class="jobs-stage-note">{{ t('jobs.hero.description') }}</p>
    </div>

    <div class="jobs-stage-side">
      <article v-for="item in stats" :key="item.label" class="jobs-stage-stat">
        <span>{{ item.value }}</span>
        <small>{{ item.label }}</small>
      </article>
      <button type="button" class="neo-button-red w-full" @click="emit('create')">
        {{ t('jobs.createAction') }}
      </button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';

defineProps<{
  stats: Array<{
    label: string;
    value: string | number;
  }>;
}>();

const emit = defineEmits<{
  (event: 'create'): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
.jobs-stage {
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

.jobs-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.jobs-stage-copy,
.jobs-stage-side {
  position: relative;
  z-index: 1;
}

.jobs-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.jobs-stage-title {
  font-size: clamp(2.1rem, 6vw, 4.6rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
  max-width: 10ch;
  text-transform: uppercase;
}

.jobs-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 38rem;
}

.jobs-stage-side {
  display: grid;
  gap: 0.75rem;
}

.jobs-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.jobs-stage-stat span {
  font-size: clamp(2.4rem, 8vw, 4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.jobs-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

@media (min-width: 768px) {
  .jobs-stage-side {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .jobs-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.15fr) minmax(18rem, 0.85fr);
  }
}
</style>
