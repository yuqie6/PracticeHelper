<template>
  <section class="neo-page space-y-6">
    <div
      v-if="showOnboarding"
      class="neo-panel space-y-4 bg-[var(--neo-yellow)]"
    >
      <p class="neo-kicker bg-white">{{ t('home.onboarding.kicker') }}</p>
      <h2 class="text-xl font-black">{{ t('home.onboarding.title') }}</h2>
      <p class="text-sm font-semibold">
        {{ t('home.onboarding.description') }}
      </p>
      <ol class="space-y-3">
        <li
          v-for="step in onboardingSteps"
          :key="step.key"
          class="border-2 border-black md:border-4"
          :class="onboardingStepClass(step.status)"
        >
          <RouterLink
            :to="step.href"
            class="flex flex-col gap-3 px-4 py-4 sm:flex-row sm:items-center sm:justify-between"
          >
            <div class="flex items-start gap-3">
              <span class="neo-badge bg-white">
                {{ step.index }}
              </span>
              <div class="space-y-1">
                <p class="text-base font-black">
                  {{ step.label }}
                </p>
                <p class="neo-note">
                  {{ step.hint }}
                </p>
              </div>
            </div>
            <span class="text-xs font-black uppercase tracking-[0.08em]">
              {{ t(`home.onboarding.status.${step.status}`) }}
            </span>
          </RouterLink>
        </li>
      </ol>
    </div>

    <div class="neo-grid xl:grid-cols-[1.3fr_0.7fr]">
      <div class="neo-panel-hero bg-[var(--neo-yellow)]">
        <p class="neo-kicker bg-white">{{ t('home.hero.kicker') }}</p>
        <p class="text-lg font-bold leading-7">
          {{ dashboard?.today_focus ?? t('common.firstTrainingHint') }}
        </p>
        <div class="mt-4 flex flex-col gap-3 sm:flex-row sm:flex-wrap">
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
      </div>

      <div class="neo-grid">
        <div class="neo-panel bg-[var(--neo-blue)]">
          <p class="neo-kicker bg-white">{{ t('home.deadline.kicker') }}</p>
          <h3 class="text-xl font-black uppercase tracking-[0.06em]">
            {{ t('home.deadline.title') }}
          </h3>
          <div class="mt-4 space-y-3">
            <p class="text-6xl font-black">
              {{ dashboard?.days_until_deadline ?? '--' }}
            </p>
            <p class="text-sm font-bold uppercase tracking-[0.08em]">
              {{ t('common.daysRemainingLabel') }}
            </p>
            <p class="neo-note">
              {{
                dashboard?.days_until_deadline == null
                  ? t('common.setDeadlineHint')
                  : dashboard?.recommended_track
              }}
            </p>
          </div>
        </div>

        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('home.currentSession.kicker') }}
          </p>
          <template v-if="currentSession">
            <h3 class="text-xl font-black uppercase tracking-[0.06em]">
              {{ t('home.currentSession.title') }}
            </h3>
            <p class="mt-3 text-base font-semibold">
              {{
                t('home.currentSession.description', {
                  name: formatSessionName(currentSession),
                  status: formatStatusLabel(t, currentSession.status),
                })
              }}
            </p>
            <p class="neo-note mt-3">
              {{
                t('common.lastUpdated', {
                  value: formatUpdatedAt(currentSession.updated_at),
                })
              }}
            </p>
            <p v-if="currentSession.job_target" class="neo-note mt-2">
              {{
                t('home.currentSession.jobTargetDescription', {
                  name: currentSession.job_target.title,
                })
              }}
            </p>
            <RouterLink
              :to="buildSessionTarget(currentSession)"
              class="neo-button-dark mt-4"
            >
              {{ t('common.resume') }}
            </RouterLink>
          </template>
          <template v-else>
            <h3 class="text-xl font-black uppercase tracking-[0.06em]">
              {{ t('home.currentSession.emptyTitle') }}
            </h3>
            <p class="neo-note mt-3">
              {{ t('home.currentSession.emptyDescription') }}
            </p>
          </template>
        </div>
      </div>
    </div>

    <div v-if="dueReviews.length" class="neo-panel bg-[var(--neo-yellow)]">
      <p class="neo-kicker bg-white">{{ t('home.dueReviews.kicker') }}</p>
      <p class="text-sm font-semibold">
        {{ t('home.dueReviews.description', { count: dueReviews.length }) }}
      </p>
      <div class="mt-3 grid gap-3 sm:flex sm:flex-wrap">
        <div
          v-for="item in dueReviews.slice(0, 5)"
          :key="item.id"
          class="border-2 border-black bg-white px-3 py-3 md:border-4"
        >
          <div
            class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between"
          >
            <div class="space-y-1">
              <p class="text-sm font-black uppercase">
                {{
                  item.weakness_kind
                    ? formatWeaknessKindLabel(t, item.weakness_kind)
                    : t('home.dueReviews.review')
                }}
              </p>
              <p class="text-base font-bold">
                {{
                  resolveDueReviewHeadline(item) ||
                  (item.topic
                    ? formatTopicLabel(t, item.topic)
                    : t('home.dueReviews.review'))
                }}
              </p>
              <p class="neo-note">
                {{
                  item.topic
                    ? t('home.dueReviews.topicHint', {
                        topic: formatTopicLabel(t, item.topic),
                      })
                    : t('home.dueReviews.genericHint')
                }}
              </p>
            </div>
            <div class="flex flex-col gap-2 sm:w-auto sm:min-w-[10rem]">
              <RouterLink
                :to="buildDueReviewTarget(item)"
                class="neo-button-dark w-full"
              >
                {{ t('home.dueReviews.startAction') }}
              </RouterLink>
              <button
                class="neo-button w-full bg-white text-xs"
                :aria-busy="completeMutation.isPending.value"
                @click="completeMutation.mutate(item.id)"
              >
                {{ t('home.dueReviews.markDone') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="neo-grid md:grid-cols-2 xl:grid-cols-4">
      <StatCard
        :kicker="t('home.cards.weaknessKicker')"
        kicker-class="bg-[var(--neo-red)]"
        :title="t('home.cards.weaknessTitle')"
        :value="String(dashboard?.weaknesses?.length ?? 0)"
        :description="weaknessSummary"
      />
      <StatCard
        :kicker="t('home.cards.trackKicker')"
        kicker-class="bg-[var(--neo-yellow)]"
        :title="t('home.cards.trackTitle')"
        :description="
          dashboard?.recommended_track ?? t('common.noRecommendation')
        "
      />
      <StatCard
        :kicker="t('home.cards.jobTargetKicker')"
        kicker-class="bg-[var(--neo-green)]"
        :title="t('home.cards.jobTargetTitle')"
        :description="jobTargetSummary"
      />
      <StatCard
        :kicker="t('home.cards.profileKicker')"
        kicker-class="bg-[var(--neo-blue)]"
        :title="t('home.cards.profileTitle')"
        :value="String(dashboard?.recent_sessions?.length ?? 0)"
        :description="profileSummary"
      />
    </div>

    <div class="neo-grid lg:grid-cols-[0.9fr_1.1fr]">
      <div class="neo-panel">
        <p class="neo-kicker bg-[var(--neo-red)]">
          {{ t('home.sections.weaknesses') }}
        </p>
        <ul class="neo-stagger-list space-y-3">
          <li
            v-for="item in dashboard?.weaknesses ?? []"
            :key="item.id"
            class="flex items-center justify-between border-2 border-black bg-white px-3 py-3 md:border-4"
          >
            <div>
              <p class="text-sm font-black uppercase">
                {{ formatWeaknessKindLabel(t, item.kind) }}
              </p>
              <p class="text-lg font-bold">{{ item.label }}</p>
            </div>
            <span class="neo-badge bg-[var(--neo-yellow)]">
              {{ t('common.severity', { value: item.severity.toFixed(2) }) }}
            </span>
          </li>
          <li
            v-if="!dashboard?.weaknesses.length"
            class="space-y-2 border-2 border-black bg-[var(--neo-paper)] px-4 py-4 md:border-4"
          >
            <p class="neo-note">{{ t('home.sections.weaknessesEmpty') }}</p>
            <RouterLink to="/train" class="neo-button-dark text-xs">
              {{ t('home.hero.actionPrimary') }}
            </RouterLink>
          </li>
        </ul>
        <div v-if="trends.length" class="mt-4 space-y-2">
          <p class="neo-subheading">{{ t('home.sections.weaknessTrends') }}</p>
          <div
            v-for="trend in trends"
            :key="trend.id"
            class="flex flex-col items-start gap-3 border-2 border-black bg-white px-3 py-2 sm:flex-row sm:items-center md:border-4"
          >
            <span class="w-24 truncate text-xs font-bold">{{
              trend.label
            }}</span>
            <svg
              v-if="trend.points.length >= 2"
              viewBox="0 0 160 40"
              class="h-10 w-40 flex-shrink-0"
              preserveAspectRatio="none"
            >
              <path
                :d="sparklinePath(trend)"
                fill="none"
                stroke="var(--neo-red)"
                stroke-width="2"
              />
              <circle
                :cx="sparklineEndpoints(trend).first.x"
                :cy="sparklineEndpoints(trend).first.y"
                r="3"
                fill="var(--neo-red)"
              />
              <circle
                :cx="sparklineEndpoints(trend).last.x"
                :cy="sparklineEndpoints(trend).last.y"
                r="3"
                fill="var(--neo-red)"
              />
            </svg>
            <span v-else class="text-xs text-gray-400">—</span>
          </div>
        </div>
      </div>

      <div class="neo-panel">
        <div class="flex items-center justify-between gap-3">
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('home.sections.sessions') }}
          </p>
          <p class="text-sm font-semibold">{{ sessionSummary }}</p>
        </div>
        <ul class="neo-stagger-list space-y-3">
          <li
            v-for="session in dashboard?.recent_sessions ?? []"
            :key="session.id"
          >
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
                <span class="neo-badge bg-[var(--neo-blue)]">{{
                  session.total_score.toFixed(1)
                }}</span>
              </div>
              <div
                class="mt-3 flex flex-wrap items-center justify-between gap-3 text-sm font-semibold"
              >
                <span>{{ formatStatusLabel(t, session.status) }}</span>
                <span>{{ formatUpdatedAt(session.updated_at) }}</span>
              </div>
              <p v-if="session.job_target" class="neo-note mt-2">
                {{
                  t('home.sessions.jobTargetDescription', {
                    name: session.job_target.title,
                  })
                }}
              </p>
            </RouterLink>
          </li>
          <li
            v-if="!dashboard?.recent_sessions.length"
            class="space-y-2 border-2 border-black bg-[var(--neo-paper)] px-4 py-4 md:border-4"
          >
            <p class="neo-note">{{ t('home.sections.sessionsEmpty') }}</p>
            <RouterLink to="/train" class="neo-button-dark text-xs">
              {{ t('home.hero.actionPrimary') }}
            </RouterLink>
          </li>
        </ul>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

