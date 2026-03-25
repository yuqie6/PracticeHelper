<template>
  <section class="neo-page session-page space-y-6 xl:space-y-8">
    <SessionStageHero
      :title="sessionHeroTitle"
      :title-dense="sessionHeroTitleDense"
      :note="t('session.hero.description', { status: currentStatusLabel })"
      :stats="sessionStageStats"
    />

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
        <SessionQuestionPanel
          :turn-label="currentTurnLabel"
          :current-status-label="currentStatusLabel"
          :followup-intent="followupIntent"
          :question="currentTurn.question"
          :question-dense="sessionQuestionDense"
          :can-answer-current-session="canAnswerCurrentSession"
          :show-submitted-answer="showSubmittedAnswer"
          :submitted-answer="submittedAnswer"
          :draft-answer="draftAnswer"
          :placeholder-text="placeholderText"
          :is-submitting="isSubmitting"
          :is-background-processing="isBackgroundProcessing"
          @submit="submit"
          @update:draft-answer="draftAnswer = $event"
          @answer-keydown="handleAnswerKeydown"
        />
      </main>

      <SessionSideSummary
        :show-progress-panel="showProgressPanel"
        :progress-title="progressTitle"
        :progress-description="progressDescription"
        :progress-steps="progressSteps"
        :progress-step-index="progressStepIndex"
        :stream-sections="streamSections"
        :job-target-title="session.job_target?.title || t('session.jobTargetEmpty')"
        :job-target-description="session.job_target ? t('session.jobTargetDescription') : t('session.jobTargetEmpty')"
        :prompt-set-summary="session.prompt_set ? `${session.prompt_set.label} · ${session.prompt_set.status}` : ''"
        :prompt-overlay-summary="sessionPromptOverlaySummary"
        :latest-evaluation="latestEvaluation"
      />
    </div>

    <section v-else class="neo-panel session-side-panel">
      <p class="neo-note">{{ t('common.loading') }}</p>
    </section>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, onBeforeUnmount, ref, watch } from 'vue';
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
import SessionQuestionPanel from '../components/SessionQuestionPanel.vue';
import SessionSideSummary from '../components/SessionSideSummary.vue';
import SessionStageHero from '../components/SessionStageHero.vue';
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
const currentTurnLabel = computed(() =>
  t('session.turnIndicator', {
    current: currentTurnIndex.value,
    total: session.value?.max_turns ?? 0,
  }),
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
const sessionStageStats = computed(() => [
  {
    value: turnDisplay.value,
    label: currentTurnLabel.value,
  },
  {
    value: latestScoreDisplay.value,
    label: t('session.mainScore', { score: latestScoreDisplay.value }),
  },
  {
    value: session.value?.job_target ? 'JD' : 'GEN',
    label: t('session.jobTargetTitle'),
  },
]);

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

.session-shell {
  display: grid;
  gap: 1rem;
}

.session-main {
  display: grid;
  gap: 1rem;
}

@media (min-width: 1280px) {
  .session-shell {
    align-items: start;
    grid-template-columns: minmax(0, 1.1fr) minmax(20rem, 0.9fr);
  }
}
</style>
