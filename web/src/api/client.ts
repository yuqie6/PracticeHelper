import {
  buildSessionExportPath,
  resolveBatchDownloadFilename,
  resolveDownloadFilename,
  type SessionExportFormat,
  SESSION_BATCH_EXPORT_PATH,
  SESSION_EXPORT_FORMAT,
} from '../lib/export';
import type {
  Dashboard,
  EvaluationLogEntry,
  JobTarget,
  JobTargetAnalysisRun,
  PaginatedList,
  ProjectImportJob,
  ProjectProfile,
  PromptExperimentReport,
  PromptSetSummary,
  ReviewCard,
  ReviewScheduleItem,
  StreamEvent,
  TrainingSession,
  TrainingSessionSummary,
  UserProfile,
  WeaknessTag,
  WeaknessTrend,
} from './types';
export type {
  Dashboard,
  EvaluationLogEntry,
  JobTarget,
  JobTargetAnalysisRun,
  JobTargetRef,
  MemoryRetrievalHit,
  MemoryRetrievalTrace,
  PaginatedList,
  ProjectImportJob,
  ProjectProfile,
  PromptExperimentMetrics,
  PromptExperimentReport,
  PromptExperimentSample,
  PromptSetSummary,
  RetrievalTrace,
  ReviewCard,
  ReviewScheduleItem,
  RuntimeTrace,
  RuntimeTraceEntry,
  StreamEvent,
  TrainingEvaluation,
  TrainingSession,
  TrainingSessionSummary,
  TrainingTurn,
  UserProfile,
  WeaknessTag,
  WeaknessTrend,
  WeaknessTrendPoint,
} from './types';

export interface ApiEnvelope<T> {
  data: T;
}

interface ApiErrorPayload {
  error?: {
    code?: string;
    message?: string;
  };
}

export class ApiError extends Error {
  code?: string;
  status?: number;

  constructor(message: string, options?: { code?: string; status?: number }) {
    super(message);
    this.name = 'ApiError';
    this.code = options?.code;
    this.status = options?.status;
  }
}

function isAbortError(error: unknown): error is DOMException {
  return error instanceof DOMException && error.name === 'AbortError';
}

function toTimeoutError(): ApiError {
  return new ApiError('Request timeout', { code: 'timeout' });
}

function bindAbortSignal(
  source: AbortSignal | null | undefined,
  target: AbortController,
): () => void {
  if (!source) {
    return () => {};
  }
  if (source.aborted) {
    target.abort();
    return () => {};
  }

  const abort = () => target.abort();
  source.addEventListener('abort', abort, { once: true });
  return () => source.removeEventListener('abort', abort);
}

async function request<T>(
  path: string,
  init?: RequestInit,
  timeoutMs = 90_000,
): Promise<T> {
  // 这里封装的是当前 PracticeHelper API 的统一 JSON 契约：
  // 成功响应默认是 { data: T }，失败响应尽量从 { error.message } 提取可展示文案。
  // 如果后续出现 204、文件下载或非 JSON 接口，应该新增专用请求函数而不是继续复用这里。
  const controller = new AbortController();
  const cleanupAbort = bindAbortSignal(init?.signal, controller);
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  try {
    const response = await fetch(path, {
      headers: {
        'Content-Type': 'application/json',
        ...(init?.headers ?? {}),
      },
      ...init,
      signal: controller.signal,
    });

    if (!response.ok) {
      const payload = (await response
        .json()
        .catch(() => null)) as ApiErrorPayload | null;
      throw new ApiError(payload?.error?.message ?? '请求失败', {
        code: payload?.error?.code,
        status: response.status,
      });
    }

    const payload = (await response.json()) as ApiEnvelope<T>;
    return payload.data;
  } catch (error) {
    if (isAbortError(error)) {
      throw toTimeoutError();
    }
    throw error;
  } finally {
    clearTimeout(timer);
    cleanupAbort();
  }
}

