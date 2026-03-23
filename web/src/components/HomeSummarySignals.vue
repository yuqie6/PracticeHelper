<template>
  <div class="home-stage-sidebar neo-stagger-list">
    <article class="neo-panel home-signal home-signal-blue">
      <p class="neo-kicker bg-white">{{ t('home.deadline.kicker') }}</p>
      <h3 class="home-signal-title">{{ t('home.deadline.title') }}</h3>
      <p class="home-signal-value">
        {{ daysUntilDeadline ?? '--' }}
      </p>
      <p class="home-signal-label">{{ t('common.daysRemainingLabel') }}</p>
      <p class="neo-note">{{ deadlineNote }}</p>
    </article>

    <article class="neo-panel home-signal home-signal-paper">
      <p class="neo-kicker bg-[var(--neo-green)]">
        {{ t('home.currentSession.kicker') }}
      </p>
      <template v-if="currentSession">
        <h3 class="home-signal-title">
          {{ t('home.currentSession.title') }}
        </h3>
        <p class="home-signal-copy">
          {{ currentSession.description }}
        </p>
        <p class="neo-note">
          {{ currentSession.updatedAtLabel }}
        </p>
        <p v-if="currentSession.jobTargetTitle" class="neo-note">
          {{
            t('home.currentSession.jobTargetDescription', {
              name: currentSession.jobTargetTitle,
            })
          }}
        </p>
        <RouterLink :to="currentSession.href" class="neo-button-dark mt-auto w-full">
          {{ t('common.resume') }}
        </RouterLink>
      </template>
      <template v-else>
        <h3 class="home-signal-title">
          {{ t('home.currentSession.emptyTitle') }}
        </h3>
        <p class="neo-note">
          {{ t('home.currentSession.emptyDescription') }}
        </p>
        <RouterLink to="/train" class="neo-button-dark mt-auto w-full">
          {{ t('home.hero.actionPrimary') }}
        </RouterLink>
      </template>
    </article>

    <article
      v-if="primaryDueReview"
      class="neo-panel home-signal home-signal-red"
    >
      <p class="neo-kicker bg-white">{{ t('home.dueReviews.kicker') }}</p>
      <h3 class="home-signal-title">
        {{ primaryDueReview.headline }}
      </h3>
      <p class="neo-note">
        {{ primaryDueReview.hint }}
      </p>
      <div class="home-signal-actions">
        <RouterLink :to="primaryDueReview.href" class="neo-button-dark w-full">
          {{ t('home.dueReviews.startAction') }}
        </RouterLink>
        <button
          class="neo-button w-full bg-white text-xs"
          :aria-busy="isCompleting"
          @click="emit('complete-review', primaryDueReview.id)"
        >
          {{ t('home.dueReviews.markDone') }}
        </button>
      </div>
    </article>

    <article v-else class="neo-panel home-signal home-signal-green">
      <p class="neo-kicker bg-white">
        {{ t('home.cards.jobTargetKicker') }}
      </p>
      <h3 class="home-signal-title">
        {{ t('home.cards.jobTargetTitle') }}
      </h3>
      <p class="home-signal-copy">
        {{ jobTargetSummary }}
      </p>
      <RouterLink to="/job-targets" class="neo-button-dark mt-auto w-full">
        {{ t('app.nav.jobs') }}
      </RouterLink>
    </article>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

defineProps<{
  daysUntilDeadline: number | null | undefined;
  deadlineNote: string;
  isCompleting: boolean;
  jobTargetSummary: string;
  currentSession: {
    description: string;
    updatedAtLabel: string;
    jobTargetTitle?: string;
    href: string;
  } | null;
  primaryDueReview: {
    id: number;
    headline: string;
    hint: string;
    href: string;
  } | null;
}>();

const emit = defineEmits<{
  (event: 'complete-review', reviewId: number): void;
}>();

const { t } = useI18n();
</script>

<style scoped>
.home-stage-sidebar {
  display: grid;
  gap: 1rem;
}

.home-signal {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-height: 0;
}

.home-signal-blue {
  background: linear-gradient(
    160deg,
    color-mix(in srgb, var(--neo-blue) 82%, white) 0%,
    color-mix(in srgb, var(--neo-blue) 58%, var(--neo-paper)) 100%
  );
}

.home-signal-paper {
  background: var(--neo-paper);
}

.home-signal-red {
  background: linear-gradient(
    160deg,
    color-mix(in srgb, var(--neo-red) 72%, white) 0%,
    color-mix(in srgb, var(--neo-red) 46%, var(--neo-paper)) 100%
  );
}

.home-signal-green {
  background: linear-gradient(
    160deg,
    color-mix(in srgb, var(--neo-green) 78%, white) 0%,
    color-mix(in srgb, var(--neo-green) 52%, var(--neo-paper)) 100%
  );
}

.home-signal-title {
  font-size: 1.2rem;
  font-weight: 900;
  letter-spacing: 0.04em;
  line-height: 1.15;
  margin: 0;
  text-transform: uppercase;
}

.home-signal-value {
  font-size: clamp(3rem, 6vw, 4.8rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
  margin: 0;
}

.home-signal-label {
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  margin: 0;
  text-transform: uppercase;
}

.home-signal-copy {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
}

.home-signal-actions {
  display: grid;
  gap: 0.65rem;
  margin-top: auto;
}
</style>
