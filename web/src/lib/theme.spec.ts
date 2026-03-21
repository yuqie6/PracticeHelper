import { describe, expect, it } from 'vitest';

import { applyTheme, readStoredTheme, resolveThemePreference } from './theme';

describe('theme helpers', () => {
  it('uses stored theme when it is valid', () => {
    expect(resolveThemePreference('dark', false)).toBe('dark');
    expect(resolveThemePreference('light', true)).toBe('light');
  });

  it('falls back to system preference when storage is empty or invalid', () => {
    expect(resolveThemePreference(null, true)).toBe('dark');
    expect(resolveThemePreference('unknown', false)).toBe('light');
  });

  it('reads stored theme and applies it to dataset', () => {
    const storage = {
      getItem: () => 'dark',
    };
    const root = { dataset: {} };

    const theme = readStoredTheme(storage);
    applyTheme(theme ?? 'light', root);

    expect(theme).toBe('dark');
    expect(root.dataset.theme).toBe('dark');
  });
});
