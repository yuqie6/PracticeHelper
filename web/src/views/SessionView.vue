<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-yellow)]">
      <p class="neo-kicker bg-white">{{ t('session.hero.kicker') }}</p>
      <h2 class="neo-heading">{{ t('session.hero.title') }}</h2>
      <p class="mt-3 text-base font-semibold">
        {{ t('session.hero.description', { status: currentStatusLabel }) }}
      </p>
    </header>

    <div v-if="session && activePrompt" class="neo-grid lg:grid-cols-[1.1fr_0.9fr]">
      <div class="neo-panel space-y-4">
        <p class="neo-kicker bg-[var(--neo-red)]">{{ t('session.currentQuestion') }}</p>
        <h3 class="text-2xl font-black">{{ activePrompt.question }}</h3>
        <ul class="flex flex-wrap gap-2">
          <li
            v-for="point in activePrompt.expectedPoints"
            :key="point"
            class="neo-badge bg-[var(--neo-blue)]"
          >
            {{ point }}
          </li>
        </ul>

        <form class="space-y-4" @submit.prevent="submit">
          <textarea v-model="answer" class="neo-textarea" :placeholder="placeholderText" />
          <button type="submit" class="neo-button-dark" :disabled="isSubmitting">
            {{ isSubmitting ? t('common.submitting') : t('common.submit') }}
          </button>
        </form>
      </div>

      <div class="neo-panel space-y-4">
        <p class="neo-kicker bg-[var(--neo-green)]">{{ t('session.feedback') }}</p>
        <template v-if="lastTurn?.evaluation">
          <p class="text-lg font-black">
            {{ t('session.mainScore', { score: lastTurn.evaluation.score }) }}
          </p>
          <div class="space-y-2">
            <p class="neo-subheading">{{ t('session.strengths') }}</p>
            <ul class="space-y-2">
              <li v-for="item in lastTurn.evaluation.strengths" :key="item" class="neo-note">
                {{ item }}
              </li>
            </ul>
          </div>
          <div class="space-y-2">
            <p class="neo-subheading">{{ t('session.gaps') }}</p>
            <ul class="space-y-2">
              <li v-for="item in lastTurn.evaluation.gaps" :key="item" class="neo-note">
                {{ item }}
              </li>
            </ul>
          </div>
        </template>
        <p v-else class="neo-note">{{ t('session.feedbackEmpty') }}</p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query';
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { useRoute, useRouter } from 'vue-router';

import { getSession, submitAnswer } from '../api/client';
import { formatStatusLabel } from '../lib/labels';

const route = useRoute();
const router = useRouter();
const queryClient = useQueryClient();
const answer = ref('');
const { t } = useI18n();

const sessionId = computed(() => route.params.id as string);

const { data } = useQuery({
  queryKey: ['session', sessionId],
  queryFn: () => getSession(sessionId.value),
});

const session = computed(() => data.value ?? null);
const lastTurn = computed(() => session.value?.turns?.[session.value.turns.length - 1]);
const currentStatusLabel = computed(() => {
  if (!session.value?.status) {
    return t('common.loading');
  }

  return formatStatusLabel(t, session.value.status);
});

const activePrompt = computed(() => {
  const turn = lastTurn.value;
  if (!turn) {
    return null;
  }
  // 这里不是单纯切换展示文案，而是在把后端 session 状态映射成当前唯一允许作答的题目。
  // completed 必须返回 null，避免前端在复盘已生成后继续展示可提交的输入框。
  if (session.value?.status === 'followup' && turn.followup_question) {
    return {
      question: turn.followup_question,
      expectedPoints: turn.followup_expected_points ?? [],
    };
  }
  if (session.value?.status === 'completed') {
    return null;
  }
  return {
    question: turn.question,
    expectedPoints: turn.expected_points,
  };
});

const placeholderText = computed(() =>
  session.value?.status === 'followup'
    ? t('session.placeholderFollowup')
    : t('session.placeholderInitial'),
);

const mutation = useMutation({
  mutationFn: (payload: string) => submitAnswer(sessionId.value, payload),
  onSuccess: async (updated) => {
    queryClient.setQueryData(['session', sessionId], updated);
    answer.value = '';
    await queryClient.invalidateQueries({ queryKey: ['dashboard'] });
    await queryClient.invalidateQueries({ queryKey: ['weaknesses'] });
  },
});

const isSubmitting = computed(() => mutation.isPending.value);

watch(
  session,
  async (value) => {
    // 训练页只承载“可继续回答”的中间态；一旦 session 已完成且 review 已落库，
    // 无论是首次加载已完成会话，还是本页刚提交完答案，都应该立即回到 review 页面。
    if (value?.status === 'completed' && value.review_id) {
      await router.push(`/reviews/${value.review_id}`);
    }
  },
  { immediate: true },
);

function submit() {
  if (!answer.value.trim()) {
    return;
  }
  mutation.mutate(answer.value);
}
</script>
