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
