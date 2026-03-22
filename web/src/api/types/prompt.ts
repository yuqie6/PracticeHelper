export interface PromptSetSummary {
  id: string;
  label: string;
  description?: string;
  status: string;
  is_default?: boolean;
}

export interface PromptExperimentMetrics {
  prompt_set: PromptSetSummary;
  session_count: number;
  completed_count: number;
  avg_total_score: number;
  avg_generate_question_latency_ms: number;
  avg_evaluate_answer_latency_ms: number;
  avg_generate_review_latency_ms: number;
}

export interface PromptExperimentSample {
  session_id: string;
  review_id?: string;
  mode: string;
  topic?: string;
  status: string;
  total_score: number;
  updated_at: string;
  prompt_set: PromptSetSummary;
}

export interface PromptExperimentReport {
  left: PromptExperimentMetrics;
  right: PromptExperimentMetrics;
  recent_samples: PromptExperimentSample[];
  applied_filters: {
    left: string;
    right: string;
    mode?: string;
    topic?: string;
    limit: number;
  };
}
