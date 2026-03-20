export type Translate = (
  key: string,
  named?: Record<string, unknown>,
) => string;

function humanizeToken(value: string): string {
  return value.replaceAll('_', ' ');
}

function translateOrFallback(
  t: Translate,
  key: string,
  fallback: string,
  named?: Record<string, unknown>,
): string {
  const translated = t(key, named);
  return translated === key ? fallback : translated;
}

export function formatModeLabel(t: Translate, mode: string): string {
  return translateOrFallback(t, `enums.mode.${mode}`, humanizeToken(mode));
}

export function formatIntensityLabel(t: Translate, intensity: string): string {
  return translateOrFallback(
    t,
    `enums.intensity.${intensity}`,
    humanizeToken(intensity),
  );
}

export function formatTopicLabel(t: Translate, topic: string): string {
  return translateOrFallback(t, `enums.topic.${topic}`, topic);
}

export function formatStatusLabel(t: Translate, status: string): string {
  return translateOrFallback(
    t,
    `enums.status.${status}`,
    humanizeToken(status),
  );
}

export function formatImportStatusLabel(t: Translate, status: string): string {
  return translateOrFallback(
    t,
    `enums.importStatus.${status}`,
    humanizeToken(status),
  );
}

export function formatImportJobStatusLabel(
  t: Translate,
  status: string,
): string {
  return translateOrFallback(
    t,
    `enums.importJobStatus.${status}`,
    humanizeToken(status),
  );
}

export function formatImportJobStageLabel(t: Translate, stage: string): string {
  return translateOrFallback(
    t,
    `enums.importJobStage.${stage}`,
    humanizeToken(stage),
  );
}

export function formatJobTargetAnalysisStatusLabel(
  t: Translate,
  status: string,
): string {
  return translateOrFallback(
    t,
    `enums.jobTargetAnalysisStatus.${status}`,
    humanizeToken(status),
  );
}

export function formatWeaknessKindLabel(t: Translate, kind: string): string {
  return translateOrFallback(
    t,
    `enums.weaknessKind.${kind}`,
    humanizeToken(kind),
  );
}