import {
  completeDueReview,
  getDashboard,
  getWeaknessTrends,
  listDueReviews,
  listProjects,
  type TrainingSessionSummary,
  type WeaknessTrend,
} from '../api/client';
import StatCard from '../components/StatCard.vue';
import {
  buildDueReviewTarget,
  resolveDueReviewHeadline,
} from '../lib/dueReviews';
import {
  describeProfile,
  describeSession,
  describeWeakness,
} from '../lib/dashboard';
import { describeJobTargetStatus } from '../lib/jobTargetStatus';
import {
  buildOnboardingHref,
  buildOnboardingProgress,
  type OnboardingStepStatus,
} from '../lib/onboarding';
import {
  formatModeLabel,
  formatStatusLabel,
  formatTopicLabel,
  formatWeaknessKindLabel,
} from '../lib/labels';
import { useToast } from '../lib/useToast';

const { data } = useQuery({
  queryKey: ['dashboard'],
  queryFn: getDashboard,
});

const { data: trendsData } = useQuery({
  queryKey: ['weakness-trends'],
  queryFn: getWeaknessTrends,
});

const { data: dueReviewsData } = useQuery({
  queryKey: ['due-reviews'],
  queryFn: listDueReviews,
});

const { data: projectsData } = useQuery({
  queryKey: ['projects'],
  queryFn: listProjects,
});

