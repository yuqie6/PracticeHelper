<template>
  <section class="neo-page home-page space-y-6 xl:space-y-8">
    <div class="home-stage-grid">
      <section class="neo-panel-hero home-stage-main bg-[var(--neo-yellow)]">
        <div class="home-stage-copy">
          <p class="neo-kicker bg-white">{{ t('home.hero.kicker') }}</p>
          <h2 class="home-stage-title">
            {{ dashboard?.today_focus ?? t('common.firstTrainingHint') }}
          </h2>
          <p class="home-stage-support">
            {{ heroSupport }}
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

          <div v-if="showOnboarding" class="home-stage-steps">
            <RouterLink
              v-for="step in onboardingSteps"
              :key="step.key"
              :to="step.href"
              class="home-stage-step"
              :class="onboardingStepClass(step.status)"
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

      <div class="home-stage-sidebar">
        <article class="neo-panel home-signal home-signal-blue">
          <p class="neo-kicker bg-white">{{ t('home.deadline.kicker') }}</p>
          <h3 class="home-signal-title">{{ t('home.deadline.title') }}</h3>
          <p class="home-signal-value">
            {{ dashboard?.days_until_deadline ?? '--' }}
          </p>
          <p class="home-signal-label">{{ t('common.daysRemainingLabel') }}</p>
          <p class="neo-note">
            {{
              dashboard?.days_until_deadline == null
                ? t('common.setDeadlineHint')
                : dashboard?.recommended_track
            }}
          </p>
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
              {{
                t('home.currentSession.description', {
                  name: formatSessionName(currentSession),
                  status: formatStatusLabel(t, currentSession.status),
                })
              }}
            </p>
            <p class="neo-note">
              {{
                t('common.lastUpdated', {
                  value: formatUpdatedAt(currentSession.updated_at),
                })
              }}
            </p>
            <p v-if="currentSession.job_target" class="neo-note">
              {{
                t('home.currentSession.jobTargetDescription', {
                  name: currentSession.job_target.title,
                })
              }}
            </p>
            <RouterLink
              :to="buildSessionTarget(currentSession)"
              class="neo-button-dark mt-auto w-full"
            >
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
            {{ formatDueReviewHeadline(primaryDueReview) }}
          </h3>
          <p class="neo-note">
            {{ formatDueReviewHint(primaryDueReview) }}
          </p>
          <div class="home-signal-actions">
            <RouterLink
              :to="buildDueReviewTarget(primaryDueReview)"
              class="neo-button-dark w-full"
            >
              {{ t('home.dueReviews.startAction') }}
            </RouterLink>
            <button
              class="neo-button w-full bg-white text-xs"
              :aria-busy="completeMutation.isPending.value"
              @click="completeMutation.mutate(primaryDueReview.id)"
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
    </div>

    <section class="neo-panel home-metric-strip">
      <div class="home-metric-grid">
        <article
          v-for="item in summaryTiles"
          :key="item.kicker"
          class="home-metric-cell"
        >
          <p class="neo-kicker" :class="item.kickerClass">{{ item.kicker }}</p>
          <p v-if="item.value" class="home-metric-value">{{ item.value }}</p>
          <h3 class="home-metric-title">{{ item.title }}</h3>
          <p class="neo-note home-metric-note">{{ item.description }}</p>
        </article>
      </div>
    </section>

    <div class="home-detail-grid">
      <div class="home-detail-main">
        <section class="neo-panel home-section">
          <div class="home-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-red)]">
                {{ t('home.sections.weaknesses') }}
              </p>
              <h3 class="home-section-title">
                {{ t('home.cards.weaknessTitle') }}
              </h3>
            </div>
            <p class="neo-note home-section-summary">{{ weaknessSummary }}</p>
          </div>

          <ul class="home-list">
            <li
              v-for="item in dashboard?.weaknesses ?? []"
              :key="item.id"
              class="home-list-row"
            >
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div class="space-y-1">
                  <p class="text-sm font-black uppercase">
                    {{ formatWeaknessKindLabel(t, item.kind) }}
                  </p>
                  <p class="text-xl font-black">
                    {{ item.label }}
                  </p>
                </div>
                <span class="neo-badge bg-[var(--neo-yellow)]">
                  {{
                    t('common.severity', { value: item.severity.toFixed(2) })
                  }}
                </span>
              </div>
            </li>
            <li v-if="!dashboard?.weaknesses.length" class="home-empty-row">
              <p class="neo-note">{{ t('home.sections.weaknessesEmpty') }}</p>
              <RouterLink to="/train" class="neo-button-dark text-xs">
                {{ t('home.hero.actionPrimary') }}
              </RouterLink>
            </li>
          </ul>

          <div v-if="trends.length" class="home-trend-grid">
            <div
              v-for="trend in trends"
              :key="trend.id"
              class="home-trend-card"
            >
              <div class="flex items-center justify-between gap-3">
                <p class="text-sm font-black uppercase">{{ trend.label }}</p>
                <span class="neo-badge bg-white">
                  {{ t('home.sections.weaknessTrends') }}
                </span>
              </div>
              <svg
                v-if="trend.points.length >= 2"
                viewBox="0 0 160 40"
                class="h-10 w-full"
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
        </section>

        <section class="neo-panel home-section">
          <div class="home-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-green)]">
                {{ t('home.sections.sessions') }}
              </p>
              <h3 class="home-section-title">
                {{ t('home.cards.sessionTitle') }}
              </h3>
            </div>
            <p class="neo-note home-section-summary">{{ sessionSummary }}</p>
          </div>

          <ul class="home-list">
            <li
              v-for="session in dashboard?.recent_sessions ?? []"
              :key="session.id"
              class="home-list-row"
            >
              <RouterLink
                :to="buildSessionTarget(session)"
                class="home-link-row"
              >
                <div class="flex flex-wrap items-start justify-between gap-3">
                  <div class="space-y-1">
                    <p class="text-sm font-black uppercase">
                      {{ formatModeLabel(t, session.mode) }}
                    </p>
                    <p class="text-xl font-black">
                      {{ formatSessionName(session) }}
                    </p>
                  </div>
                  <span class="neo-badge bg-[var(--neo-blue)]">
                    {{ session.total_score.toFixed(1) }}
                  </span>
                </div>
                <div
                  class="mt-4 flex flex-wrap items-center justify-between gap-3 text-sm font-semibold"
                >
                  <span>{{ formatStatusLabel(t, session.status) }}</span>
                  <span>{{ formatUpdatedAt(session.updated_at) }}</span>
                </div>
                <p v-if="session.job_target" class="neo-note mt-3">
                  {{
                    t('home.sections.jobTargetDescription', {
                      name: session.job_target.title,
                    })
                  }}
                </p>
              </RouterLink>
            </li>
            <li
              v-if="!dashboard?.recent_sessions.length"
              class="home-empty-row"
            >
              <p class="neo-note">{{ t('home.sections.sessionsEmpty') }}</p>
              <RouterLink to="/train" class="neo-button-dark text-xs">
                {{ t('home.hero.actionPrimary') }}
              </RouterLink>
            </li>
          </ul>
        </section>
      </div>

      <aside class="home-detail-side">
        <section
          v-if="dueReviews.length"
          class="neo-panel home-queue-panel bg-[var(--neo-yellow)]"
        >
          <div class="space-y-2">
            <p class="neo-kicker bg-white">{{ t('home.dueReviews.kicker') }}</p>
            <h3 class="home-section-title">
              {{
                t('home.dueReviews.description', { count: dueReviews.length })
              }}
            </h3>
          </div>

          <div class="home-queue-list">
            <article
              v-for="item in dueReviews.slice(0, 5)"
              :key="item.id"
              class="home-queue-item"
            >
              <div class="space-y-1">
                <p class="text-sm font-black uppercase">
                  {{
                    item.weakness_kind
                      ? formatWeaknessKindLabel(t, item.weakness_kind)
                      : t('home.dueReviews.review')
                  }}
                </p>
                <p class="text-lg font-black">
                  {{ formatDueReviewHeadline(item) }}
                </p>
                <p class="neo-note">{{ formatDueReviewHint(item) }}</p>
              </div>
              <div class="home-queue-actions">
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
          <RouterLink
            :to="currentOnboardingStep.href"
            class="neo-button-dark w-full"
          >
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
            {{ dashboard?.recommended_track ?? t('common.noRecommendation') }}
          </p>
          <RouterLink to="/train" class="neo-button-dark w-full">
            {{ t('home.hero.actionPrimary') }}
          </RouterLink>
        </section>
      </aside>
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
  type ReviewScheduleItem,
  type TrainingSessionSummary,
  type WeaknessTrend,
} from '../api/client';
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
const primaryDueReview = computed(() => dueReviews.value[0] ?? null);
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
const currentOnboardingStep = computed(
  () => onboardingSteps.value.find((step) => step.status !== 'done') ?? null,
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
const heroSupport = computed(() => {
  locale.value;
  return (
    dashboard.value?.recommended_track ?? t('common.nextRecommendationHint')
  );
});
const deadlineDisplay = computed(() => {
  const days = dashboard.value?.days_until_deadline;
  return days == null ? '--' : String(days).padStart(2, '0');
});
const dueReviewDisplay = computed(() =>
  String(dueReviews.value.length).padStart(2, '0'),
);
const summaryTiles = computed(() => {
  locale.value;
  return [
    {
      kicker: t('home.cards.weaknessKicker'),
      kickerClass: 'bg-[var(--neo-red)]',
      title: t('home.cards.weaknessTitle'),
      value: String(dashboard.value?.weaknesses?.length ?? 0),
      description: weaknessSummary.value,
    },
    {
      kicker: t('home.cards.trackKicker'),
      kickerClass: 'bg-[var(--neo-yellow)]',
      title: t('home.cards.trackTitle'),
      description:
        dashboard.value?.recommended_track ?? t('common.noRecommendation'),
    },
    {
      kicker: t('home.cards.jobTargetKicker'),
      kickerClass: 'bg-[var(--neo-green)]',
      title: t('home.cards.jobTargetTitle'),
      description: jobTargetSummary.value,
    },
    {
      kicker: t('home.cards.profileKicker'),
      kickerClass: 'bg-[var(--neo-blue)]',
      title: t('home.cards.profileTitle'),
      description: profileSummary.value,
    },
  ];
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

function formatDueReviewHeadline(item: ReviewScheduleItem): string {
  return (
    resolveDueReviewHeadline(item) ||
    (item.topic ? formatTopicLabel(t, item.topic) : t('home.dueReviews.review'))
  );
}

function formatDueReviewHint(item: ReviewScheduleItem): string {
  if (item.topic) {
    return t('home.dueReviews.topicHint', {
      topic: formatTopicLabel(t, item.topic),
    });
  }

  return t('home.dueReviews.genericHint');
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

<style scoped>
.home-page {
  position: relative;
}

.home-stage-grid {
  display: grid;
  gap: 1rem;
}

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
    transform 180ms ease,
    box-shadow 180ms ease,
    background-color 180ms ease;
}

.home-stage-step:hover {
  box-shadow: 8px 8px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(-2px, -2px);
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

.home-metric-strip {
  overflow: hidden;
  padding: 0;
}

.home-metric-grid {
  display: grid;
}

.home-metric-cell {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-height: 100%;
  padding: 1.25rem;
}

.home-metric-cell + .home-metric-cell {
  border-top: 2px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
}

.home-metric-value {
  font-size: 2.75rem;
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
}

.home-metric-title {
  font-size: 0.95rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  line-height: 1.3;
  margin: 0;
  text-transform: uppercase;
}

.home-metric-note {
  line-height: 1.7;
  max-width: 30rem;
}

.home-detail-grid {
  display: grid;
  gap: 1rem;
}

.home-detail-main {
  display: grid;
  gap: 1rem;
}

.home-section {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.home-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.home-section-title {
  font-size: 1.5rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.home-section-summary {
  line-height: 1.7;
  margin: 0;
  max-width: 28rem;
}

.home-list {
  display: grid;
  gap: 0;
  list-style: none;
  margin: 0;
  padding: 0;
}

.home-list-row {
  border-top: 1px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  padding: 1rem 0;
}

.home-list-row:first-child {
  border-top: 0;
  padding-top: 0;
}

.home-empty-row {
  border-top: 1px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  padding-top: 1rem;
}

.home-link-row {
  display: block;
  transition: transform 180ms ease;
}

.home-link-row:hover {
  transform: translateX(4px);
}

.home-trend-grid {
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(auto-fit, minmax(13rem, 1fr));
}

.home-trend-card {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
}

.home-detail-side {
  min-width: 0;
}

.home-queue-panel {
  display: flex;
  flex-direction: column;
  gap: 1rem;
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
    transform 180ms ease,
    box-shadow 180ms ease;
}

.home-queue-item:hover {
  box-shadow: 8px 8px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(-2px, -2px);
}

.home-queue-actions {
  display: grid;
  gap: 0.65rem;
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
  .home-stage-grid {
    gap: 1.5rem;
  }

  .home-stage-steps {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .home-metric-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .home-metric-cell:nth-child(2) {
    border-top: 0;
    border-left: 2px solid
      color-mix(in srgb, var(--neo-border) 18%, transparent);
  }

  .home-metric-cell:nth-child(3),
  .home-metric-cell:nth-child(4) {
    border-top: 2px solid color-mix(in srgb, var(--neo-border) 18%, transparent);
  }

  .home-metric-cell:nth-child(4) {
    border-left: 2px solid
      color-mix(in srgb, var(--neo-border) 18%, transparent);
  }
}

@media (min-width: 1280px) {
  .home-stage-grid {
    align-items: stretch;
    grid-template-columns: minmax(0, 1.3fr) minmax(22rem, 0.7fr);
  }

  .home-stage-main {
    align-items: stretch;
    gap: 2rem;
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 0.72fr);
    min-height: 32rem;
  }

  .home-stage-visual {
    min-height: auto;
  }

  .home-metric-grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }

  .home-metric-cell:nth-child(3),
  .home-metric-cell:nth-child(4) {
    border-top: 0;
  }

  .home-metric-cell + .home-metric-cell {
    border-left: 2px solid
      color-mix(in srgb, var(--neo-border) 18%, transparent);
    border-top: 0;
  }

  .home-detail-grid {
    align-items: start;
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 22rem);
  }

  .home-queue-panel {
    position: sticky;
    top: 1.5rem;
  }
}

@media (prefers-reduced-motion: reduce) {
  .home-stage-step,
  .home-stage-chip,
  .home-link-row,
  .home-queue-item {
    animation: none;
    transition: none;
  }

  .home-stage-step:hover,
  .home-link-row:hover,
  .home-queue-item:hover {
    box-shadow: inherit;
    transform: none;
  }
}
</style>
