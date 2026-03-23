<template>
  <section class="neo-page session-page space-y-6 xl:space-y-8">
    <header class="neo-panel-hero session-stage bg-[var(--neo-yellow)]">
      <div class="session-stage-copy">
        <p class="neo-kicker bg-white">{{ t('session.hero.kicker') }}</p>
        <h1
          class="session-stage-title"
          :class="{ 'session-stage-title-dense': sessionHeroTitleDense }"
        >
          {{ sessionHeroTitle }}
        </h1>
        <p class="session-stage-note">
          {{ t('session.hero.description', { status: currentStatusLabel }) }}
        </p>
      </div>

      <div class="session-stage-stats">
        <article class="session-stage-stat">
          <span>{{ turnDisplay }}</span>
          <small>{{
            t('session.turnIndicator', {
              current: currentTurnIndex,
              total: session?.max_turns ?? 0,
            })
          }}</small>
        </article>
        <article class="session-stage-stat">
          <span>{{ latestScoreDisplay }}</span>
          <small>{{
            t('session.mainScore', { score: latestScoreDisplay })
          }}</small>
        </article>
        <article class="session-stage-stat">
          <span>{{ session?.job_target ? 'JD' : 'GEN' }}</span>
          <small>{{ t('session.jobTargetTitle') }}</small>
        </article>
      </div>
    </header>

    <NoticePanel
      v-if="submitError"
      tone="error"
      dismissible
      :title="t('session.submitErrorTitle')"
      :message="submitError"
      @dismiss="submitError = ''"
    />

    <div
      v-if="showReviewWrapUp"
      class="neo-panel neo-stamp space-y-3 bg-[var(--neo-green)]"
    >
      <p class="neo-kicker bg-white">{{ t('session.reviewWrapUpTitle') }}</p>
      <p class="text-sm font-semibold">
        {{ t('session.reviewWrapUpDescription') }}
      </p>
      <div class="neo-countdown-bar" />
    </div>

    <div
      v-else-if="showReviewRecovery"
      class="neo-panel space-y-3 bg-[var(--neo-yellow)]"
    >
      <p class="neo-kicker bg-white">{{ t('session.reviewPendingTitle') }}</p>
      <p class="text-sm font-semibold">
        {{ t('session.reviewPendingDescription') }}
      </p>
      <button
        type="button"
        class="neo-button-dark w-full sm:w-auto"
        :disabled="isRetryingReview"
        @click="retryReview"
      >
        {{
          isRetryingReview
            ? t('common.starting')
            : t('session.retryReviewAction')
        }}
      </button>
    </div>

    <div v-else-if="session && currentTurn" class="session-shell">
      <main class="session-main">
        <section class="neo-panel session-question-panel">
          <div class="session-section-head">
            <div class="space-y-2">
              <p class="neo-kicker bg-[var(--neo-red)]">
                {{ t('session.currentQuestion') }}
              </p>
              <h2 class="session-section-title">
                {{
                  t('session.turnIndicator', {
                    current: currentTurnIndex,
                    total: session.max_turns,
                  })
                }}
              </h2>
            </div>
            <span class="neo-badge bg-white">{{ currentStatusLabel }}</span>
          </div>

          <div v-if="followupIntent" class="session-intent-box">
            <p class="neo-subheading">{{ t('session.followupIntentTitle') }}</p>
            <p class="text-sm font-semibold leading-6">{{ followupIntent }}</p>
          </div>

          <h3
            class="session-question-title"
            :class="{ 'session-question-title-dense': sessionQuestionDense }"
          >
            {{ currentTurn.question }}
          </h3>

          <form
            v-if="canAnswerCurrentSession"
            class="session-answer-form"
            @submit.prevent="submit"
          >
            <Transition name="neo-turn" mode="out-in">
              <div
                v-if="showSubmittedAnswer"
                key="submitted"
                class="session-submitted-box"
              >
                <p class="neo-subheading">
                  {{ t('session.submittedAnswerTitle') }}
                </p>
                <p class="whitespace-pre-wrap text-sm font-semibold leading-6">
                  {{ submittedAnswer }}
                </p>
                <p class="neo-note">
                  {{ t('session.submittedAnswerDescription') }}
                </p>
              </div>
              <div v-else key="draft" class="space-y-4">
                <textarea
                  v-model="draftAnswer"
                  class="neo-textarea session-answer-input"
                  :placeholder="placeholderText"
                  :aria-label="placeholderText"
                  :disabled="isSubmitting || isBackgroundProcessing"
                  @keydown="handleAnswerKeydown"
                />
                <div class="session-answer-footer">
                  <p class="neo-note">
                    {{ t('session.submitShortcutHint') }}
                  </p>
                  <button
                    type="submit"
                    class="neo-button-dark w-full sm:w-auto"
                    :class="{
                      'animate-[neo-working_600ms_ease_infinite]': isSubmitting,
                    }"
                    :aria-busy="isSubmitting"
                    :disabled="isSubmitting || isBackgroundProcessing"
                  >
                    {{
                      isSubmitting ? t('common.submitting') : t('common.submit')
                    }}
                  </button>
                </div>
              </div>
            </Transition>
          </form>
          <p v-else class="neo-note">
            {{ t('session.answerLockedWhileProcessing') }}
          </p>
        </section>
      </main>

      <aside class="session-side neo-stagger-list">
        <section
          v-if="showProgressPanel"
          class="session-side-panel"
          aria-live="polite"
        >
          <ProgressPanel
            :kicker="t('session.processingKicker')"
            :title="progressTitle"
            :description="progressDescription"
            :steps="progressSteps"
            :active-index="progressStepIndex"
          />

          <div
            v-if="streamSections.length"
            class="max-h-[50vh] overflow-y-auto"
          >
            <StreamTracePanel
              :kicker="t('session.processingKicker')"
              :title="progressTitle"
              :description="progressDescription"
              :reasoning-title="t('session.reasoningTitle')"
              :content-title="t('session.contentTitle')"
              :sections="streamSections"
            />
          </div>
        </section>

        <section class="neo-panel session-side-panel">
          <p class="neo-kicker bg-[var(--neo-blue)]">
            {{ t('session.jobTargetTitle') }}
          </p>
          <h2 class="session-section-title">
            {{ session.job_target?.title || t('session.jobTargetEmpty') }}
          </h2>
          <p v-if="session.job_target" class="neo-note">
            {{ t('session.jobTargetDescription') }}
          </p>
          <p v-else class="neo-note">
            {{ t('session.jobTargetEmpty') }}
          </p>
          <p v-if="session.prompt_set" class="neo-note">
            {{ session.prompt_set.label }} · {{ session.prompt_set.status }}
          </p>
          <p v-if="sessionPromptOverlaySummary" class="neo-note">
            {{
              t('session.promptOverlaySummary', {
                summary: sessionPromptOverlaySummary,
              })
            }}
          </p>
        </section>

        <section class="neo-panel session-side-panel">
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('session.feedback') }}
          </p>
          <FeedbackPanel
            v-if="latestEvaluation"
            :evaluation="latestEvaluation"
          />
          <p v-else class="neo-note">{{ t('session.feedbackEmpty') }}</p>
        </section>
      </aside>
    </div>

    <section v-else class="neo-panel session-side-panel">
      <p class="neo-note">{{ t('common.loading') }}</p>
    </section>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, onBeforeUnmount, ref, Transition, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import {
  ApiError,
  getSession,
  retrySessionReview,
  submitAnswerStream,
  type StreamEvent,
} from '../api/client';
import FeedbackPanel from '../components/FeedbackPanel.vue';
import NoticePanel from '../components/NoticePanel.vue';
import ProgressPanel from '../components/ProgressPanel.vue';
import StreamTracePanel from '../components/StreamTracePanel.vue';
import { resolveApiErrorMessage } from '../lib/apiErrors';
import { formatStatusLabel } from '../lib/labels';
import { formatPromptOverlaySummary } from '../lib/promptOverlay';
import { isSubmitShortcut } from '../lib/shortcuts';
import { appendStreamEvent, type StreamSection } from '../lib/streaming';
import { useProgressSteps } from '../lib/useProgressSteps';

