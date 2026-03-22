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
              :aria-current="item.active ? 'page' : undefined"
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
  backdrop-filter: blur(6px);
  background: color-mix(in srgb, var(--neo-bg) 90%, transparent);
  border-bottom: 1px solid
    color-mix(in srgb, var(--neo-border) 16%, transparent);
  position: sticky;
  top: 0;
  z-index: 40;
}

.app-header-main {
  align-items: start;
  display: grid;
  gap: 0.6rem;
  padding-bottom: 0.45rem;
  padding-top: 0.55rem;
}

.app-brand {
  display: grid;
  gap: 0.5rem;
}

.app-brand-title {
  font-size: clamp(1.1rem, 2.5vw, 1.55rem);
  font-weight: 900;
  letter-spacing: -0.04em;
  line-height: 1;
  margin: 0;
  text-transform: uppercase;
}

.app-brand-note {
  font-size: 0.62rem;
  font-weight: 900;
  letter-spacing: 0.14em;
  line-height: 1.4;
  margin: 0;
  opacity: 0.56;
  text-transform: uppercase;
}

.app-import-notice {
  background: color-mix(in srgb, var(--neo-yellow) 18%, var(--neo-bg));
  border: 2px solid var(--neo-border);
  box-shadow: 3px 3px 0 0
    rgba(var(--neo-shadow-rgb), calc(var(--neo-shadow-alpha) * 0.9));
  display: inline-flex;
  font-size: 0.66rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  max-width: 100%;
  padding: 0.42rem 0.62rem;
  text-transform: uppercase;
  transition:
    transform var(--motion-duration-base) var(--motion-ease-standard),
    box-shadow var(--motion-duration-base) var(--motion-ease-standard),
    background-color var(--motion-duration-fast) var(--motion-ease-soft);
}

.app-import-notice:hover {
  box-shadow: 5px 5px 0 0 rgba(var(--neo-shadow-rgb), var(--neo-shadow-alpha));
  transform: translate(var(--motion-lift-sm), var(--motion-lift-sm));
}

.app-controls {
  display: grid;
  gap: 0.4rem;
}

.app-toggle-group {
  align-items: start;
  display: grid;
  gap: 0.22rem;
}

.app-toggle-label {
  font-size: 0.6rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.app-toggle-row {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
}

.app-toggle-chip {
  background: color-mix(in srgb, var(--neo-bg) 86%, transparent);
  border: 1px solid color-mix(in srgb, var(--neo-border) 26%, transparent);
  box-shadow: 2px 2px 0 0
    rgba(var(--neo-shadow-rgb), calc(var(--neo-shadow-alpha) * 0.55));
  font-size: 0.68rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  min-height: 1.65rem;
  padding: 0.24rem 0.52rem;
  text-transform: uppercase;
  transition:
    transform var(--motion-duration-base) var(--motion-ease-standard),
    box-shadow var(--motion-duration-base) var(--motion-ease-standard),
    background-color var(--motion-duration-fast) var(--motion-ease-soft),
    border-color var(--motion-duration-fast) var(--motion-ease-soft);
}

.app-toggle-chip:hover {
  border-color: color-mix(in srgb, var(--neo-border) 40%, transparent);
  box-shadow: 3px 3px 0 0
    rgba(var(--neo-shadow-rgb), calc(var(--neo-shadow-alpha) * 0.72));
  transform: translate(var(--motion-lift-sm), var(--motion-lift-sm));
}

.app-toggle-chip-active {
  background: color-mix(in srgb, var(--neo-yellow) 34%, var(--neo-bg));
  border-color: var(--neo-border);
  box-shadow: 3px 3px 0 0
    rgba(var(--neo-shadow-rgb), calc(var(--neo-shadow-alpha) * 0.7));
}

.app-nav-shell {
  border-top: 1px solid color-mix(in srgb, var(--neo-border) 14%, transparent);
  padding-bottom: 0.28rem;
}

.app-nav-wrap {
  padding-bottom: 0;
  padding-top: 0.32rem;
}

.app-nav {
  display: flex;
  gap: 0.35rem;
  overflow-x: auto;
  padding-bottom: 0.1rem;
}

.app-nav-link {
  background: color-mix(in srgb, var(--neo-bg) 82%, transparent);
  border: 1px solid color-mix(in srgb, var(--neo-border) 16%, transparent);
  color: var(--neo-text);
  display: inline-flex;
  font-size: 0.66rem;
  font-weight: 900;
  letter-spacing: 0.08em;
  min-height: 1.8rem;
  min-width: max-content;
  overflow: hidden;
  padding: 0.3rem 0.58rem;
  position: relative;
  text-transform: uppercase;
  transition:
    transform var(--motion-duration-base) var(--motion-ease-standard),
    box-shadow var(--motion-duration-base) var(--motion-ease-standard),
    background-color var(--motion-duration-fast) var(--motion-ease-soft),
    border-color var(--motion-duration-fast) var(--motion-ease-soft),
    color var(--motion-duration-fast) var(--motion-ease-soft);
}

.app-nav-link::after {
  background: color-mix(in srgb, var(--neo-yellow) 88%, var(--neo-red) 12%);
  bottom: 0.24rem;
  content: '';
  height: 2px;
  left: 0.52rem;
  opacity: 0;
  position: absolute;
  right: 0.52rem;
  transform: scaleX(0.35);
  transform-origin: left;
  transition:
    opacity var(--motion-duration-fast) var(--motion-ease-soft),
    transform var(--motion-duration-base) var(--motion-ease-standard);
}

.app-nav-link:hover {
  background: color-mix(in srgb, var(--neo-paper) 56%, var(--neo-bg));
  border-color: color-mix(in srgb, var(--neo-border) 28%, transparent);
  box-shadow: 3px 3px 0 0
    rgba(var(--neo-shadow-rgb), calc(var(--neo-shadow-alpha) * 0.7));
  transform: translate(var(--motion-lift-sm), var(--motion-lift-sm));
}

.app-nav-link:hover::after {
  opacity: 0.72;
  transform: scaleX(1);
}

.app-nav-link-active {
  background: color-mix(in srgb, var(--neo-yellow) 22%, var(--neo-bg));
  border-color: color-mix(in srgb, var(--neo-border) 32%, transparent);
  box-shadow: 3px 3px 0 0
    rgba(var(--neo-shadow-rgb), calc(var(--neo-shadow-alpha) * 0.72));
  color: var(--neo-text);
}

.app-nav-link-active::after {
  opacity: 1;
  transform: scaleX(1);
}

.app-main {
  min-height: calc(100vh - 6.4rem);
}

@media (min-width: 1024px) {
  .app-header-main {
    align-items: end;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 1rem;
  }

  .app-controls {
    justify-items: end;
  }
}

@media (prefers-reduced-motion: reduce) {
  .app-import-notice,
  .app-toggle-chip,
  .app-nav-link,
  .app-nav-link::after {
    transition: none;
  }

  .app-import-notice:hover,
  .app-toggle-chip:hover,
  .app-nav-link:hover {
    box-shadow: inherit;
    transform: none;
  }

  .app-nav-link::after,
  .app-nav-link:hover::after,
  .app-nav-link-active::after {
    opacity: 1;
    transform: none;
  }
}
</style>
