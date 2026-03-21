export function hasEvaluationRawOutput(rawOutput?: string | null): boolean {
  return Boolean(rawOutput && rawOutput.trim());
}

export function formatEvaluationRawOutput(rawOutput?: string | null): string {
  if (!hasEvaluationRawOutput(rawOutput)) {
    return '';
  }
  return rawOutput!.trim();
}
