<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-blue)]">
      <p class="neo-kicker bg-white">{{ t('review.hero.kicker') }}</p>
      <h2 class="neo-heading">{{ t('review.hero.title') }}</h2>
      <p class="mt-3 text-base font-semibold">
        {{ review?.overall ?? t('review.hero.loading') }}
      </p>
    </header>

    <div v-if="review" class="neo-grid lg:grid-cols-[0.9fr_1.1fr]">
      <div class="neo-panel space-y-4">
        <p class="neo-kicker bg-[var(--neo-green)]">{{ t('review.scoreBreakdown') }}</p>
        <div class="space-y-3">
          <div
            v-for="(value, key) in review.score_breakdown"
            :key="key"
            class="border-2 border-black bg-white px-4 py-3 md:border-4"
          >
            <div class="flex items-center justify-between gap-3">
              <span class="font-black">{{ key }}</span>
              <span class="neo-badge bg-[var(--neo-yellow)]">{{ value }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="neo-grid">
        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-yellow)]">{{ t('review.highlights') }}</p>
          <ul class="space-y-2">
            <li v-for="item in review.highlights" :key="item" class="neo-note">{{ item }}</li>
          </ul>
        </div>

        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-red)]">{{ t('review.gaps') }}</p>
          <ul class="space-y-2">
            <li v-for="item in review.gaps" :key="item" class="neo-note">{{ item }}</li>
          </ul>
        </div>

        <div class="neo-panel">
          <p class="neo-kicker bg-[var(--neo-green)]">{{ t('review.nextFocus') }}</p>
          <ul class="space-y-2">
            <li v-for="item in review.next_training_focus" :key="item" class="neo-note">{{ item }}</li>
          </ul>
          <RouterLink to="/train" class="neo-button-red mt-4">
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

const route = useRoute();
const reviewId = computed(() => route.params.id as string);
const { t } = useI18n();

const { data } = useQuery({
  queryKey: ['review', reviewId],
  queryFn: () => getReview(reviewId.value),
});

const review = computed(() => data.value ?? null);
</script>
