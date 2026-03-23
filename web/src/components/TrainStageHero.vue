<template>
  <header class="neo-panel-hero train-stage bg-[var(--neo-red)] text-black">
    <div class="train-stage-copy">
      <p class="neo-kicker bg-white">{{ t('train.hero.kicker') }}</p>
      <h1 class="train-stage-title">
        {{ t('train.hero.title') }}
      </h1>
      <p class="train-stage-note">
        {{ t('train.hero.description') }}
      </p>

      <div class="train-stage-stats neo-stagger-list">
        <article class="train-stage-stat">
          <span>{{ projectCount }}</span>
          <small>{{ t('projects.listTitle') }}</small>
        </article>
        <article class="train-stage-stat">
          <span>{{ readyJobTargetCount }}</span>
          <small>{{ t('jobs.listTitle') }}</small>
        </article>
        <article class="train-stage-stat">
          <span>{{ promptSetCount }}</span>
          <small>{{ t('train.fields.promptSet') }}</small>
        </article>
      </div>
    </div>

    <div class="train-stage-side">
      <article v-if="currentSession" class="train-stage-context">
        <p class="neo-kicker bg-white">
          {{ t('home.currentSession.kicker') }}
        </p>
        <h2 class="text-xl font-black">{{ t('train.resumeTitle') }}</h2>
        <p class="neo-note">
          {{ currentSession.description }}
        </p>
        <RouterLink :to="currentSession.href" class="neo-button-dark w-full">
          {{ t('common.resume') }}
        </RouterLink>
      </article>

      <article
        v-else-if="onboardingMode"
        class="train-stage-context bg-[color:var(--neo-yellow)]"
      >
        <p class="neo-kicker bg-white">{{ t('train.onboarding.kicker') }}</p>
        <h2 class="text-xl font-black">{{ t('train.onboarding.title') }}</h2>
        <p class="neo-note">{{ t('train.onboarding.description') }}</p>
      </article>

      <article v-else class="train-stage-context">
        <p class="neo-kicker bg-white">{{ t('train.fields.jobTarget') }}</p>
        <h2 class="text-xl font-black">
          {{ fallbackJobTargetTitle }}
        </h2>
        <p class="neo-note">
          {{ fallbackJobTargetDescription }}
        </p>
      </article>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

defineProps<{
  onboardingMode: boolean;
  projectCount: number;
  readyJobTargetCount: number;
  promptSetCount: number;
  fallbackJobTargetTitle: string;
  fallbackJobTargetDescription: string;
  currentSession: {
    description: string;
    href: string;
  } | null;
}>();

const { t } = useI18n();
</script>

<style scoped>
.train-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-red) 82%, white) 0%,
    color-mix(in srgb, var(--neo-red) 58%, var(--neo-yellow)) 100%
  );
}

.train-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.train-stage-copy,
.train-stage-side {
  position: relative;
  z-index: 1;
}

.train-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.train-stage-title {
  font-size: clamp(2.1rem, 6vw, 4.8rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
  max-width: 11ch;
  text-transform: uppercase;
}

.train-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 38rem;
}

.train-stage-stats {
  display: grid;
  gap: 0.75rem;
}

.train-stage-stat,
.train-stage-context {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.train-stage-stat span {
  font-size: clamp(2.4rem, 8vw, 4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.train-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.train-stage-side {
  display: grid;
  gap: 1rem;
}

@media (min-width: 768px) {
  .train-stage-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .train-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.2fr) minmax(19rem, 0.8fr);
  }
}
</style>
