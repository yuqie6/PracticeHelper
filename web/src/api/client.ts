export interface ApiEnvelope<T> {
  data: T;
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
  today_focus: string;
  recommended_track: string;
  days_until_deadline?: number;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  if (!response.ok) {
    const payload = (await response.json().catch(() => null)) as
      | { error?: { message?: string } }
      | null;
    throw new Error(payload?.error?.message ?? '请求失败');
  }

  const payload = (await response.json()) as ApiEnvelope<T>;
  return payload.data;
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

export function importProject(repoUrl: string): Promise<ProjectProfile> {
  return request('/api/projects/import', {
    method: 'POST',
    body: JSON.stringify({ repo_url: repoUrl }),
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

export function getSession(sessionId: string): Promise<TrainingSession> {
  return request(`/api/sessions/${sessionId}`);
}

export function submitAnswer(sessionId: string, answer: string): Promise<TrainingSession> {
  return request(`/api/sessions/${sessionId}/answer`, {
    method: 'POST',
    body: JSON.stringify({ answer }),
  });
}

export function getReview(reviewId: string): Promise<ReviewCard> {
  return request(`/api/reviews/${reviewId}`);
}

export function listWeaknesses(): Promise<WeaknessTag[]> {
  return request('/api/weaknesses');
}
