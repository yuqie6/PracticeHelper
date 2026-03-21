export const THEME_STORAGE_KEY = 'practicehelper.theme';

export type AppTheme = 'light' | 'dark';

export function isAppTheme(
  value: string | null | undefined,
): value is AppTheme {
  return value === 'light' || value === 'dark';
}

export function resolveThemePreference(
  savedValue: string | null | undefined,
  prefersDark: boolean,
): AppTheme {
  if (isAppTheme(savedValue)) {
    return savedValue;
  }

  return prefersDark ? 'dark' : 'light';
}

export function readStoredTheme(
  storage: Pick<Storage, 'getItem'> | null | undefined,
): AppTheme | null {
  if (!storage) {
    return null;
  }

  const stored = storage.getItem(THEME_STORAGE_KEY);
  return isAppTheme(stored) ? stored : null;
}

export function persistTheme(
  storage: Pick<Storage, 'setItem'> | null | undefined,
  theme: AppTheme,
) {
  storage?.setItem(THEME_STORAGE_KEY, theme);
}

export function applyTheme(
  theme: AppTheme,
  root: { dataset: DOMStringMap } | null | undefined,
) {
  if (!root) {
    return;
  }

  root.dataset.theme = theme;
}
