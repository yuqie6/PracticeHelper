import type { JobTargetRef } from './job-target';

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