const route = useRoute();
const router = useRouter();
const queryClient = useQueryClient();
const draftAnswer = ref('');
const submittedAnswer = ref('');
const streamSections = ref<StreamSection[]>([]);
const streamEvents = ref<StreamEvent[]>([]);
const submitError = ref('');
const isWrappingUpReview = ref(false);
const { t, tm } = useI18n();
let reviewRedirectTimer: number | null = null;
let reviewRedirectTarget = '';

const sessionId = computed(() => route.params.id as string);

const { data } = useQuery({
  queryKey: ['session', sessionId],
  queryFn: () => getSession(sessionId.value),
});

const session = computed(() => data.value ?? null);
const turns = computed(() => session.value?.turns ?? []);
const currentTurn = computed(() => turns.value[turns.value.length - 1] ?? null);
const currentTurnIndex = computed(() => currentTurn.value?.turn_index ?? 1);
const latestEvaluation = computed(() => {
  // 当前 turn 已经有评估结果（已作答）→ 显示当前 turn 的评估
  if (currentTurn.value?.evaluation) {
    return currentTurn.value.evaluation;
  }
  // 当前 turn 还没作答，且是追问轮 → 显示上一轮评估作为上下文
  if (currentTurnIndex.value > 1 && turns.value.length >= 2) {
    return turns.value[turns.value.length - 2]?.evaluation ?? null;
  }
  return null;
});
const followupIntent = computed(() => {
  // 追问意图来自上一轮评估（当前 turn 还没答时才显示）
  if (
    currentTurnIndex.value > 1 &&
    !currentTurn.value?.answer &&
    turns.value.length >= 2
  ) {
    return (
      turns.value[turns.value.length - 2]?.evaluation?.followup_intent ?? ''
    );
  }
  return '';
});
const canAnswerCurrentSession = computed(() =>
  ['waiting_answer', 'active'].includes(session.value?.status ?? ''),
);
const currentStatusLabel = computed(() => {
  if (!session.value?.status) {
    return t('common.loading');
  }

  return formatStatusLabel(t, session.value.status);
});
const sessionPromptOverlaySummary = computed(() =>
  formatPromptOverlaySummary(
    t,
    session.value?.prompt_overlay,
    session.value?.prompt_overlay_hash,
  ),
);
const sessionHeroTitle = computed(
  () => currentTurn.value?.question || t('session.currentQuestion'),
);
const sessionHeroTitleDense = computed(
  () => sessionHeroTitle.value.length > 26,
);
const turnDisplay = computed(
  () => `${currentTurnIndex.value}/${session.value?.max_turns ?? 0}`,
);
const latestScoreDisplay = computed(() => {
  const score = latestEvaluation.value?.score;
  if (score == null || Number.isNaN(score)) {
    return '--';
  }
  return Number(score).toFixed(1);
});
const sessionQuestionDense = computed(
  () => (currentTurn.value?.question?.length ?? 0) > 30,
);

