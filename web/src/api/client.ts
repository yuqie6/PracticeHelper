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
  strengths: string[];
  gaps: string[];
  followup_question?: string;
  followup_expected_points?: string[];
}

export interface TrainingTurn {
  id: string;
  question: string;
  expected_points: string[];
  answer: string;
  evaluation?: TrainingEvaluation;
  followup_question?: string;
  followup_expected_points?: string[];
  followup_answer?: string;
  followup_evaluation?: TrainingEvaluation;
}

export interface TrainingSession {
  id: string;
  mode: 'basics' | 'project';
  topic?: string;
  project_id?: string;
  intensity: string;
  status: string;
  total_score: number;
  review_id?: string;
  turns: TrainingTurn[];
  project?: ProjectProfile;
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
}

export interface ReviewCard {
  id: string;
  session_id: string;
  overall: string;
  highlights: string[];
  gaps: string[];
  suggested_topics: string[];
  next_training_focus: string[];
  score_breakdown: Record<string, number>;
}

export interface Dashboard {
  profile: UserProfile | null;
  weaknesses: WeaknessTag[];
  recent_sessions: TrainingSessionSummary[];
  current_session?: TrainingSessionSummary | null;
  today_focus: string;
  recommended_track: string;
  days_until_deadline?: number;
}

export interface StreamEvent {
  type: 'phase' | 'context' | 'reasoning' | 'content' | 'result' | 'error';
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
    const payload = (await response.json().catch(() => null)) as ApiErrorPayload | null;
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
    const payload = (await response.json().catch(() => null)) as ApiErrorPayload | null;
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

export function saveProfile(payload: Partial<UserProfile>): Promise<UserProfile> {
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

export function updateProject(projectId: string, payload: Partial<ProjectProfile>): Promise<ProjectProfile> {
  return request(`/api/projects/${projectId}`, {
    method: 'PATCH',
    body: JSON.stringify(payload),
  });
}

export function createSession(payload: {
  mode: 'basics' | 'project';
  topic?: string;
  project_id?: string;
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
    intensity: string;
  },
  onEvent: (event: StreamEvent) => void,
): Promise<TrainingSession> {
  return requestStream('/api/sessions/stream', {
    method: 'POST',
    body: JSON.stringify(payload),
  }, onEvent);
}

export function getSession(sessionId: string): Promise<TrainingSession> {
  return request(`/api/sessions/${sessionId}`);
}

export function submitAnswer(sessionId: string, answer: string): Promise<TrainingSession> {
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
  return requestStream(`/api/sessions/${sessionId}/answer/stream`, {
    method: 'POST',
    body: JSON.stringify({ answer }),
  }, onEvent);
}

export function retrySessionReview(sessionId: string): Promise<TrainingSession> {
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
