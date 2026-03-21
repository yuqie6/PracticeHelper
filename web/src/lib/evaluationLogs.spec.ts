import { describe, expect, it } from 'vitest';

import {
  formatEvaluationRawOutput,
  hasEvaluationRawOutput,
} from './evaluationLogs';

describe('evaluation log helpers', () => {
  it('detects whether a raw output payload is present', () => {
    expect(hasEvaluationRawOutput('{"score": 88}')).toBe(true);
    expect(hasEvaluationRawOutput('   ')).toBe(false);
    expect(hasEvaluationRawOutput(undefined)).toBe(false);
  });

  it('trims raw output for display', () => {
    expect(formatEvaluationRawOutput('  {"score": 88}\n')).toBe(
      '{"score": 88}',
    );
    expect(formatEvaluationRawOutput('')).toBe('');
  });
});