const placeholderText = computed(() =>
  currentTurnIndex.value > 1
    ? t('session.placeholderFollowup')
    : t('session.placeholderInitial'),
);
const showSubmittedAnswer = computed(
  () =>
    Boolean(submittedAnswer.value) &&
    (isSubmitting.value || isBackgroundProcessing.value),
);

const mutation = useMutation({
  mutationFn: (payload: string) => {
    streamSections.value = [];
    streamEvents.value = [];
    submitError.value = '';
    return submitAnswerStream(sessionId.value, payload, handleStreamEvent);
  },
  onMutate: (payload) => {
    submittedAnswer.value = payload;
  },
  onSuccess: async (updated) => {
    if (
      isWrappingUpReview.value &&
      updated.status === 'completed' &&
      updated.review_id
    ) {
      scheduleReviewRedirect(updated.review_id);
    }
    queryClient.setQueryData(['session', sessionId], updated);
    draftAnswer.value = '';
    submittedAnswer.value = '';
    await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    await queryClient.invalidateQueries({ queryKey: ['weaknesses'] });
  },
  onError: async (error) => {
    submittedAnswer.value = '';
    submitError.value = resolveSessionErrorMessage(error);
    if (shouldRefreshSession(error)) {
      await queryClient.invalidateQueries({ queryKey: ['session', sessionId] });
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    }
  },
});

const retryReviewMutation = useMutation({
  mutationFn: () => retrySessionReview(sessionId.value),
  onSuccess: async (updated) => {
    if (updated.status === 'completed' && updated.review_id) {
      isWrappingUpReview.value = true;
      scheduleReviewRedirect(updated.review_id);
    }
    queryClient.setQueryData(['session', sessionId], updated);
    await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
  },
  onError: async (error) => {
    submitError.value = resolveSessionErrorMessage(error);
    if (shouldRefreshSession(error)) {
      await queryClient.invalidateQueries({ queryKey: ['session', sessionId] });
      await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    }
  },
});

