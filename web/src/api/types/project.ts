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
