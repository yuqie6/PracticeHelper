import { describe, expect, it } from 'vitest';

import {
  buildPromptExperimentLink,
  resolvePromptExperimentSelection,
} from './promptExperiments';

describe('prompt experiment helpers', () => {
  const promptSets = [
    { id: 'stable-v1', label: 'Stable v1', status: 'stable', is_default: true },
    { id: 'candidate-v1', label: 'Candidate v1', status: 'candidate' },
  ];

  it('prefers requested versions when they are valid', () => {
    expect(
      resolvePromptExperimentSelection(promptSets, 'candidate-v1', 'stable-v1'),
    ).toEqual({
      left: 'candidate-v1',
      right: 'stable-v1',
    });
  });

  it('falls back to default left and another right version', () => {
    expect(resolvePromptExperimentSelection(promptSets)).toEqual({
      left: 'stable-v1',
      right: 'candidate-v1',
    });
  });

  it('builds prompt experiment links from prompt set ids', () => {
    expect(buildPromptExperimentLink('candidate-v1')).toBe(
      '/prompt-experiments?left=candidate-v1',
    );
  });
});
