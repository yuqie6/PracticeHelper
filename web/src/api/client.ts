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

export interface UserProfile {
  id: number;
  target_role: string;
  target_company_type: string;
  current_stage: string;
  application_deadline?: string | null;
  tech_stacks: string[];
  primary_projects: string[];
  self_reported_weaknesses: string[];
  active_job_target_id?: string;
  active_job_target?: JobTargetRef | null;
}

export interface ProjectProfile {
  id: string;
  name: string;
  repo_url: string;
  default_branch: string;
  import_commit: string;
  summary: string;
  tech_stack: string[];
  highlights: string[];
  challenges: string[];
  tradeoffs: string[];
  ownership_points: string[];
  followup_points: string[];
  import_status: string;
}

export interface JobTargetRef {
  id: string;
  title: string;
  company_name?: string;
  latest_analysis_status?:
    | 'idle'
    | 'running'
    | 'succeeded'
    | 'failed'
    | 'stale';
}

export interface JobTargetAnalysisRun {
  id: string;
  job_target_id: string;
  source_text_snapshot: string;
  status: 'running' | 'succeeded' | 'failed';
  error_message?: string;
  summary?: string;
  must_have_skills: string[];
  bonus_skills: string[];
  responsibilities: string[];
  evaluation_focus: string[];
  created_at: string;
  finished_at?: string;
}

export interface JobTarget {
  id: string;
  title: string;
  company_name?: string;
  source_text: string;
  latest_analysis_id?: string;
  latest_analysis_status: 'idle' | 'running' | 'succeeded' | 'failed' | 'stale';
  last_used_at?: string;
  created_at: string;
  updated_at: string;
  latest_successful_analysis?: JobTargetAnalysisRun | null;
}

export interface ProjectImportJob {
  id: string;
  repo_url: string;
  status: 'queued' | 'running' | 'completed' | 'failed';
  stage:
    | 'queued'
    | 'analyzing_repository'
    | 'persisting_project'
    | 'completed'
    | 'failed';
  message: string;
  error_message?: string;
  project_id?: string;
  project_name?: string;
  created_at: string;
  updated_at: string;
  started_at?: string;
  finished_at?: string;
}

export interface WeaknessTag {
  id: string;
  kind: string;
  label: string;
  severity: number;
  frequency: number;
  last_seen_at: string;
  evidence_session_id: string;
}

export interface TrainingEvaluation {
  score: number;
  score_breakdown: Record<string, number>;
  headline?: string;
  strengths: string[];
  gaps: string[];
  suggestion?: string;
  followup_intent?: string;
  followup_question?: string;
  followup_expected_points?: string[];
}

export interface TrainingTurn {
  id: string;
  turn_index: number;
  question: string;
  expected_points: string[];
  answer: string;
  evaluation?: TrainingEvaluation;
}

export interface TrainingSession {
  id: string;
  mode: 'basics' | 'project';
  topic?: string;
  project_id?: string;
  job_target_id?: string;
  job_target_analysis_id?: string;
  intensity: string;
  status: string;
  max_turns: number;
  total_score: number;
  review_id?: string;
  turns: TrainingTurn[];
  project?: ProjectProfile;
  job_target?: JobTargetRef | null;
}

export interface TrainingSessionSummary {
  id: string;
  mode: string;
  topic?: string;
  project_name?: string;
  status: string;
  total_score: number;
  review_id?: string;
  updated_at: string;
  job_target?: JobTargetRef | null;
}

export interface ReviewCard {
  id: string;
  session_id: string;
  job_target_id?: string;
  job_target_analysis_id?: string;
  overall: string;
  top_fix?: string;
  top_fix_reason?: string;
  highlights: string[];
  gaps: string[];
  suggested_topics: string[];
  next_training_focus: string[];
  recommended_next?: {
    mode: 'basics' | 'project';
    topic?: string;
    project_id?: string;
    reason?: string;
  } | null;
  score_breakdown: Record<string, number>;
  job_target?: JobTargetRef | null;
}

export interface Dashboard {
  profile: UserProfile | null;
  weaknesses: WeaknessTag[];
  recent_sessions: TrainingSessionSummary[];
  current_session?: TrainingSessionSummary | null;
  today_focus: string;
  recommended_track: string;
  active_job_target?: JobTargetRef | null;
  recommendation_scope: 'generic' | 'job_target';
  days_until_deadline?: number;
}

export interface StreamEvent {
  type:
    | 'phase'
    | 'context'
    | 'reasoning'
    | 'content'
    | 'status'
    | 'result'
    | 'error';
  code?: string;
  phase?: string;
  name?: string;
  text?: string;
  message?: string;
  data?: unknown;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  // 这里封装的是当前 PracticeHelper API 的统一 JSON 契约：
  // 成功响应默认是 { data: T }，失败响应尽量从 { error.message } 提取可展示文案。
  // 如果后续出现 204、文件下载或非 JSON 接口，应该新增专用请求函数而不是继续复用这里。
  const response = await fetch(path, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
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
}

async function requestStream<T>(
  path: string,
  init: RequestInit,
  onEvent: (event: StreamEvent) => void,
): Promise<T> {
  const response = await fetch(path, {
    headers: {
      'Content-Type': 'application/json',
      ...(init.headers ?? {}),
    },
    ...init,
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
  ignore_active_job_target?: boolean;
  intensity: string;
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
    ignore_active_job_target?: boolean;
    intensity: string;
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

export interface PaginatedList<T> {
  items: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
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

export interface WeaknessTrendPoint {
  session_id: string;
  severity: number;
  created_at: string;
}

export interface WeaknessTrend {
  id: string;
  kind: string;
  label: string;
  points: WeaknessTrendPoint[];
}

export function getWeaknessTrends(): Promise<WeaknessTrend[]> {
  return request('/api/weaknesses/trends');
}

export interface ReviewScheduleItem {
  id: number;
  session_id: string;
  review_card_id?: string;
  topic?: string;
  next_review_at: string;
  interval_days: number;
}

export function listDueReviews(): Promise<ReviewScheduleItem[]> {
  return request('/api/reviews/due');
}

export function completeDueReview(
  id: number,
  score: number,
): Promise<string> {
  return request(`/api/reviews/due/${id}/complete`, {
    method: 'POST',
    body: JSON.stringify({ score }),
  });
}