async function requestStream<T>(
  path: string,
  init: RequestInit,
  onEvent: (event: StreamEvent) => void,
): Promise<T> {
  const controller = new AbortController();
  const cleanupAbort = bindAbortSignal(init.signal, controller);
  const totalTimer = setTimeout(() => controller.abort(), 300_000);
  let idleTimer: ReturnType<typeof setTimeout> | null = null;
  const resetIdle = () => {
    if (idleTimer != null) {
      clearTimeout(idleTimer);
    }
    idleTimer = setTimeout(() => controller.abort(), 120_000);
  };

  try {
    resetIdle();
    const response = await fetch(path, {
      headers: {
        'Content-Type': 'application/json',
        ...(init.headers ?? {}),
      },
      ...init,
      signal: controller.signal,
    });

    if (!response.ok || !response.body) {
      const payload = (await response
        .json()
        .catch(() => null)) as ApiErrorPayload | null;
      throw new ApiError(payload?.error?.message ?? '请求失败', {
        code: payload?.error?.code,
        status: response.status,
      });
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';
    let result: T | null = null;

    while (true) {
      const { value, done } = await reader.read();
      if (done) {
        break;
      }
      resetIdle();

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() ?? '';

      for (const rawLine of lines) {
        const line = rawLine.trim();
        if (!line) {
          continue;
        }

        const event = JSON.parse(line) as StreamEvent;
        onEvent(event);

        if (event.type === 'error') {
          throw new ApiError(event.message ?? '流式请求失败', {
            code: event.code,
            status: response.status,
          });
        }

        if (event.type === 'result') {
          result = event.data as T;
        }
      }
    }

    if (buffer.trim()) {
      const event = JSON.parse(buffer.trim()) as StreamEvent;
      onEvent(event);
      if (event.type === 'error') {
        throw new ApiError(event.message ?? '流式请求失败', {
          code: event.code,
          status: response.status,
        });
      }
      if (event.type === 'result') {
        result = event.data as T;
      }
    }

    if (result == null) {
      throw new Error('流式请求未返回最终结果');
    }

    return result;
  } catch (error) {
    if (isAbortError(error)) {
      throw toTimeoutError();
    }
    throw error;
  } finally {
    if (idleTimer != null) {
      clearTimeout(idleTimer);
    }
    clearTimeout(totalTimer);
    cleanupAbort();
  }
}

export function getDashboard(): Promise<Dashboard> {
  return request('/api/dashboard');
}

export function getProfile(): Promise<UserProfile | null> {
  return request('/api/profile');
}

export function saveProfile(
  payload: Partial<UserProfile>,
): Promise<UserProfile> {
  return request('/api/profile', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function importProject(repoUrl: string): Promise<ProjectImportJob> {
  return request('/api/projects/import', {
    method: 'POST',
    body: JSON.stringify({ repo_url: repoUrl }),
  });
}

export function listImportJobs(): Promise<ProjectImportJob[]> {
  return request('/api/import-jobs');
}

export function retryImportJob(jobId: string): Promise<ProjectImportJob> {
  return request(`/api/import-jobs/${jobId}/retry`, {
    method: 'POST',
  });
}

export function listProjects(): Promise<ProjectProfile[]> {
  return request('/api/projects');
}

export function listJobTargets(): Promise<JobTarget[]> {
  return request('/api/job-targets');
}

export function listPromptSets(): Promise<PromptSetSummary[]> {
  return request('/api/prompt-sets');
}

export function listPromptExperimentPromptSets(): Promise<PromptSetSummary[]> {
  return request('/api/prompt-experiments/prompt-sets');
}

export function getPromptExperiment(params: {
  left: string;
  right: string;
  mode?: string;
  topic?: string;
  limit?: number;
}): Promise<PromptExperimentReport> {
  const query = new URLSearchParams({
    left: params.left,
    right: params.right,
  });
  if (params.mode) query.set('mode', params.mode);
  if (params.topic) query.set('topic', params.topic);
  if (params.limit) query.set('limit', String(params.limit));
  return request(`/api/prompt-experiments?${query.toString()}`);
}

export function createJobTarget(payload: {
  title: string;
  company_name?: string;
  source_text: string;
}): Promise<JobTarget> {
  return request('/api/job-targets', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function getJobTarget(jobTargetId: string): Promise<JobTarget> {
  return request(`/api/job-targets/${jobTargetId}`);
}

export function updateJobTarget(
  jobTargetId: string,
  payload: {
    title: string;
    company_name?: string;
    source_text: string;
  },
): Promise<JobTarget> {
  return request(`/api/job-targets/${jobTargetId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  });
}

export function analyzeJobTarget(
  jobTargetId: string,
): Promise<JobTargetAnalysisRun> {
  return request(`/api/job-targets/${jobTargetId}/analyze`, {
    method: 'POST',
  });
}

export function activateJobTarget(jobTargetId: string): Promise<UserProfile> {
  return request(`/api/job-targets/${jobTargetId}/activate`, {
    method: 'POST',
  });
}

export function clearActiveJobTarget(): Promise<UserProfile> {
  return request('/api/job-targets/clear-active', {
    method: 'POST',
  });
}

export function listJobTargetAnalysisRuns(
  jobTargetId: string,
): Promise<JobTargetAnalysisRun[]> {
  return request(`/api/job-targets/${jobTargetId}/analysis-runs`);
}

export function getJobTargetAnalysisRun(
  runId: string,
): Promise<JobTargetAnalysisRun> {
  return request(`/api/job-targets/analysis-runs/${runId}`);
}

export function updateProject(
  projectId: string,
  payload: Partial<ProjectProfile>,
): Promise<ProjectProfile> {
  return request(`/api/projects/${projectId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  });
}

export function createSession(payload: {
  mode: 'basics' | 'project';
  topic?: string;
  project_id?: string;
  job_target_id?: string;
  prompt_set_id?: string;
  ignore_active_job_target?: boolean;
  intensity: string;
  max_turns?: number;
}): Promise<TrainingSession> {
  return request('/api/sessions', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function createSessionStream(
  payload: {
    mode: 'basics' | 'project';
    topic?: string;
    project_id?: string;
    job_target_id?: string;
    prompt_set_id?: string;
    ignore_active_job_target?: boolean;
    intensity: string;
    max_turns?: number;
  },
  onEvent: (event: StreamEvent) => void,
): Promise<TrainingSession> {
  return requestStream(
    '/api/sessions/stream',
    {
      method: 'POST',
      body: JSON.stringify(payload),
    },
    onEvent,
  );
}

export function listSessions(params?: {
  page?: number;
  per_page?: number;
  mode?: string;
  topic?: string;
  status?: string;
}): Promise<PaginatedList<TrainingSessionSummary>> {
  const query = new URLSearchParams();
  if (params?.page) query.set('page', String(params.page));
  if (params?.per_page) query.set('per_page', String(params.per_page));
  if (params?.mode) query.set('mode', params.mode);
  if (params?.topic) query.set('topic', params.topic);
  if (params?.status) query.set('status', params.status);
  const qs = query.toString();
  return request(`/api/sessions${qs ? `?${qs}` : ''}`);
}

export function getSession(sessionId: string): Promise<TrainingSession> {
  return request(`/api/sessions/${sessionId}`);
}

export function listSessionEvaluationLogs(
  sessionId: string,
): Promise<EvaluationLogEntry[]> {
  return request(`/api/sessions/${sessionId}/evaluation-logs`);
}

export async function downloadSessionExport(
  sessionId: string,
  format: SessionExportFormat = SESSION_EXPORT_FORMAT,
): Promise<{ blob: Blob; filename: string }> {
  const response = await fetch(buildSessionExportPath(sessionId, format));

  if (!response.ok) {
    const payload = (await response
      .json()
      .catch(() => null)) as ApiErrorPayload | null;
    throw new ApiError(payload?.error?.message ?? '导出失败', {
      code: payload?.error?.code,
      status: response.status,
    });
  }

  return {
    blob: await response.blob(),
    filename: resolveDownloadFilename(
      sessionId,
      format,
      response.headers.get('Content-Disposition'),
    ),
  };
}

export async function downloadSessionBatchExport(
  sessionIds: string[],
  format: SessionExportFormat = SESSION_EXPORT_FORMAT,
): Promise<{ blob: Blob; filename: string }> {
  const response = await fetch(SESSION_BATCH_EXPORT_PATH, {
    method: 'POST',
    headers: {
      Accept: 'application/zip',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      session_ids: sessionIds,
      format,
    }),
  });

  if (!response.ok) {
    const payload = (await response
      .json()
      .catch(() => null)) as ApiErrorPayload | null;
    throw new ApiError(payload?.error?.message ?? '导出失败', {
      code: payload?.error?.code,
      status: response.status,
    });
  }

  return {
    blob: await response.blob(),
    filename: resolveBatchDownloadFilename(
      sessionIds.length,
      format,
      response.headers.get('Content-Disposition'),
    ),
  };
}

export function submitAnswer(
  sessionId: string,
  answer: string,
): Promise<TrainingSession> {
  return request(`/api/sessions/${sessionId}/answer`, {
    method: 'POST',
    body: JSON.stringify({ answer }),
  });
}

export function submitAnswerStream(
  sessionId: string,
  answer: string,
  onEvent: (event: StreamEvent) => void,
): Promise<TrainingSession> {
  return requestStream(
    `/api/sessions/${sessionId}/answer/stream`,
    {
      method: 'POST',
      body: JSON.stringify({ answer }),
    },
    onEvent,
  );
}

export function retrySessionReview(
  sessionId: string,
): Promise<TrainingSession> {
  return request(`/api/sessions/${sessionId}/retry-review`, {
    method: 'POST',
  });
}

export function getReview(reviewId: string): Promise<ReviewCard> {
  return request(`/api/reviews/${reviewId}`);
}

export function listWeaknesses(): Promise<WeaknessTag[]> {
  return request('/api/weaknesses');
}

export function getWeaknessTrends(): Promise<WeaknessTrend[]> {
  return request('/api/weaknesses/trends');
}

export function listDueReviews(): Promise<ReviewScheduleItem[]> {
  return request('/api/reviews/due');
}

export function completeDueReview(id: number): Promise<string> {
  return request(`/api/reviews/due/${id}/complete`, {
    method: 'POST',
  });
}
