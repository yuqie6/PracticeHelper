<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-blue)]">
      <p class="neo-kicker bg-white">{{ t('review.hero.kicker') }}</p>
      <p class="text-base font-semibold">
        {{ review?.overall ?? reviewHeaderText }}
      </p>
    </header>

    <NoticePanel
      v-if="loadError"
      tone="error"
      :title="t('review.loadErrorTitle')"
      :message="loadError"
    />
    <button
      v-if="loadError"
      type="button"
      class="neo-button-dark"
      @click="refetch()"
    >
      {{ t('common.retry') }}
    </button>

    <div v-else-if="isLoading" class="neo-panel space-y-2">
      <p class="neo-kicker bg-[var(--neo-yellow)]">
        {{ t('review.loadingTitle') }}
      </p>
      <p class="text-sm font-semibold">{{ t('review.hero.loading') }}</p>
    </div>

    <div v-else-if="!review" class="neo-panel space-y-2">
      <p class="neo-kicker bg-[var(--neo-yellow)]">
        {{ t('review.emptyTitle') }}
      </p>
      <p class="text-sm font-semibold">{{ t('review.emptyDescription') }}</p>
      <RouterLink to="/train" class="neo-button-dark mt-2">
        {{ t('review.continueAction') }}
      </RouterLink>
    </div>

    <div v-if="review" class="neo-grid lg:grid-cols-[0.9fr_1.1fr]">
      <div class="neo-panel space-y-4">
        <div class="space-y-4">
          <p class="neo-kicker bg-[var(--neo-red)]">
            {{ t('review.topFixTitle') }}
          </p>
          <div
            class="space-y-3 border-2 border-black bg-white px-4 py-4 md:border-4"
          >
            <p class="text-lg font-black leading-7">
              {{ review.top_fix || review.overall }}
            </p>
            <p class="neo-note">
              {{ review.top_fix_reason || t('review.topFixFallbackReason') }}
            </p>
          </div>
        </div>

        <div
          v-if="recommendedNextLabel || review.recommended_next?.reason"
          class="space-y-4"
        >
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('review.recommendedNextTitle') }}
          </p>
          <div
            class="space-y-3 border-2 border-black bg-white px-4 py-4 md:border-4"
          >
            <p class="text-base font-black">
              {{ recommendedNextLabel || t('review.continueAction') }}
            </p>
            <p v-if="review.recommended_next?.reason" class="neo-note">
              {{ review.recommended_next.reason }}
            </p>
            <RouterLink :to="continueTarget" class="neo-button-red mt-2">
              {{ t('review.startRecommendedAction') }}
            </RouterLink>
          </div>
        </div>

        <div class="space-y-4">
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('review.scoreBreakdown') }}
          </p>
          <div class="space-y-3">
            <div
              v-for="(value, key) in review.score_breakdown"
              :key="key"
              class="border-2 border-black bg-white px-4 py-3 md:border-4"
            >
              <div class="flex items-center justify-between gap-3">
                <span class="font-black">{{ key }}</span>
                <span class="neo-badge bg-[var(--neo-yellow)]">{{
                  value
                }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="neo-grid">
        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-yellow)]">
            {{ t('review.highlights') }}
          </p>
          <ul class="space-y-2">
            <li v-for="item in review.highlights" :key="item" class="neo-note">
              {{ item }}
            </li>
          </ul>
        </div>

        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-red)]">{{ t('review.gaps') }}</p>
          <ul class="space-y-2">
            <li v-for="item in review.gaps" :key="item" class="neo-note">
              {{ item }}
            </li>
          </ul>
        </div>

        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-green)]">
            {{ t('review.nextFocus') }}
          </p>
          <ul class="space-y-2">
            <li
              v-for="item in review.next_training_focus"
              :key="item"
              class="neo-note"
            >
              {{ item }}
            </li>
          </ul>
          <RouterLink :to="continueTarget" class="neo-button-red mt-4">
            {{ t('review.continueAction') }}
          </RouterLink>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink, useRoute } from 'vue-router';

import { getReview } from '../api/client';
import NoticePanel from '../components/NoticePanel.vue';
import { formatModeLabel, formatTopicLabel } from '../lib/labels';

const route = useRoute();
const reviewId = computed(() => route.params.id as string);
const { t } = useI18n();

const { data, error, isLoading, refetch } = useQuery({
  queryKey: ['review', reviewId],
  queryFn: () => getReview(reviewId.value),
});

const review = computed(() => data.value ?? null);
const loadError = computed(() =>
  error.value instanceof Error ? error.value.message : '',
);
const reviewHeaderText = computed(() => {
  if (loadError.value) {
    return t('review.headerError');
  }
  if (isLoading.value) {
    return t('review.hero.loading');
  }
  return t('review.emptyDescription');
});

const continueTarget = computed(() => {
  const recommended = review.value?.recommended_next;
  if (!recommended) {
    return '/train';
  }

  const params = new URLSearchParams();
  params.set('mode', recommended.mode);
  if (recommended.topic) {
    params.set('topic', recommended.topic);
  }
  if (recommended.project_id) {
    params.set('project_id', recommended.project_id);
  }

  const query = params.toString();
  return query ? `/train?${query}` : '/train';
});

const recommendedNextLabel = computed(() => {
  const recommended = review.value?.recommended_next;
  if (!recommended) {
    return '';
  }

  if (recommended.mode === 'basics' && recommended.topic) {
    return t('review.recommendedNextBasics', {
      topic: formatTopicLabel(t, recommended.topic),
    });
  }

  return t('review.recommendedNextMode', {
    mode: formatModeLabel(t, recommended.mode),
  });
});
</script>
