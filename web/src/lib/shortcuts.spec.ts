import { describe, expect, it } from 'vitest';

import { isSubmitShortcut } from './shortcuts';

describe('submit shortcut helper', () => {
  it('accepts Ctrl+Enter as submit shortcut', () => {
    expect(
      isSubmitShortcut({
        key: 'Enter',
        ctrlKey: true,
      }),
    ).toBe(true);
  });

  it('rejects plain Enter and modifier combinations', () => {
    expect(
      isSubmitShortcut({
        key: 'Enter',
        ctrlKey: false,
      }),
    ).toBe(false);

    expect(
      isSubmitShortcut({
        key: 'Enter',
        ctrlKey: true,
        shiftKey: true,
      }),
    ).toBe(false);
  });
});