const isSubmitting = computed(() => mutation.isPending.value);
const isRetryingReview = computed(() => retryReviewMutation.isPending.value);
const isBackgroundProcessing = computed(() =>
  ['generating_question', 'evaluating'].includes(session.value?.status ?? ''),
);
const showReviewWrapUp = computed(() => isWrappingUpReview.value);
const showProgressPanel = computed(
  () =>
    !showReviewWrapUp.value &&
    (isSubmitting.value ||
      isRetryingReview.value ||
      isBackgroundProcessing.value),
);
const showReviewRecovery = computed(
  () =>
    !showReviewWrapUp.value &&
    session.value?.status === 'review_pending' &&
    !isRetryingReview.value &&
    !isSubmitting.value,
);
const progressMode = computed(() => {
  if (session.value?.status === 'review_pending' || isRetryingReview.value) {
    return 'review';
  }

  if (session.value?.status === 'generating_question') {
    return 'question';
  }

  const isLastTurn = currentTurnIndex.value >= (session.value?.max_turns ?? 2);
  if (isSubmitting.value) {
    return isLastTurn ? 'review' : 'answer';
  }

  return 'answer';
});
const progressTitle = computed(() => {
  switch (progressMode.value) {
    case 'review':
      return t('session.processingReviewTitle');
    case 'answer':
      return t('session.processingEvaluatingTitle');
    default:
      return t('session.processingGeneratingQuestionTitle');
  }
});
const progressDescription = computed(() => {
  switch (progressMode.value) {
    case 'review':
      return t('session.processingReviewDescription');
    case 'answer':
      return t('session.processingEvaluatingDescription');
    default:
      return t('session.processingGeneratingQuestionDescription');
  }
});
const progressSteps = computed(() => {
  const key =
    progressMode.value === 'review'
      ? 'progress.evaluateFollowup.steps'
      : progressMode.value === 'answer'
        ? 'progress.evaluateMain.steps'
        : 'progress.createSession.steps';

  return tm(key) as string[];
});
const progressStepDefinitions = computed(() =>
  buildProgressStepDefinitions(progressMode.value, progressSteps.value),
);
const { activeIndex: progressStepIndex } = useProgressSteps(
  showProgressPanel,
  progressStepDefinitions,
  streamEvents,
);

watch(
  session,
  async (value) => {
    if (value?.status === 'completed' && value.review_id) {
      if (isWrappingUpReview.value) {
        scheduleReviewRedirect(value.review_id);
        return;
      }
      await router.push(`/reviews/${value.review_id}`);
    }
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  clearReviewRedirectTimer();
});

function submit() {
  if (
    !draftAnswer.value.trim() ||
    !canAnswerCurrentSession.value ||
    isBackgroundProcessing.value
  ) {
    return;
  }
  mutation.mutate(draftAnswer.value);
}

function handleAnswerKeydown(event: KeyboardEvent) {
  if (!isSubmitShortcut(event)) {
    return;
  }

  event.preventDefault();
  submit();
}

function retryReview() {
  submitError.value = '';
  retryReviewMutation.mutate();
}

function handleStreamEvent(event: StreamEvent) {
  streamEvents.value = [...streamEvents.value, event];
  streamSections.value = appendStreamEvent(streamSections.value, event);
  if (event.type === 'status' && event.name === 'review_saved') {
    isWrappingUpReview.value = true;
    if (session.value?.review_id) {
      scheduleReviewRedirect(session.value.review_id);
    }
  }
}

function shouldRefreshSession(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    [
      'session_busy',
      'session_review_pending',
      'session_completed',
      'session_not_recoverable',
      'session_answer_conflict',
      'review_generation_retry',
    ].includes(error.code ?? '')
  );
}

function resolveSessionErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return resolveApiErrorMessage(t, error);
  }

  return error instanceof Error ? error.message : t('common.requestFailed');
}

function scheduleReviewRedirect(reviewId: string) {
  if (!reviewId || reviewRedirectTarget === reviewId) {
    return;
  }

  clearReviewRedirectTimer();
  reviewRedirectTarget = reviewId;
  reviewRedirectTimer = window.setTimeout(() => {
    reviewRedirectTimer = null;
    void router.push(`/reviews/${reviewId}`);
  }, 1500);
}

function clearReviewRedirectTimer() {
  if (reviewRedirectTimer != null) {
    window.clearTimeout(reviewRedirectTimer);
    reviewRedirectTimer = null;
  }
  reviewRedirectTarget = '';
}

