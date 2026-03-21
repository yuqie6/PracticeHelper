<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-yellow)]">
      <p class="neo-kicker bg-white">{{ t('session.hero.kicker') }}</p>
      <p class="text-base font-semibold">
        {{ t('session.hero.description', { status: currentStatusLabel }) }}
      </p>
      <p v-if="session" class="mt-1 text-sm font-black">
        {{
          t('session.turnIndicator', {
            current: currentTurnIndex,
            total: session.max_turns,
          })
        }}
      </p>
    </header>

    <div class="neo-panel space-y-2 bg-white">
      <p class="neo-kicker bg-[var(--neo-blue)]">
        {{ t('session.jobTargetTitle') }}
      </p>
      <p v-if="session?.job_target" class="text-base font-black">
        {{ session.job_target.title }}
      </p>
      <p v-else class="text-sm font-semibold">
        {{ t('session.jobTargetEmpty') }}
      </p>
      <p v-if="session?.job_target" class="neo-note">
        {{ t('session.jobTargetDescription') }}
      </p>
    </div>

    <div
      v-if="showProgressPanel"
      class="sticky top-4 z-10 space-y-4"
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
    </div>

    <NoticePanel
      v-if="submitError"
      tone="error"
      :title="t('session.submitErrorTitle')"
      :message="submitError"
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

    <div
      v-else-if="session && currentTurn"
      class="neo-grid lg:grid-cols-[1.1fr_0.9fr]"
    >
      <div class="neo-panel space-y-4">
        <p class="neo-kicker bg-[var(--neo-red)]">
          {{ t('session.currentQuestion') }}
        </p>
        <div
          v-if="followupIntent"
          class="space-y-2 border-2 border-black bg-white px-4 py-4 md:border-4"
        >
          <p class="neo-subheading">{{ t('session.followupIntentTitle') }}</p>
          <p class="text-sm font-semibold leading-6">{{ followupIntent }}</p>
        </div>
        <h3 class="text-2xl font-black">{{ currentTurn.question }}</h3>

        <form
          v-if="canAnswerCurrentSession"
          class="space-y-4"
          @submit.prevent="submit"
        >
          <Transition name="neo-turn" mode="out-in">
            <div
              v-if="showSubmittedAnswer"
              key="submitted"
              class="space-y-2 border-2 border-black bg-white px-4 py-4 md:border-4"
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
                class="neo-textarea"
                :placeholder="placeholderText"
                :disabled="isSubmitting || isBackgroundProcessing"
                @keydown="handleAnswerKeydown"
              />
              <p class="neo-note">
                {{ t('session.submitShortcutHint') }}
              </p>
              <button
                type="submit"
                class="neo-button-dark w-full sm:w-auto"
                :class="{ 'animate-[neo-working_600ms_ease_infinite]': isSubmitting }"
                :disabled="isSubmitting || isBackgroundProcessing"
              >
                {{ isSubmitting ? t('common.submitting') : t('common.submit') }}
              </button>
            </div>
          </Transition>
        </form>
        <p v-else class="neo-note">
          {{ t('session.answerLockedWhileProcessing') }}
        </p>
      </div>

      <div class="neo-panel space-y-4">
        <p class="neo-kicker bg-[var(--neo-green)]">
          {{ t('session.feedback') }}
        </p>
        <template v-if="latestEvaluation">
          <div
            class="flex items-center gap-4 border-l-8 bg-white px-4 py-4"
            :style="{ borderColor: `var(${scoreColor(latestEvaluation.score)})` }"
          >
            <span class="text-5xl font-black">{{ latestEvaluation.score }}</span>
            <div>
              <p class="text-sm font-bold uppercase tracking-[0.08em]">/10</p>
              <p class="text-base font-semibold leading-7">
                {{
                  latestEvaluation.headline || t('session.feedbackHeadlineFallback')
                }}
              </p>
            </div>
          </div>

          <details v-if="latestEvaluation.strengths.length" class="space-y-2">
            <summary class="neo-subheading cursor-pointer">
              {{ t('session.strengths') }}
            </summary>
            <ul class="mt-3 space-y-2">
              <li
                v-for="item in latestEvaluation.strengths"
                :key="item"
                class="neo-note"
              >
                {{ item }}
              </li>
            </ul>
          </details>

          <details v-if="latestEvaluation.gaps.length" class="space-y-2" open>
            <summary class="neo-subheading cursor-pointer">
              {{ t('session.gaps') }}
            </summary>
            <ul class="mt-3 space-y-2">
              <li
                v-for="item in latestEvaluation.gaps"
                :key="item"
                class="neo-note"
              >
                {{ item }}
              </li>
            </ul>
          </details>

          <div class="space-y-2">
            <p class="neo-subheading">{{ t('session.suggestionTitle') }}</p>
            <p class="neo-note">
              {{
                latestEvaluation.suggestion || t('session.suggestionFallback')
              }}
            </p>
          </div>

          <details
            v-if="Object.keys(latestEvaluation.score_breakdown ?? {}).length"
            class="space-y-2"
          >
            <summary class="neo-subheading cursor-pointer">
              {{ t('session.scoreBreakdownTitle') }}
            </summary>
            <ul class="mt-3 space-y-2">
              <li
                v-for="(score, label) in latestEvaluation.score_breakdown"
                :key="label"
                class="border-2 border-black bg-white px-3 py-2 md:border-4"
              >
                <div class="flex items-center justify-between text-sm font-semibold">
                  <span>{{ label }}</span>
                  <span>{{ score }}/10</span>
                </div>
                <div
                  class="neo-score-bar mt-1"
                  :style="{
                    width: `${Number(score) * 10}%`,
                    background: `var(${scoreColor(Number(score))})`,
                  }"
                />
              </li>
            </ul>
          </details>
        </template>
        <p v-else class="neo-note">{{ t('session.feedbackEmpty') }}</p>
      </div>
    </div>
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
import NoticePanel from '../components/NoticePanel.vue';
import ProgressPanel from '../components/ProgressPanel.vue';
import StreamTracePanel from '../components/StreamTracePanel.vue';
import { formatStatusLabel } from '../lib/labels';
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
    switch (error.code) {
      case 'session_busy':
        return t('session.conflictBusy');
      case 'session_review_pending':
        return t('session.conflictReviewPending');
      case 'session_completed':
        return t('session.conflictCompleted');
      case 'session_not_recoverable':
        return t('session.retryReviewNotRecoverable');
      case 'session_answer_conflict':
        return t('session.conflictInvalidStatus');
      case 'review_generation_retry':
        return t('session.reviewGenerationRetry');
      default:
        return error.message;
    }
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

function scoreColor(score: number): string {
  if (score >= 8) return '--neo-green';
  if (score >= 5) return '--neo-yellow';
  return '--neo-red';
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
