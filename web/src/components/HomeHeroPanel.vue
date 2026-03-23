<template>
  <section class="neo-panel-hero home-stage-main bg-[var(--neo-yellow)]">
    <div class="home-stage-copy">
      <p class="neo-kicker bg-white">{{ t('home.hero.kicker') }}</p>
      <h2 class="home-stage-title">
        {{ title }}
      </h2>
      <p class="home-stage-support">
        {{ support }}
      </p>

      <div class="home-stage-actions">
        <RouterLink to="/train" class="neo-button-red w-full sm:w-auto">
          {{ t('home.hero.actionPrimary') }}
        </RouterLink>
        <RouterLink
          to="/projects"
          class="neo-button w-full bg-white sm:w-auto"
        >
          {{ t('home.hero.actionSecondary') }}
        </RouterLink>
      </div>

      <div v-if="showOnboarding" class="home-stage-steps neo-stagger-list">
        <RouterLink
          v-for="step in onboardingSteps"
          :key="step.key"
          :to="step.href"
          class="home-stage-step"
          :class="step.toneClass"
        >
          <div class="flex items-start gap-3">
            <span class="neo-badge bg-white">{{ step.index }}</span>
            <div class="space-y-1">
              <p class="text-base font-black">
                {{ step.label }}
              </p>
              <p class="neo-note">
                {{ step.hint }}
              </p>
            </div>
          </div>
          <span class="home-stage-step-status">
            {{ t(`home.onboarding.status.${step.status}`) }}
          </span>
        </RouterLink>
      </div>
    </div>

    <div class="home-stage-visual" aria-hidden="true">
      <div class="home-stage-ring home-stage-ring-lg"></div>
      <div class="home-stage-ring home-stage-ring-sm"></div>
      <div class="home-stage-banner">{{ t('app.name') }}</div>
      <div class="home-stage-chip home-stage-chip-deadline">
        <span>{{ deadlineDisplay }}</span>
        <small>{{ t('common.daysRemainingLabel') }}</small>
      </div>
      <div class="home-stage-chip home-stage-chip-reviews">
        <span>{{ dueReviewDisplay }}</span>
        <small>{{ t('home.dueReviews.kicker') }}</small>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

type HomeOnboardingStatus = 'done' | 'current' | 'next';

defineProps<{
  title: string;
  support: string;
  showOnboarding: boolean;
  deadlineDisplay: string;
  dueReviewDisplay: string;
  onboardingSteps: Array<{
    key: string;
    index: number;
    label: string;
    hint: string;
    href: string;
    status: HomeOnboardingStatus;
    toneClass: string;
  }>;
}>();

const { t } = useI18n();
</script>

<style scoped>
.home-stage-main {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-yellow) 88%, white) 0%,
    color-mix(in srgb, var(--neo-yellow) 64%, var(--neo-red)) 100%
  );
}

.home-stage-main::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 24%, transparent);
  pointer-events: none;
}

.home-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  position: relative;
  z-index: 1;
}

.home-stage-title {
  font-size: clamp(2.25rem, 7vw, 5.5rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.92;
  margin: 0;
  max-width: 10ch;
  text-transform: uppercase;
}

.home-stage-support {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 34rem;
}

.home-stage-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.home-stage-steps {
  border-top: 2px solid color-mix(in srgb, var(--neo-border) 22%, transparent);
  display: grid;
  gap: 0.75rem;
  margin-top: 0.25rem;
  padding-top: 1rem;
}

.home-stage-step {
  border: 2px solid var(--neo-border);
  box-shadow: 4px 4px 0 0
    rgba(var(--neo-shadow-rgb), calc(var(--neo-shadow-alpha) * 0.75));
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
  transition:
    transform var(--motion-duration-base) var(--motion-ease-standard),
    box-shadow var(--motion-duration-base) var(--motion-ease-standard),
    background-color var(--motion-duration-fast) var(--motion-ease-soft);
}

.home-stage-step:hover {
  box-shadow: 8px 8px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(var(--motion-lift-md), var(--motion-lift-md));
}

.home-stage-step-status {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.home-stage-visual {
  min-height: 16rem;
  position: relative;
}

.home-stage-ring {
  border: 2px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  border-radius: 999px;
  position: absolute;
}

.home-stage-ring-lg {
  height: 13rem;
  right: -2rem;
  top: 0.5rem;
  width: 13rem;
}

.home-stage-ring-sm {
  bottom: 1.5rem;
  height: 7rem;
  left: 1rem;
  width: 7rem;
}

.home-stage-banner {
  border: 2px solid var(--neo-border);
  box-shadow: 4px 4px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  font-size: 0.78rem;
  font-weight: 900;
  left: 1rem;
  letter-spacing: 0.24em;
  padding: 0.55rem 0.85rem;
  position: absolute;
  text-transform: uppercase;
  top: 1rem;
  transform: rotate(-8deg);
}

.home-stage-chip {
  --home-chip-rotate: 0deg;
  animation: home-chip-in 420ms cubic-bezier(0.22, 1, 0.36, 1) both;
  background: color-mix(in srgb, var(--neo-surface) 92%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 8px 8px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  padding: 1rem 1.1rem;
  position: absolute;
}

.home-stage-chip span {
  font-size: clamp(2.5rem, 9vw, 4.8rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.home-stage-chip small {
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  line-height: 1.4;
  max-width: 11rem;
  text-transform: uppercase;
}

.home-stage-chip-deadline {
  --home-chip-rotate: 4deg;
  right: 0.5rem;
  top: 3.25rem;
  transform: rotate(var(--home-chip-rotate));
}

.home-stage-chip-reviews {
  --home-chip-rotate: -4deg;
  bottom: 1rem;
  left: 3rem;
  transform: rotate(var(--home-chip-rotate));
}

@keyframes home-chip-in {
  from {
    opacity: 0;
    transform: translateY(18px) rotate(var(--home-chip-rotate)) scale(0.96);
  }

  to {
    opacity: 1;
    transform: rotate(var(--home-chip-rotate)) scale(1);
  }
}

@media (min-width: 768px) {
  .home-stage-steps {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (min-width: 1280px) {
  .home-stage-main {
    align-items: stretch;
    gap: 2rem;
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 0.72fr);
    min-height: 32rem;
  }

  .home-stage-visual {
    min-height: auto;
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-stage-step,
  .home-stage-chip {
    animation: none;
    transition: none;
  }

  .home-stage-step:hover {
    box-shadow: inherit;
    transform: none;
  }
}
</style>
