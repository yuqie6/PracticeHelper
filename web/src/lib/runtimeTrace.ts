import type { RuntimeTraceEntry } from '../api/client';
import type { Translate } from './labels';

function humanizeToken(value: string): string {
  return value.replaceAll('_', ' ');
}

function translateOrFallback(
  t: Translate,
  key: string,
  fallback: string,
): string {
  const translated = t(key);
  return translated === key ? fallback : translated;
}

export function normalizeRuntimeTraceEntry(
  value: unknown,
): RuntimeTraceEntry | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return null;
  }

  const record = value as Record<string, unknown>;
  if (typeof record.flow !== 'string' || typeof record.phase !== 'string') {
    return null;
  }

  return {
    flow: record.flow,
    phase: record.phase,
    status: typeof record.status === 'string' ? record.status : 'info',
    code: typeof record.code === 'string' ? record.code : '',
    message: typeof record.message === 'string' ? record.message : '',
    attempt: typeof record.attempt === 'number' ? record.attempt : 0,
    tool_name: typeof record.tool_name === 'string' ? record.tool_name : '',
  };
}

export function formatRuntimeTracePhaseLabel(
  t: Translate,
  phase: string,
): string {
  return translateOrFallback(
    t,
    `session.tracePhases.${phase}`,
    humanizeToken(phase),
  );
}

export function formatRuntimeTraceStatusLabel(
  t: Translate,
  status: string,
): string {
  return translateOrFallback(
    t,
    `session.traceStatuses.${status}`,
    humanizeToken(status),
  );
}