const { t, locale } = useI18n();
const queryClient = useQueryClient();
const { show: showToast } = useToast();

const dashboard = computed(() => data.value ?? null);
const trends = computed(() => trendsData.value ?? []);
const dueReviews = computed(() => dueReviewsData.value ?? []);
const projects = computed(() => projectsData.value ?? []);
const completeMutation = useMutation({
  mutationFn: (id: number) => completeDueReview(id),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: ['due-reviews'] });
    showToast(t('common.operationSuccess'), 'success');
  },
});
const currentSession = computed(() => dashboard.value?.current_session ?? null);
const hasProfile = computed(() => Boolean(dashboard.value?.profile));
const hasStartedSession = computed(
  () =>
    Boolean(dashboard.value?.current_session) ||
    Boolean(dashboard.value?.recent_sessions?.length),
);
const onboarding = computed(() =>
  buildOnboardingProgress({
    hasProfile: hasProfile.value,
    hasProjects: projects.value.length > 0,
    hasSession: hasStartedSession.value,
  }),
);
const showOnboarding = computed(() => !onboarding.value.completed);
const onboardingSteps = computed(() =>
  onboarding.value.steps.map((step, index) => ({
    ...step,
    index: index + 1,
    label: t(`home.onboarding.steps.${step.key}.label`),
    hint: t(`home.onboarding.steps.${step.key}.hint`),
    href: buildOnboardingHref(step.key),
  })),
);
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
const jobTargetSummary = computed(() => {
  const active = dashboard.value?.active_job_target;
  if (!active) {
    return t('home.cards.jobTargetEmpty');
  }
  if (dashboard.value?.recommendation_scope === 'job_target') {
    return t('home.cards.jobTargetScoped', { name: active.title });
  }
  return describeJobTargetStatus(
    t,
    'homeActive',
    active.latest_analysis_status,
    {
      name: active.title,
    },
  );
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

function sparklinePath(trend: WeaknessTrend): string {
  const pts = trend.points;
  if (pts.length < 2) return '';
  const w = 160;
  const h = 40;
  const maxSev = Math.max(...pts.map((p) => p.severity), 0.1);
  const step = w / (pts.length - 1);
  return pts
    .map((p, i) => {
      const x = i * step;
      const y = h - (p.severity / maxSev) * (h - 8) - 4;
      return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(' ');
}

function sparklineEndpoints(trend: WeaknessTrend) {
  const pts = trend.points;
  const w = 160;
  const h = 40;
  const maxSev = Math.max(...pts.map((p) => p.severity), 0.1);
  const step = w / (pts.length - 1);
  const toY = (sev: number) => h - (sev / maxSev) * (h - 8) - 4;
  return {
    first: { x: 0, y: toY(pts[0].severity) },
    last: {
      x: (pts.length - 1) * step,
      y: toY(pts[pts.length - 1].severity),
    },
  };
}

function onboardingStepClass(status: OnboardingStepStatus): string {
  switch (status) {
    case 'done':
      return 'bg-[var(--neo-green)]';
    case 'current':
      return 'bg-white';
    default:
      return 'bg-[var(--neo-paper)]';
  }
}
</script>
