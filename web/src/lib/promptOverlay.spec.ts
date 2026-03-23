import { describe, expect, it } from 'vitest';

import {
  formatPromptOverlaySummary,
  hasPromptOverlay,
  normalizePromptOverlay,
  toPromptOverlayFormState,
} from './promptOverlay';

const t = (key: string, named?: Record<string, unknown>) => {
  if (key === 'promptOverlay.focusTagSummary') {
    return `关注 ${named?.tags}`;
  }
  if (key === 'promptOverlay.customInstructionSummary') {
    return '含补充说明';
  }
  if (key === 'promptOverlay.enabledSummary') {
    return '已启用自定义风格';
  }
  return key;
};

describe('promptOverlay', () => {
  it('normalizes valid overlay fields and drops empty values', () => {
    expect(
      normalizePromptOverlay({
        tone: ' direct ',
        detail_level: 'detailed',
        followup_intensity: 'pressure',
        answer_language: 'en-us',
        focus_tags: ['depth', 'structure', 'depth', 'ignored'],
        custom_instruction: '  多问边界和取舍。  ',
      }),
    ).toEqual({
      tone: 'direct',
      detail_level: 'detailed',
      followup_intensity: 'pressure',
      answer_language: 'en-US',
      focus_tags: ['depth', 'structure'],
      custom_instruction: '多问边界和取舍。',
    });
  });

  it('returns undefined when overlay is effectively empty', () => {
    expect(
      normalizePromptOverlay({
        tone: 'unknown',
        detail_level: '',
        focus_tags: [],
        custom_instruction: '   ',
      }),
    ).toBeUndefined();
  });

  it('formats a compact overlay summary for UI', () => {
    const summary = formatPromptOverlaySummary(
      t,
      normalizePromptOverlay({
        tone: 'direct',
        detail_level: 'balanced',
        focus_tags: ['depth', 'structure'],
        custom_instruction: '多追问一点。',
      }),
    );

    expect(summary).toContain('promptOverlay.tone.direct');
    expect(summary).toContain('promptOverlay.detailLevel.balanced');
    expect(summary).toContain(
      '关注 promptOverlay.focusTags.depth、promptOverlay.focusTags.structure',
    );
    expect(summary).toContain('含补充说明');
  });

  it('can rebuild stable form state from stored overlay', () => {
    expect(
      toPromptOverlayFormState({
        tone: 'direct',
        focus_tags: ['depth'],
      }),
    ).toEqual({
      tone: 'direct',
      detail_level: '',
      followup_intensity: '',
      answer_language: '',
      focus_tags: ['depth'],
      custom_instruction: '',
    });
  });

  it('detects whether a session used prompt overlay', () => {
    expect(hasPromptOverlay(undefined, '')).toBe(false);
    expect(hasPromptOverlay(undefined, 'hash_1')).toBe(true);
    expect(hasPromptOverlay({ tone: 'direct' }, '')).toBe(true);
  });
});
