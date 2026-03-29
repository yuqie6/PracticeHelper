import type { ApiError } from '../api/client';
import type { Translate } from './labels';

const ERROR_KEY_BY_CODE: Record<string, string> = {
  session_busy: 'session.conflictBusy',
  session_review_pending: 'session.conflictReviewPending',
  session_completed: 'session.conflictCompleted',
  session_not_recoverable: 'session.retryReviewNotRecoverable',
  session_answer_conflict: 'session.conflictInvalidStatus',
  review_generation_retry: 'session.reviewGenerationRetry',
  tool_context_missing: 'common.apiErrors.toolContextMissing',
  tool_call_failed: 'common.apiErrors.toolCallFailed',
  tool_loop_exhausted: 'common.apiErrors.toolLoopExhausted',
  json_parse_failed: 'common.apiErrors.jsonParseFailed',
  semantic_validation_failed: 'common.apiErrors.semanticValidationFailed',
  single_shot_failed: 'common.apiErrors.singleShotFailed',
  backend_callback_failed: 'common.apiErrors.backendCallbackFailed',
  timeout: 'common.apiErrors.timeout',
  canceled: 'common.apiErrors.canceled',
  llm_required: 'common.apiErrors.llmRequired',
  unknown_error: 'common.apiErrors.unknownError',
};

export function resolveApiErrorMessage(
  t: Translate,
  error: ApiError,
  fallbackMessage = error.message,
): string {
  if (!error.code) {
    return fallbackMessage;
  }

  const key = ERROR_KEY_BY_CODE[error.code];
  if (!key) {
    return fallbackMessage;
  }

  const translated = t(key);
  return translated === key ? fallbackMessage : translated;
}
