import type { JobTargetRef } from './job-target';
import type { ProjectProfile } from './project';
import type { UserProfile } from './profile';
import type { PromptSetSummary } from './prompt';

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
  prompt_set_id?: string;
  intensity: string;
  status: string;
  max_turns: number;
  total_score: number;
  review_id?: string;
  turns: TrainingTurn[];
  project?: ProjectProfile;
  job_target?: JobTargetRef | null;
  prompt_set?: PromptSetSummary | null;
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
  prompt_set_id?: string;
  prompt_set?: PromptSetSummary | null;
}

export interface RuntimeTraceEntry {
  flow: string;
  phase: string;
  status: string;
  code?: string;
  message?: string;
  attempt?: number;
  tool_name?: string;
}

export interface RuntimeTrace {
  entries: RuntimeTraceEntry[];
}

export interface MemoryRetrievalHit {
  source: string;
  memory_index_id?: string;
  ref_table?: string;
  ref_id?: string;
  scope_type?: string;
  scope_id?: string;
  topic?: string;
  summary?: string;
  rule_score?: number;
  vector_score?: number;
  rerank_score?: number;
  final_score?: number;
  reason?: string;
}

export interface MemoryRetrievalTrace {
  memory_type: string;
  query?: string;
  strategy: string;
  candidate_count: number;
  selected_count: number;
  fallback_used?: boolean;
  fallback_reason?: string;
  hits: MemoryRetrievalHit[];
}

export interface RetrievalTrace {
  generated_at: string;
  topic?: string;
  project_id?: string;
  job_target_id?: string;
  observations?: MemoryRetrievalTrace | null;
  session_summaries?: MemoryRetrievalTrace | null;
}

export interface ReviewCard {
  id: string;
  session_id: string;
  job_target_id?: string;
  job_target_analysis_id?: string;
  prompt_set_id?: string;
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
  retrieval_trace?: RetrievalTrace | null;
  job_target?: JobTargetRef | null;
  prompt_set?: PromptSetSummary | null;
}

export interface EvaluationLogEntry {
  id: number;
  session_id: string;
  turn_id?: string;
  flow_name: string;
  model_name?: string;
  prompt_set_id?: string;
  prompt_hash?: string;
  raw_output?: string;
  runtime_trace?: RuntimeTrace | null;
  latency_ms: number;
  created_at: string;
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
    | 'trace'
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

export interface PaginatedList<T> {
  items: T[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
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

export interface ReviewScheduleItem {
  id: number;
  session_id: string;
  review_card_id?: string;
  weakness_tag_id?: string;
  weakness_kind?: string;
  weakness_label?: string;
  topic?: string;
  next_review_at: string;
  interval_days: number;
}
