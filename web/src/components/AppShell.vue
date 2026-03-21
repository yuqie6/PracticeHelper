<template>
  <div class="min-h-screen bg-[var(--neo-bg)] text-[var(--neo-text)]">
    <header class="border-b-2 border-black bg-[var(--neo-red)] md:border-b-4">
      <div
        class="neo-page flex flex-col gap-4 md:flex-row md:items-center md:justify-between"
      >
        <div>
          <p class="neo-kicker bg-[var(--neo-yellow)]">{{ t('app.name') }}</p>
          <h1
            class="text-2xl font-black uppercase tracking-[0.08em] md:text-4xl"
          >
            {{ t('app.title') }}
          </h1>
          <RouterLink
            v-if="activeImportSummary"
            to="/projects"
            class="mt-3 inline-flex w-full border-2 border-black bg-white px-3 py-2 text-sm font-black uppercase tracking-[0.08em] shadow-[4px_4px_0px_0px_rgba(0,0,0,1)] md:w-auto md:border-4"
          >
            {{ activeImportSummary }}
          </RouterLink>
        </div>

        <div class="flex flex-col gap-3 md:items-end">
          <div class="flex flex-wrap items-center gap-2">
            <span class="text-sm font-black uppercase tracking-[0.08em]">
              {{ t('app.language') }}
            </span>
            <button
              v-for="item in localeOptions"
              :key="item.value"
              type="button"
              class="border-2 border-black px-3 py-1 text-xs font-black uppercase transition-colors md:border-4"
              :class="
                currentLocale === item.value
                  ? 'bg-[var(--neo-yellow)]'
                  : 'bg-white'
              "
              @click="switchLocale(item.value)"
            >
              {{ item.label }}
            </button>
          </div>

          <div class="flex flex-wrap items-center gap-2">
            <span class="text-sm font-black uppercase tracking-[0.08em]">
              {{ t('app.theme') }}
            </span>
            <button
              v-for="item in themeOptions"
              :key="item.value"
              type="button"
              class="border-2 border-black px-3 py-1 text-xs font-black uppercase transition-colors md:border-4"
              :class="
                currentTheme === item.value
                  ? 'bg-[var(--neo-yellow)]'
                  : 'bg-white'
              "
              @click="switchTheme(item.value)"
            >
              {{ item.label }}
            </button>
          </div>

          <nav
            class="flex gap-3 overflow-x-auto pb-1 sm:flex-wrap sm:justify-end"
          >
            <RouterLink
              v-for="item in navItems"
              :key="item.to"
              :to="item.to"
              class="neo-button-dark min-w-max shrink-0"
            >
              {{ item.label }}
            </RouterLink>
          </nav>
        </div>
      </div>
    </header>

    <main>
      <slot />
    </main>
  </div>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink } from 'vue-router';

import { listImportJobs, type ProjectImportJob } from '../api/client';
import { setLocale, type AppLocale } from '../i18n';
import { formatImportJobStageLabel } from '../lib/labels';
import {
  applyTheme,
  persistTheme,
  readStoredTheme,
  resolveThemePreference,
  type AppTheme,
} from '../lib/theme';

const { t, locale } = useI18n();

const currentLocale = computed(() => locale.value as AppLocale);
const currentTheme = ref<AppTheme>(
  resolveThemePreference(readStoredTheme(getStorage()), prefersDarkMode()),
);

applyTheme(currentTheme.value, getThemeRoot());

const { data: importJobsData } = useQuery({
  queryKey: ['import-jobs'],
  queryFn: listImportJobs,
  refetchInterval: (query) => {
    const jobs = (query.state.data as ProjectImportJob[] | undefined) ?? [];
    return jobs.some((job) => ['queued', 'running'].includes(job.status))
      ? 3000
      : false;
  },
});

const activeImportSummary = computed(() => {
  const job = (importJobsData.value ?? []).find((item) =>
    ['queued', 'running'].includes(item.status),
  );
  if (!job) {
    return '';
  }

  return t('app.importNotice', {
    stage: formatImportJobStageLabel(t, job.stage),
  });
});

const navItems = computed(() => [
  { label: t('app.nav.home'), to: '/' },
  { label: t('app.nav.profile'), to: '/profile' },
  { label: t('app.nav.jobs'), to: '/job-targets' },
  { label: t('app.nav.projects'), to: '/projects' },
  { label: t('app.nav.promptExperiments'), to: '/prompt-experiments' },
  { label: t('app.nav.train'), to: '/train' },
  { label: t('app.nav.history'), to: '/history' },
]);

const localeOptions = computed(() => [
  { value: 'zh-CN' as AppLocale, label: t('app.locales.zhCN') },
  { value: 'en' as AppLocale, label: t('app.locales.en') },
]);

const themeOptions = computed(() => [
  { value: 'light' as AppTheme, label: t('app.themes.light') },
  { value: 'dark' as AppTheme, label: t('app.themes.dark') },
]);

function switchLocale(nextLocale: AppLocale) {
  setLocale(nextLocale);
}

function switchTheme(nextTheme: AppTheme) {
  currentTheme.value = nextTheme;
  persistTheme(getStorage(), nextTheme);
  applyTheme(nextTheme, getThemeRoot());
}

function getStorage(): Storage | null {
  if (typeof window === 'undefined') {
    return null;
  }
  return window.localStorage;
}

function getThemeRoot(): HTMLElement | null {
  if (typeof document === 'undefined') {
    return null;
  }
  return document.documentElement;
}

function prefersDarkMode(): boolean {
  if (
    typeof window === 'undefined' ||
    typeof window.matchMedia !== 'function'
  ) {
    return false;
  }

  return window.matchMedia('(prefers-color-scheme: dark)').matches;
}
</script>
