<template>
  <div class="app-shell min-h-screen bg-[var(--neo-bg)] text-[var(--neo-text)]">
    <header class="app-header">
      <div class="neo-page app-header-main">
        <div class="app-brand">
          <div class="space-y-3">
            <p class="neo-kicker bg-[var(--neo-yellow)]">{{ t('app.name') }}</p>
            <div class="space-y-2">
              <h1 class="app-brand-title">{{ t('app.title') }}</h1>
              <p class="app-brand-note">{{ activeNavLabel }}</p>
            </div>
          </div>

          <RouterLink
            v-if="activeImportSummary"
            to="/projects"
            class="app-import-notice"
          >
            {{ activeImportSummary }}
          </RouterLink>
        </div>

        <div class="app-controls">
          <div class="app-toggle-group">
            <span class="app-toggle-label">{{ t('app.language') }}</span>
            <div class="app-toggle-row">
              <button
                v-for="item in localeOptions"
                :key="item.value"
                type="button"
                class="app-toggle-chip"
                :aria-pressed="currentLocale === item.value"
                :class="
                  currentLocale === item.value ? 'app-toggle-chip-active' : ''
                "
                @click="switchLocale(item.value)"
              >
                {{ item.label }}
              </button>
            </div>
          </div>

          <div class="app-toggle-group">
            <span class="app-toggle-label">{{ t('app.theme') }}</span>
            <div class="app-toggle-row">
              <button
                v-for="item in themeOptions"
                :key="item.value"
                type="button"
                class="app-toggle-chip"
                :aria-pressed="currentTheme === item.value"
                :class="
                  currentTheme === item.value ? 'app-toggle-chip-active' : ''
                "
                @click="switchTheme(item.value)"
              >
                {{ item.label }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <div class="app-nav-shell">
        <div class="neo-page app-nav-wrap">
          <nav class="app-nav" aria-label="Main navigation">
            <RouterLink
              v-for="item in navItems"
              :key="item.to"
              :to="item.to"
              class="app-nav-link"
              :class="item.active ? 'app-nav-link-active' : ''"
            >
              {{ item.label }}
            </RouterLink>
          </nav>
        </div>
      </div>
    </header>

    <main class="app-main">
      <slot />
    </main>

    <ToastContainer />
  </div>
</template>

<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query';
import { computed, ref } from 'vue';
import { useI18n } from 'vue-i18n';
import { RouterLink, useRoute } from 'vue-router';

import { listImportJobs, type ProjectImportJob } from '../api/client';
import ToastContainer from './ToastContainer.vue';
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
const route = useRoute();

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

const navItems = computed(() =>
  [
    { label: t('app.nav.home'), to: '/', prefixes: ['/'] },
    { label: t('app.nav.profile'), to: '/profile', prefixes: ['/profile'] },
    {
      label: t('app.nav.jobs'),
      to: '/job-targets',
      prefixes: ['/job-targets'],
    },
    {
      label: t('app.nav.projects'),
      to: '/projects',
      prefixes: ['/projects'],
    },
    {
      label: t('app.nav.promptExperiments'),
      to: '/prompt-experiments',
      prefixes: ['/prompt-experiments'],
    },
    {
      label: t('app.nav.train'),
      to: '/train',
      prefixes: ['/train', '/sessions'],
    },
    {
      label: t('app.nav.history'),
      to: '/history',
      prefixes: ['/history', '/reviews'],
    },
  ].map((item) => ({
    ...item,
    active: isRouteActive(item.prefixes),
  })),
);
const activeNavLabel = computed(() => {
  const active = navItems.value.find((item) => item.active);
  return active?.label ?? t('app.nav.home');
});

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

function isRouteActive(prefixes: string[]): boolean {
  if (prefixes.length === 1 && prefixes[0] === '/') {
    return route.path === '/';
  }

  return prefixes.some(
    (prefix) => route.path === prefix || route.path.startsWith(`${prefix}/`),
  );
}
</script>

<style scoped>
.app-shell {
  position: relative;
}

.app-header {
  backdrop-filter: blur(10px);
  background: color-mix(in srgb, var(--neo-bg) 82%, transparent);
  border-bottom: 2px solid
    color-mix(in srgb, var(--neo-border) 18%, transparent);
  position: sticky;
  top: 0;
  z-index: 40;
}

.app-header-main {
  align-items: start;
  display: grid;
  gap: 1rem;
  padding-bottom: 1rem;
  padding-top: 1.25rem;
}

.app-brand {
  display: grid;
  gap: 0.9rem;
}

.app-brand-title {
  font-size: clamp(1.8rem, 4vw, 2.9rem);
  font-weight: 900;
  letter-spacing: -0.06em;
  line-height: 0.95;
  margin: 0;
  text-transform: uppercase;
}

.app-brand-note {
  font-size: 0.82rem;
  font-weight: 900;
  letter-spacing: 0.18em;
  line-height: 1.4;
  margin: 0;
  opacity: 0.72;
  text-transform: uppercase;
}

.app-import-notice {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  box-shadow: 4px 4px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  display: inline-flex;
  font-size: 0.78rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  max-width: 100%;
  padding: 0.75rem 0.9rem;
  text-transform: uppercase;
}

.app-controls {
  display: grid;
  gap: 0.75rem;
}

.app-toggle-group {
  align-items: start;
  display: grid;
  gap: 0.45rem;
}

.app-toggle-label {
  font-size: 0.72rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.app-toggle-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.app-toggle-chip {
  background: color-mix(in srgb, var(--neo-surface) 90%, transparent);
  border: 2px solid var(--neo-border);
  font-size: 0.74rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  min-height: 2.1rem;
  padding: 0.45rem 0.8rem;
  text-transform: uppercase;
  transition: background-color 180ms ease;
}

.app-toggle-chip-active {
  background: color-mix(in srgb, var(--neo-yellow) 70%, white);
}

.app-nav-shell {
  border-top: 1px solid color-mix(in srgb, var(--neo-border) 14%, transparent);
  padding-bottom: 0.75rem;
}

.app-nav-wrap {
  padding-bottom: 0;
  padding-top: 0.75rem;
}

.app-nav {
  display: flex;
  gap: 0.65rem;
  overflow-x: auto;
  padding-bottom: 0.15rem;
}

.app-nav-link {
  background: color-mix(in srgb, var(--neo-surface) 86%, transparent);
  border: 2px solid color-mix(in srgb, var(--neo-border) 24%, transparent);
  color: var(--neo-text);
  display: inline-flex;
  font-size: 0.8rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  min-height: 2.4rem;
  min-width: max-content;
  padding: 0.55rem 0.9rem;
  text-transform: uppercase;
  transition:
    transform 180ms ease,
    background-color 180ms ease,
    border-color 180ms ease;
}

.app-nav-link:hover {
  transform: translateY(-1px);
}

.app-nav-link-active {
  background: var(--neo-black);
  border-color: var(--neo-border);
  color: white;
}

.app-main {
  min-height: calc(100vh - 10rem);
}

@media (min-width: 1024px) {
  .app-header-main {
    align-items: end;
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .app-controls {
    justify-items: end;
  }
}
</style>
