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

  // 已保存的语言选择优先级高于浏览器探测，避免用户手动切换后又被系统语言反向覆盖。
  // 无浏览器环境统一回退中文，是为了让 SSR/测试环境首屏行为稳定且可预测。
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
