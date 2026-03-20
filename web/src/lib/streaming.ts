import type { StreamEvent } from '../api/client';

export interface StreamSection {
  id: string;
  phase: string;
  contexts: string[];
  reasoning: string[];
  rawContent: string;
}

export type ParsedStreamPayload =
  | {
      kind: 'question';
      question: string;
      expectedPoints: string[];
    }
  | {
      kind: 'evaluation';
      score: number;
      scoreBreakdown: Record<string, number>;
      headline: string;
      strengths: string[];
      gaps: string[];
      suggestion: string;
      followupIntent: string;
      followupQuestion: string;
      followupExpectedPoints: string[];
    }
  | {
      kind: 'review';
      overall: string;
      topFix: string;
      topFixReason: string;
      highlights: string[];
      gaps: string[];
      suggestedTopics: string[];
      nextTrainingFocus: string[];
      recommendedNext: {
        mode: string;
        topic: string;
        projectId: string;
        reason: string;
      } | null;
      scoreBreakdown: Record<string, number>;
    };

export function appendStreamEvent(
  sections: StreamSection[],
  event: StreamEvent,
): StreamSection[] {
  const next = sections.map((section) => ({
    ...section,
    contexts: [...section.contexts],
    reasoning: [...section.reasoning],
  }));

  let current = next[next.length - 1];
  const shouldStartNewSection =
    event.type === 'phase' &&
    event.phase === 'prepare_context' &&
    current != null &&
    hasSectionActivity(current);

  if (!current || shouldStartNewSection) {
    current = createStreamSection();
    next.push(current);
  }

  switch (event.type) {
    case 'phase':
      current.phase = event.phase ?? current.phase;
      break;
    case 'context':
      if (event.name) {
        current.contexts.push(event.name);
      }
      break;
    case 'reasoning':
      if (event.text) {
        current.reasoning.push(event.text);
      }
      break;
    case 'content':
      if (event.text) {
        current.rawContent += event.text;
      }
      break;
    default:
      break;
  }

  return next;
}

export function parseStreamPayload(
  rawContent: string,
): ParsedStreamPayload | null {
  const candidate = extractJsonBlock(rawContent);
  if (!candidate) {
    return null;
  }

  let data: unknown;
  try {
    data = JSON.parse(candidate);
  } catch {
    return null;
  }

  if (!data || typeof data !== 'object') {
    return null;
  }

  const record = data as Record<string, unknown>;

  if (typeof record.question === 'string') {
    return {
      kind: 'question',
      question: record.question,
      expectedPoints: asStringArray(record.expected_points),
    };
  }

  if (typeof record.score === 'number') {
    return {
      kind: 'evaluation',
      score: record.score,
      scoreBreakdown: asNumberRecord(record.score_breakdown),
      headline: asString(record.headline),
      strengths: asStringArray(record.strengths),
      gaps: asStringArray(record.gaps),
      suggestion: asString(record.suggestion),
      followupIntent: asString(record.followup_intent),
      followupQuestion: asString(record.followup_question),
      followupExpectedPoints: asStringArray(record.followup_expected_points),
    };
  }

  if (typeof record.overall === 'string') {
    return {
      kind: 'review',
      overall: record.overall,
      topFix: asString(record.top_fix),
      topFixReason: asString(record.top_fix_reason),
      highlights: asStringArray(record.highlights),
      gaps: asStringArray(record.gaps),
      suggestedTopics: asStringArray(record.suggested_topics),
      nextTrainingFocus: asStringArray(record.next_training_focus),
      recommendedNext: asNextSession(record.recommended_next),
      scoreBreakdown: asNumberRecord(record.score_breakdown),
    };
  }

  return null;
}

function createStreamSection(): StreamSection {
  return {
    id: `stream-${Math.random().toString(36).slice(2, 10)}`,
    phase: '',
    contexts: [],
    reasoning: [],
    rawContent: '',
  };
}

function hasSectionActivity(section: StreamSection): boolean {
  return Boolean(
    section.phase ||
    section.contexts.length ||
    section.reasoning.length ||
    section.rawContent.trim(),
  );
}

function extractJsonBlock(rawContent: string): string | null {
  const stripped = rawContent.trim();
  if (!stripped) {
    return null;
  }

  if (isValidJson(stripped)) {
    return stripped;
  }

  const start = stripped.indexOf('{');
  const end = stripped.lastIndexOf('}');
  if (start === -1 || end === -1 || end <= start) {
    return null;
  }

  const candidate = stripped.slice(start, end + 1);
  return isValidJson(candidate) ? candidate : null;
}

function isValidJson(value: string): boolean {
  try {
    JSON.parse(value);
    return true;
  } catch {
    return false;
  }
}

function asString(value: unknown): string {
  return typeof value === 'string' ? value : '';
}

function asStringArray(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : [];
}

function asNumberRecord(value: unknown): Record<string, number> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {};
  }

  return Object.fromEntries(
    Object.entries(value).filter(
      (entry): entry is [string, number] => typeof entry[1] === 'number',
    ),
  );
}

function asNextSession(value: unknown): {
  mode: string;
  topic: string;
  projectId: string;
  reason: string;
} | null {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return null;
  }

  const record = value as Record<string, unknown>;
  const mode = asString(record.mode);
  if (!mode) {
    return null;
  }

  return {
    mode,
    topic: asString(record.topic),
    projectId: asString(record.project_id),
    reason: asString(record.reason),
  };
}
