<template>
  <aside class="home-detail-side">
    <section
      v-if="dueReviews.length"
      class="neo-panel home-queue-panel bg-[var(--neo-yellow)]"
    >
      <div class="space-y-2">
        <p class="neo-kicker bg-white">{{ t('home.dueReviews.kicker') }}</p>
        <h3 class="home-section-title">
          {{ t('home.dueReviews.description', { count: dueReviews.length }) }}
        </h3>
      </div>

      <div class="home-queue-list neo-stagger-list">
        <article
          v-for="item in dueReviews"
          :key="item.id"
          class="home-queue-item"
        >
          <div class="space-y-1">
            <p class="text-sm font-black uppercase">
              {{ item.kindLabel }}
            </p>
            <p class="text-lg font-black">
              {{ item.headline }}
            </p>
            <p class="neo-note">{{ item.hint }}</p>
          </div>
          <div class="home-queue-actions">
            <RouterLink :to="item.href" class="neo-button-dark w-full">
              {{ t('home.dueReviews.startAction') }}
            </RouterLink>
            <button
              class="neo-button w-full bg-white text-xs"
              :aria-busy="isCompleting"
              @click="emit('complete-review', item.id)"
            >
              {{ t('home.dueReviews.markDone') }}
            </button>
          </div>
        </article>
      </div>
    </section>

    <section
      v-else-if="currentOnboardingStep"
      class="neo-panel home-queue-panel bg-[var(--neo-green)]"
    >
      <div class="space-y-2">
        <p class="neo-kicker bg-white">{{ t('home.onboarding.kicker') }}</p>
        <h3 class="home-section-title">
          {{ currentOnboardingStep.label }}
        </h3>
      </div>
      <p class="neo-note">{{ currentOnboardingStep.hint }}</p>
      <div class="flex items-center gap-3">
        <span class="neo-badge bg-white">
          {{ currentOnboardingStep.index }}
        </span>
        <span class="text-sm font-black uppercase tracking-[0.08em]">
          {{ t(`home.onboarding.status.${currentOnboardingStep.status}`) }}
        </span>
      </div>
      <RouterLink :to="currentOnboardingStep.href" class="neo-button-dark w-full">
        {{ currentOnboardingStep.label }}
      </RouterLink>
    </section>

    <section v-else class="neo-panel home-queue-panel bg-[var(--neo-blue)]">
      <div class="space-y-2">
        <p class="neo-kicker bg-white">{{ t('home.cards.trackKicker') }}</p>
        <h3 class="home-section-title">
          {{ t('home.cards.trackTitle') }}
        </h3>
      </div>
      <p class="neo-note">
        {{ recommendedTrack }}
      </p>
      <RouterLink to="/train" class="neo-button-dark w-full">
        {{ t('home.hero.actionPrimary') }}
      </RouterLink>
    </section>
  </aside>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

type HomeOnboardingStatus = 'done' | 'current' | 'next';

defineProps<{
  isCompleting: boolean;
  recommendedTrack: string;
  dueReviews: Array<{
    id: number;
    kindLabel: string;
    headline: string;
    hint: string;
    href: string;
  }>;
  currentOnboardingStep: {
    index: number;
    label: string;
    hint: string;
    href: string;
    status: HomeOnboardingStatus;
  } | null;
}>();

const emit = defineEmits<{
  (event: 'complete-review', reviewId: number): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
.home-detail-side {
  min-width: 0;
}

.home-queue-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.home-section-title {
  font-size: 1.5rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.home-queue-list {
  display: grid;
  gap: 0.85rem;
}

.home-queue-item {
  background: color-mix(in srgb, var(--neo-surface) 88%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.85rem;
  padding: 1rem;
  transition:
    transform var(--motion-duration-base) var(--motion-ease-standard),
    box-shadow var(--motion-duration-base) var(--motion-ease-standard);
}

.home-queue-item:hover {
  box-shadow: 8px 8px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(var(--motion-lift-md), var(--motion-lift-md));
}

.home-queue-actions {
  display: grid;
  gap: 0.65rem;
}

@media (min-width: 1280px) {
  .home-queue-panel {
    position: sticky;
    top: 1.5rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-queue-item {
    transition: none;
  }

  .home-queue-item:hover {
    box-shadow: inherit;
    transform: none;
  }
}
</style>
