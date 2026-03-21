<template>
  <section class="neo-page space-y-6">
    <header class="neo-panel bg-[var(--neo-yellow)]">
      <p class="neo-kicker bg-white">{{ t('history.hero.kicker') }}</p>
      <h1 class="text-xl font-black md:text-2xl">
        {{ t('history.hero.title') }}
      </h1>
    </header>

    <div class="grid gap-3 md:flex md:flex-wrap">
      <select v-model="filters.mode" class="neo-select w-full md:w-auto">
        <option value="">{{ t('history.filters.allModes') }}</option>
        <option value="basics">{{ formatModeLabel(t, 'basics') }}</option>
        <option value="project">{{ formatModeLabel(t, 'project') }}</option>
      </select>
      <select v-model="filters.topic" class="neo-select w-full md:w-auto">
        <option value="">{{ t('history.filters.allTopics') }}</option>
        <option v-for="topic in availableTopics" :key="topic" :value="topic">
          {{ formatTopicLabel(t, topic) }}
        </option>
      </select>
      <select v-model="filters.status" class="neo-select w-full md:w-auto">
        <option value="">{{ t('history.filters.allStatuses') }}</option>
        <option value="completed">
          {{ formatStatusLabel(t, 'completed') }}
        </option>
        <option value="waiting_answer">
          {{ formatStatusLabel(t, 'waiting_answer') }}
        </option>
        <option value="review_pending">
          {{ formatStatusLabel(t, 'review_pending') }}
        </option>
      </select>
    </div>

    <div v-if="isLoading" class="neo-panel bg-white">
      <p class="neo-note">{{ t('common.loading') }}</p>
    </div>

    <div v-else-if="!sessions.length" class="neo-panel bg-white">
      <p class="neo-note">{{ t('history.empty') }}</p>
    </div>

    <div v-else class="space-y-3">
      <router-link
        v-for="item in sessions"
        :key="item.id"
        :to="
          item.review_id ? `/reviews/${item.review_id}` : `/sessions/${item.id}`
        "
        class="neo-panel block space-y-3 bg-white transition-transform hover:-translate-y-0.5"
      >
        <div
          class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"
        >
          <span class="text-base font-black">
            {{ formatModeLabel(t, item.mode) }}
            <template v-if="item.topic">
              · {{ formatTopicLabel(t, item.topic) }}</template
            >
            <template v-if="item.project_name">
              · {{ item.project_name }}</template
            >
          </span>
          <span class="text-sm font-semibold">
            {{ formatStatusLabel(t, item.status) }}
          </span>
        </div>
        <div
          class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between"
        >
          <span v-if="item.job_target" class="neo-note">
            {{ item.job_target.title }}
          </span>
          <span v-else class="neo-note">{{ t('history.noJobTarget') }}</span>
          <span class="text-sm font-black">
            {{ item.total_score > 0 ? item.total_score : '—' }}
          </span>
        </div>
        <p class="neo-note text-xs">
          {{ new Date(item.updated_at).toLocaleString() }}
        </p>
      </router-link>
    </div>

    <div
      v-if="totalPages > 1"
      class="flex flex-col items-stretch gap-3 sm:flex-row sm:items-center sm:justify-center"
    >
      <button
        class="neo-button-dark w-full sm:w-auto"
        :disabled="currentPage <= 1"
        @click="currentPage--"
      >
        {{ t('history.prev') }}
      </button>
      <span class="text-center text-sm font-semibold">
        {{ currentPage }} / {{ totalPages }}
      </span>
      <button
        class="neo-button-dark w-full sm:w-auto"
        :disabled="currentPage >= totalPages"
        @click="currentPage++"
      >
        {{ t('history.next') }}
      </button>
    </div>
  </section>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed, reactive, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';

import { listSessions } from '../api/client';
import {
  formatModeLabel,
  formatStatusLabel,
  formatTopicLabel,
} from '../lib/labels';

const { t } = useI18n();
const currentPage = ref(1);
const filters = reactive({
  mode: '',
  topic: '',
  status: '',
});

const availableTopics = [
  'mixed',
  'go',
  'redis',
  'kafka',
  'mysql',
  'system_design',
  'distributed',
  'network',
  'microservice',
  'os',
  'docker_k8s',
];

watch(filters, () => {
  currentPage.value = 1;
});

const { data, isLoading } = useQuery({
  queryKey: ['sessions', currentPage, filters],
  queryFn: () =>
    listSessions({
      page: currentPage.value,
      per_page: 20,
      mode: filters.mode || undefined,
      topic: filters.topic || undefined,
      status: filters.status || undefined,
    }),
});

const sessions = computed(() => data.value?.items ?? []);
const totalPages = computed(() => data.value?.total_pages ?? 1);
</script>
