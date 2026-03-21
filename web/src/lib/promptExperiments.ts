import type { PromptSetSummary } from '../api/client';

export function resolvePromptExperimentSelection(
  promptSets: PromptSetSummary[],
  preferredLeft?: string,
  preferredRight?: string,
): { left: string; right: string } {
  if (promptSets.length === 0) {
    return { left: '', right: '' };
  }

  const availableIds = new Set(promptSets.map((item) => item.id));
  const defaultLeft =
    (preferredLeft && availableIds.has(preferredLeft) && preferredLeft) ||
    promptSets.find((item) => item.is_default)?.id ||
    promptSets[0].id;

  const defaultRight =
    (preferredRight &&
      preferredRight !== defaultLeft &&
      availableIds.has(preferredRight) &&
      preferredRight) ||
    promptSets.find((item) => item.id !== defaultLeft)?.id ||
    defaultLeft;

  return {
    left: defaultLeft,
    right: defaultRight,
  };
}

export function buildPromptExperimentLink(promptSetId?: string): string {
  if (!promptSetId) {
    return '/prompt-experiments';
  }

  return `/prompt-experiments?left=${encodeURIComponent(promptSetId)}`;
}
