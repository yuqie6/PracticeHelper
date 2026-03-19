<template>
  <section class="neo-page space-y-6">
    <div class="neo-grid xl:grid-cols-[1.3fr_0.7fr]">
      <div class="neo-panel bg-[var(--neo-yellow)]">
        <p class="neo-kicker bg-white">{{ t('home.hero.kicker') }}</p>
        <p class="text-lg font-bold leading-7">
          {{ dashboard?.today_focus ?? t('common.firstTrainingHint') }}
        </p>
        <div class="mt-4 flex flex-wrap gap-3">
          <RouterLink to="/train" class="neo-button-red">
            {{ t('home.hero.actionPrimary') }}
          </RouterLink>
          <RouterLink to="/projects" class="neo-button bg-white">
            {{ t('home.hero.actionSecondary') }}
          </RouterLink>
        </div>
      </div>

      <div class="neo-grid">
        <div class="neo-panel bg-[var(--neo-blue)]">
          <p class="neo-kicker bg-white">{{ t('home.deadline.kicker') }}</p>
          <h3 class="text-xl font-black uppercase tracking-[0.06em]">
            {{ t('home.deadline.title') }}
          </h3>
          <div class="mt-4 space-y-3">
            <p class="text-6xl font-black">{{ dashboard?.days_until_deadline ?? '--' }}</p>
            <p class="text-sm font-bold uppercase tracking-[0.08em]">
              {{ t('common.daysRemainingLabel') }}
            </p>
            <p class="neo-note">
              {{ dashboard?.days_until_deadline == null ? t('common.setDeadlineHint') : dashboard?.recommended_track }}
            </p>
          </div>
        </div>

        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-green)]">{{ t('home.currentSession.kicker') }}</p>
          <template v-if="currentSession">
            <h3 class="text-xl font-black uppercase tracking-[0.06em]">
              {{ t('home.currentSession.title') }}
            </h3>
            <p class="mt-3 text-base font-semibold">
              {{ t('home.currentSession.description', { name: formatSessionName(currentSession), status: formatStatusLabel(t, currentSession.status) }) }}
            </p>
            <p class="neo-note mt-3">
              {{ t('common.lastUpdated', { value: formatUpdatedAt(currentSession.updated_at) }) }}
            </p>
            <RouterLink :to="buildSessionTarget(currentSession)" class="neo-button-dark mt-4">
              {{ t('common.resume') }}
            </RouterLink>
          </template>
          <template v-else>
            <h3 class="text-xl font-black uppercase tracking-[0.06em]">
              {{ t('home.currentSession.emptyTitle') }}
            </h3>
            <p class="mt-3 neo-note">
              {{ t('home.currentSession.emptyDescription') }}
            </p>
          </template>
        </div>
      </div>
    </div>

    <div class="neo-grid md:grid-cols-3">
      <StatCard
        :kicker="t('home.cards.weaknessKicker')"
        kicker-class="bg-[var(--neo-red)]"
        :title="t('home.cards.weaknessTitle')"
        :description="weaknessSummary"
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
        <div class="flex items-center justify-between gap-3">
          <p class="neo-kicker bg-[var(--neo-green)]">{{ t('home.sections.sessions') }}</p>
          <p class="text-sm font-semibold">{{ sessionSummary }}</p>
        </div>
        <ul class="space-y-3">
          <li v-for="session in dashboard?.recent_sessions ?? []" :key="session.id">
            <RouterLink :to="buildSessionTarget(session)" class="neo-link-card">
              <div class="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <p class="text-sm font-black uppercase">
                    {{ formatModeLabel(t, session.mode) }}
                  </p>
                  <p class="text-lg font-bold">
                    {{ formatSessionName(session) }}
                  </p>
                </div>
                <span class="neo-badge bg-[var(--neo-blue)]">{{ session.total_score.toFixed(1) }}</span>
              </div>
              <div class="mt-3 flex flex-wrap items-center justify-between gap-3 text-sm font-semibold">
                <span>{{ formatStatusLabel(t, session.status) }}</span>
                <span>{{ formatUpdatedAt(session.updated_at) }}</span>
              </div>
            </RouterLink>
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
const currentSession = computed(() => dashboard.value?.current_session ?? null);
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
  if (session.project_name) {
    return session.project_name;
  }

  if (session.topic) {
    return formatTopicLabel(t, session.topic);
  }

  if (session.mode) {
    return formatModeLabel(t, session.mode);
  }

  return t('common.unknownSession');
}

function buildSessionTarget(session: TrainingSessionSummary): string {
  if (session.status === 'completed' && session.review_id) {
    return `/reviews/${session.review_id}`;
  }

  return `/sessions/${session.id}`;
}

function formatUpdatedAt(raw: string): string {
  const parsed = new Date(raw);
  if (Number.isNaN(parsed.getTime())) {
    return raw;
  }

  return new Intl.DateTimeFormat(locale.value, {
    month: 'numeric',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(parsed);
}
</script>
