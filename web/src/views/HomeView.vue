<template>
  <section class="neo-page space-y-6">
    <div class="neo-grid md:grid-cols-[1.35fr_0.65fr]">
      <div class="neo-panel bg-[var(--neo-yellow)]">
        <p class="neo-kicker bg-white">{{ t('home.hero.kicker') }}</p>
        <h2 class="neo-heading">{{ t('home.hero.title') }}</h2>
        <p class="mt-4 max-w-3xl text-base font-semibold leading-7">
          {{ dashboard?.today_focus ?? t('common.firstTrainingHint') }}
        </p>
        <div class="mt-6 flex flex-wrap gap-3">
          <RouterLink to="/train" class="neo-button-red">
            {{ t('home.hero.actionPrimary') }}
          </RouterLink>
          <RouterLink to="/projects" class="neo-button bg-white">
            {{ t('home.hero.actionSecondary') }}
          </RouterLink>
        </div>
      </div>

      <div class="neo-panel bg-[var(--neo-blue)]">
        <p class="neo-kicker bg-white">{{ t('home.deadline.kicker') }}</p>
        <div class="space-y-3">
          <p class="text-6xl font-black">{{ dashboard?.days_until_deadline ?? '--' }}</p>
          <p class="text-sm font-bold uppercase tracking-[0.08em]">
            {{ t('common.daysRemainingLabel') }}
          </p>
          <p class="neo-note">
            {{ dashboard?.recommended_track ?? t('common.nextRecommendationHint') }}
          </p>
        </div>
      </div>
    </div>

    <div class="neo-grid md:grid-cols-2 xl:grid-cols-4">
      <StatCard
        :kicker="t('home.cards.weaknessKicker')"
        kicker-class="bg-[var(--neo-red)]"
        :title="t('home.cards.weaknessTitle')"
        :description="weaknessSummary"
      />
      <StatCard
        :kicker="t('home.cards.sessionKicker')"
        kicker-class="bg-[var(--neo-green)]"
        :title="t('home.cards.sessionTitle')"
        :description="sessionSummary"
      />
      <StatCard
        :kicker="t('home.cards.trackKicker')"
        kicker-class="bg-[var(--neo-yellow)]"
        :title="t('home.cards.trackTitle')"
        :description="dashboard?.recommended_track ?? t('common.noRecommendation')"
      />
      <StatCard
        :kicker="t('home.cards.profileKicker')"
        kicker-class="bg-[var(--neo-blue)]"
        :title="t('home.cards.profileTitle')"
        :description="profileSummary"
      />
    </div>

    <div class="neo-grid lg:grid-cols-[0.9fr_1.1fr]">
      <div class="neo-panel">
        <p class="neo-kicker bg-[var(--neo-red)]">{{ t('home.sections.weaknesses') }}</p>
        <ul class="space-y-3">
          <li
            v-for="item in dashboard?.weaknesses ?? []"
            :key="item.id"
            class="flex items-center justify-between border-2 border-black bg-white px-3 py-3 md:border-4"
          >
            <div>
              <p class="text-sm font-black uppercase">{{ formatWeaknessKindLabel(t, item.kind) }}</p>
              <p class="text-lg font-bold">{{ item.label }}</p>
            </div>
            <span class="neo-badge bg-[var(--neo-yellow)]">
              {{ t('common.severity', { value: item.severity.toFixed(2) }) }}
            </span>
          </li>
          <li v-if="!dashboard?.weaknesses.length" class="neo-note">
            {{ t('home.sections.weaknessesEmpty') }}
          </li>
        </ul>
      </div>

      <div class="neo-panel">
        <p class="neo-kicker bg-[var(--neo-green)]">{{ t('home.sections.sessions') }}</p>
        <ul class="space-y-3">
          <li
            v-for="session in dashboard?.recent_sessions ?? []"
            :key="session.id"
            class="border-2 border-black bg-white p-4 md:border-4"
          >
            <div class="flex flex-wrap items-center justify-between gap-3">
              <div>
                <p class="text-sm font-black uppercase">
                  {{ formatModeLabel(t, session.mode) }}
                </p>
                <p class="text-lg font-bold">
                  {{ session.project_name || formatSessionName(session) }}
                </p>
              </div>
              <span class="neo-badge bg-[var(--neo-blue)]">{{ session.total_score.toFixed(1) }}</span>
            </div>
            <p class="mt-2 text-sm font-semibold">
              {{ formatStatusLabel(t, session.status) }}
            </p>
          </li>
          <li v-if="!dashboard?.recent_sessions.length" class="neo-note">
            {{ t('home.sections.sessionsEmpty') }}
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

import { getDashboard, type TrainingSessionSummary } from '../api/client';
import StatCard from '../components/StatCard.vue';
import { describeProfile, describeSession, describeWeakness } from '../lib/dashboard';
import {
  formatModeLabel,
  formatStatusLabel,
  formatTopicLabel,
  formatWeaknessKindLabel,
} from '../lib/labels';

const { data } = useQuery({
  queryKey: ['dashboard'],
  queryFn: getDashboard,
});

const { t, locale } = useI18n();

const dashboard = computed(() => data.value ?? null);
const weaknessSummary = computed(() => {
  locale.value;
  return describeWeakness(dashboard.value, t);
});
const sessionSummary = computed(() => {
  locale.value;
  return describeSession(dashboard.value, t);
});
const profileSummary = computed(() => {
  locale.value;
  return describeProfile(dashboard.value, t);
});

function formatSessionName(session: TrainingSessionSummary): string {
  if (session.topic) {
    return formatTopicLabel(t, session.topic);
  }

  if (session.mode) {
    return formatModeLabel(t, session.mode);
  }

  return t('common.unnamedSession');
}
</script>
