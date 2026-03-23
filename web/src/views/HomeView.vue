<template>
  <section class="neo-page home-page space-y-6 xl:space-y-8">
    <div class="home-stage-grid">
      <HomeHeroPanel
        :title="dashboard?.today_focus ?? t('common.firstTrainingHint')"
        :support="heroSupport"
        :show-onboarding="showOnboarding"
        :deadline-display="deadlineDisplay"
        :due-review-display="dueReviewDisplay"
        :onboarding-steps="onboardingSteps"
      />

      <HomeSummarySignals
        :days-until-deadline="dashboard?.days_until_deadline"
        :deadline-note="deadlineNote"
        :job-target-summary="jobTargetSummary"
        :current-session="currentSessionCard"
        :primary-due-review="primaryDueReviewCard"
        :is-completing="completeMutation.isPending.value"
        @complete-review="completeReview"
      />
    </div>

    <section class="neo-panel home-metric-strip">
      <div class="home-metric-grid neo-stagger-list">
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

          <ul class="home-list neo-stagger-list">
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

          <div v-if="trends.length" class="home-trend-grid neo-stagger-list">
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

          <ul class="home-list neo-stagger-list">
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
                      {{ formatSessionName(t, session) }}
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

      <HomeFocusPanel
        :due-reviews="focusDueReviews"
        :current-onboarding-step="currentOnboardingStep"
        :recommended-track="dashboard?.recommended_track ?? t('common.noRecommendation')"
        :is-completing="completeMutation.isPending.value"
        @complete-review="completeReview"
      />
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
  type WeaknessTrend,
} from '../api/client';
import HomeFocusPanel from '../components/HomeFocusPanel.vue';
import HomeHeroPanel from '../components/HomeHeroPanel.vue';
import HomeSummarySignals from '../components/HomeSummarySignals.vue';
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
import { buildSessionTarget, formatSessionName } from '../lib/sessionSummary';
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
    toneClass: onboardingStepClass(step.status),
  })),
);
const currentOnboardingStep = computed(
  () => onboardingSteps.value.find((step) => step.status !== 'done') ?? null,
);
const deadlineNote = computed(() =>
  dashboard.value?.days_until_deadline == null
    ? t('common.setDeadlineHint')
    : (dashboard.value?.recommended_track ?? t('common.noRecommendation')),
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
const currentSessionCard = computed(() => {
  const session = currentSession.value;
  if (!session) {
    return null;
  }

  return {
    description: t('home.currentSession.description', {
      name: formatSessionName(t, session),
      status: formatStatusLabel(t, session.status),
    }),
    updatedAtLabel: t('common.lastUpdated', {
      value: formatUpdatedAt(session.updated_at),
    }),
    jobTargetTitle: session.job_target?.title,
    href: buildSessionTarget(session),
  };
});
const primaryDueReviewCard = computed(() => {
  const item = primaryDueReview.value;
  if (!item) {
    return null;
  }

  return {
    id: item.id,
    headline: formatDueReviewHeadline(item),
    hint: formatDueReviewHint(item),
    href: buildDueReviewTarget(item),
  };
});
const focusDueReviews = computed(() =>
  dueReviews.value.slice(0, 5).map((item) => ({
    id: item.id,
    kindLabel: item.weakness_kind
      ? formatWeaknessKindLabel(t, item.weakness_kind)
      : t('home.dueReviews.review'),
    headline: formatDueReviewHeadline(item),
    hint: formatDueReviewHint(item),
    href: buildDueReviewTarget(item),
  })),
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

function completeReview(id: number) {
  completeMutation.mutate(id);
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
  transition: transform var(--motion-duration-base) var(--motion-ease-standard);
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

@media (min-width: 768px) {
  .home-stage-grid {
    gap: 1.5rem;
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
}

@media (prefers-reduced-motion: reduce) {
  .home-link-row {
    animation: none;
    transition: none;
  }

  .home-link-row:hover {
    transform: none;
  }
}
</style>
