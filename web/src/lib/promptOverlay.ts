import type { PromptOverlay } from '../api/client';
import type { Translate } from './labels';

export const PROMPT_OVERLAY_TONES = ['supportive', 'direct', 'strict'] as const;
export const PROMPT_OVERLAY_DETAIL_LEVELS = [
  'concise',
  'balanced',
  'detailed',
] as const;
export const PROMPT_OVERLAY_FOLLOWUP_INTENSITIES = [
  'light',
  'standard',
  'pressure',
] as const;
export const PROMPT_OVERLAY_LANGUAGES = ['zh-CN', 'en-US'] as const;
export const PROMPT_OVERLAY_FOCUS_TAGS = [
  'expression',
  'structure',
  'depth',
  'practicality',
  'confidence',
] as const;

const MAX_FOCUS_TAGS = 2;
const MAX_CUSTOM_INSTRUCTION_LENGTH = 280;

export interface PromptOverlayFormState {
  tone: string;
  detail_level: string;
  followup_intensity: string;
  answer_language: string;
  focus_tags: string[];
  custom_instruction: string;
}

export function emptyPromptOverlayForm(): PromptOverlayFormState {
  return {
    tone: '',
    detail_level: '',
    followup_intensity: '',
    answer_language: '',
    focus_tags: [],
    custom_instruction: '',
  };
}

export function toPromptOverlayFormState(
  overlay?: PromptOverlay | null,
): PromptOverlayFormState {
  const normalized = normalizePromptOverlay(overlay);
  return {
    tone: normalized?.tone ?? '',
    detail_level: normalized?.detail_level ?? '',
    followup_intensity: normalized?.followup_intensity ?? '',
    answer_language: normalized?.answer_language ?? '',
    focus_tags: normalized?.focus_tags ?? [],
    custom_instruction: normalized?.custom_instruction ?? '',
  };
}

export function normalizePromptOverlay(
  overlay?: PromptOverlay | PromptOverlayFormState | null,
): PromptOverlay | undefined {
  if (!overlay) {
    return undefined;
  }

  const tone = normalizeChoice(overlay.tone, PROMPT_OVERLAY_TONES);
  const detailLevel = normalizeChoice(
    overlay.detail_level,
    PROMPT_OVERLAY_DETAIL_LEVELS,
  );
  const followupIntensity = normalizeChoice(
    overlay.followup_intensity,
    PROMPT_OVERLAY_FOLLOWUP_INTENSITIES,
  );
  const answerLanguage = normalizeChoice(
    overlay.answer_language,
    PROMPT_OVERLAY_LANGUAGES,
  );
  const focusTags = Array.from(
    new Set(
      (overlay.focus_tags ?? []).filter((item): item is string =>
        PROMPT_OVERLAY_FOCUS_TAGS.includes(
          item as (typeof PROMPT_OVERLAY_FOCUS_TAGS)[number],
        ),
      ),
    ),
  )
    .slice(0, MAX_FOCUS_TAGS)
    .sort();
  const customInstruction = (overlay.custom_instruction ?? '')
    .trim()
    .slice(0, MAX_CUSTOM_INSTRUCTION_LENGTH);

  if (
    !tone &&
    !detailLevel &&
    !followupIntensity &&
    !answerLanguage &&
    focusTags.length === 0 &&
    !customInstruction
  ) {
    return undefined;
  }

  return {
    ...(tone ? { tone } : {}),
    ...(detailLevel ? { detail_level: detailLevel } : {}),
    ...(followupIntensity ? { followup_intensity: followupIntensity } : {}),
    ...(answerLanguage ? { answer_language: answerLanguage } : {}),
    ...(focusTags.length ? { focus_tags: focusTags } : {}),
    ...(customInstruction ? { custom_instruction: customInstruction } : {}),
  };
}

export function hasPromptOverlay(
  overlay?: PromptOverlay | null,
  overlayHash?: string,
): boolean {
  return Boolean(overlayHash) || Boolean(normalizePromptOverlay(overlay));
}

export function formatPromptOverlaySummary(
  t: Translate,
  overlay?: PromptOverlay | null,
  overlayHash?: string,
): string {
  const normalized = normalizePromptOverlay(overlay);
  if (!normalized) {
    return overlayHash ? t('promptOverlay.enabledSummary') : '';
  }

  const parts: string[] = [];
  if (normalized.tone) {
    parts.push(t(`promptOverlay.tone.${normalized.tone}`));
  }
  if (normalized.detail_level) {
    parts.push(t(`promptOverlay.detailLevel.${normalized.detail_level}`));
  }
  if (normalized.followup_intensity) {
    parts.push(
      t(`promptOverlay.followupIntensity.${normalized.followup_intensity}`),
    );
  }
  if (normalized.answer_language) {
    parts.push(t(`promptOverlay.answerLanguage.${normalized.answer_language}`));
  }
  if (normalized.focus_tags?.length) {
    const tags = normalized.focus_tags
      .map((item) => t(`promptOverlay.focusTags.${item}`))
      .join('、');
    parts.push(t('promptOverlay.focusTagSummary', { tags }));
  }
  if (normalized.custom_instruction) {
    parts.push(t('promptOverlay.customInstructionSummary'));
  }

  return parts.join(' / ');
}

function normalizeChoice<T extends readonly string[]>(
  value: string | undefined,
  allowed: T,
): T[number] | '' {
  const trimmed = value?.trim() ?? '';
  if (!trimmed) {
    return '';
  }

  const match = allowed.find(
    (item) => item.toLowerCase() === trimmed.toLowerCase(),
  );
  return match ?? '';
}
