import { createI18n } from 'vue-i18n';

import { messages } from './messages';

export const SUPPORTED_LOCALES = ['zh-CN', 'en'] as const;
export type AppLocale = (typeof SUPPORTED_LOCALES)[number];

const STORAGE_KEY = 'practicehelper.locale';

function isSupportedLocale(value: string): value is AppLocale {
  return SUPPORTED_LOCALES.includes(value as AppLocale);
}

function detectBrowserLocale(): AppLocale {
  if (typeof navigator === 'undefined') {
    return 'zh-CN';
  }

  return navigator.language.toLowerCase().startsWith('zh') ? 'zh-CN' : 'en';
}

function getInitialLocale(): AppLocale {
  if (typeof window === 'undefined') {
    return 'zh-CN';
  }

  const saved = window.localStorage.getItem(STORAGE_KEY);
  if (saved && isSupportedLocale(saved)) {
    return saved;
  }

  return detectBrowserLocale();
}

export const i18n = createI18n({
  legacy: false,
  locale: getInitialLocale(),
  fallbackLocale: 'en',
  messages,
});

export function setLocale(locale: AppLocale) {
  i18n.global.locale.value = locale;

  if (typeof window !== 'undefined') {
    window.localStorage.setItem(STORAGE_KEY, locale);
  }
}