function buildProgressStepDefinitions(
  mode: 'question' | 'answer' | 'review',
  labels: string[],
) {
  if (mode === 'review') {
    return [
      {
        label: labels[0] ?? '',
        signals: [{ type: 'status' as const, value: 'answer_received' }],
      },
      {
        label: labels[1] ?? '',
        signals: [{ type: 'status' as const, value: 'feedback_ready' }],
      },
      {
        label: labels[2] ?? '',
        signals: [
          { type: 'status' as const, value: 'review_started' },
          { type: 'status' as const, value: 'review_saved' },
        ],
      },
    ];
  }

  if (mode === 'answer') {
    return [
      {
        label: labels[0] ?? '',
        signals: [{ type: 'status' as const, value: 'answer_received' }],
      },
      {
        label: labels[1] ?? '',
        signals: [{ type: 'status' as const, value: 'evaluation_started' }],
      },
      {
        label: labels[2] ?? '',
        signals: [
          { type: 'status' as const, value: 'feedback_ready' },
          { type: 'status' as const, value: 'followup_ready' },
        ],
      },
    ];
  }

  return [
    {
      label: labels[0] ?? '',
      signals: [{ type: 'phase' as const, value: 'prepare_context' }],
    },
    {
      label: labels[1] ?? '',
      signals: [{ type: 'phase' as const, value: 'call_model' }],
    },
    {
      label: labels[2] ?? '',
      signals: [{ type: 'phase' as const, value: 'parse_result' }],
    },
  ];
}
</script>

<style scoped>
.session-page {
  position: relative;
}

.session-stage {
  display: grid;
  gap: 1.5rem;
  overflow: hidden;
  position: relative;
  background: linear-gradient(
    135deg,
    color-mix(in srgb, var(--neo-yellow) 88%, white) 0%,
    color-mix(in srgb, var(--neo-yellow) 60%, var(--neo-red)) 100%
  );
}

.session-stage::before {
  content: '';
  position: absolute;
  inset: 1rem;
  border: 1px solid color-mix(in srgb, var(--neo-border) 20%, transparent);
  pointer-events: none;
}

.session-stage-copy,
.session-stage-stats {
  position: relative;
  z-index: 1;
}

.session-stage-copy {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.session-stage-title {
  font-size: clamp(1.8rem, 5vw, 3.8rem);
  font-weight: 900;
  letter-spacing: -0.05em;
  line-height: 1;
  margin: 0;
  max-width: 16ch;
  text-wrap: balance;
}

.session-stage-title-dense {
  font-size: clamp(1.35rem, 2.8vw, 2.3rem);
  letter-spacing: -0.03em;
  line-height: 1.24;
  max-width: 28ch;
  text-wrap: pretty;
}

.session-stage-note {
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.7;
  margin: 0;
  max-width: 40rem;
  text-wrap: pretty;
}

.session-stage-stats {
  display: grid;
  gap: 0.75rem;
}

.session-stage-stat {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 6px 6px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  padding: 1rem;
}

.session-stage-stat span {
  font-size: clamp(2rem, 7vw, 3.4rem);
  font-weight: 900;
  letter-spacing: -0.08em;
  line-height: 0.9;
}

.session-stage-stat small {
  font-size: 0.75rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  line-height: 1.4;
  text-transform: uppercase;
}

.session-shell {
  display: grid;
  gap: 1rem;
}

.session-main,
.session-side {
  display: grid;
  gap: 1rem;
}

.session-question-panel,
.session-side-panel {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}

.session-section-head {
  align-items: end;
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
  justify-content: space-between;
  padding-bottom: 1rem;
}

.session-section-title {
  font-size: 1.25rem;
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.session-intent-box,
.session-submitted-box {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  display: grid;
  gap: 0.75rem;
  padding: 1rem;
}

.session-question-title {
  font-size: clamp(1.6rem, 4vw, 2.8rem);
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1.15;
  margin: 0;
  text-wrap: balance;
}

.session-question-title-dense {
  font-size: clamp(1.2rem, 2.4vw, 1.8rem);
  line-height: 1.38;
  max-width: 34ch;
  text-wrap: pretty;
}

.session-answer-form {
  display: grid;
  gap: 1rem;
}

.session-answer-input {
  min-height: 18rem;
}

.session-answer-footer {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  justify-content: space-between;
}

@media (min-width: 768px) {
  .session-stage-stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .session-answer-footer {
    align-items: center;
    flex-direction: row;
  }
}

@media (min-width: 1280px) {
  .session-stage {
    align-items: start;
    grid-template-columns: minmax(0, 1.15fr) minmax(18rem, 0.85fr);
  }

  .session-shell {
    align-items: start;
    grid-template-columns: minmax(0, 1.1fr) minmax(20rem, 0.9fr);
  }

  .session-side {
    position: sticky;
    top: 1.5rem;
  }
}
</style>
