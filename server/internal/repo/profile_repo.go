package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"practicehelper/server/internal/domain"
)

func (s *Store) GetUserProfile(ctx context.Context) (*domain.UserProfile, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, target_role, target_company_type, current_stage, application_deadline, tech_stacks_json, primary_projects_json, self_reported_weaknesses_json, active_job_target_id, created_at, updated_at FROM user_profile WHERE id = 1`)
	profile, err := scanUserProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if profile.ActiveJobTargetID != "" {
		target, err := s.getJobTargetRow(ctx, profile.ActiveJobTargetID)
		if err != nil {
			return nil, err
		}
		if target != nil {
			profile.ActiveJobTarget = &domain.JobTargetRef{
				ID:                   target.ID,
				Title:                target.Title,
				CompanyName:          target.CompanyName,
				LatestAnalysisStatus: target.LatestAnalysisStatus,
			}
		}
	}

	return profile, nil
}

func (s *Store) SaveUserProfile(ctx context.Context, input domain.UserProfileInput) (*domain.UserProfile, error) {
	now := nowUTC()
	deadline := ""
	if input.ApplicationDeadline != nil {
		deadline = normalizeDateString(*input.ApplicationDeadline)
	}
	activeJobTargetID, err := s.getStoredActiveJobTargetID(ctx)
	if err != nil {
		return nil, err
	}
	if input.ActiveJobTargetID != "" {
		activeJobTargetID = input.ActiveJobTargetID
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO user_profile (
			id, target_role, target_company_type, current_stage, application_deadline, tech_stacks_json, primary_projects_json, self_reported_weaknesses_json, active_job_target_id, created_at, updated_at
		) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			target_role = excluded.target_role,
			target_company_type = excluded.target_company_type,
			current_stage = excluded.current_stage,
			application_deadline = excluded.application_deadline,
			tech_stacks_json = excluded.tech_stacks_json,
			primary_projects_json = excluded.primary_projects_json,
			self_reported_weaknesses_json = excluded.self_reported_weaknesses_json,
			active_job_target_id = excluded.active_job_target_id,
			updated_at = excluded.updated_at
	`,
		input.TargetRole,
		input.TargetCompanyType,
		input.CurrentStage,
		deadline,
		mustJSON(input.TechStacks),
		mustJSON(input.PrimaryProjects),
		mustJSON(input.SelfReportedWeakness),
		activeJobTargetID,
		now,
		now,
	); err != nil {
		return nil, fmt.Errorf("save user profile: %w", err)
	}

	return s.GetUserProfile(ctx)
}

func (s *Store) SetActiveJobTarget(ctx context.Context, jobTargetID string) (*domain.UserProfile, error) {
	now := nowUTC()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO user_profile (
			id, target_role, target_company_type, current_stage, application_deadline,
			tech_stacks_json, primary_projects_json, self_reported_weaknesses_json,
			active_job_target_id, created_at, updated_at
		) VALUES (1, '', '', '', '', '[]', '[]', '[]', ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			active_job_target_id = excluded.active_job_target_id,
			updated_at = excluded.updated_at
	`, jobTargetID, now, now); err != nil {
		return nil, fmt.Errorf("set active job target: %w", err)
	}
	return s.GetUserProfile(ctx)
}

func (s *Store) ClearActiveJobTarget(ctx context.Context) (*domain.UserProfile, error) {
	now := nowUTC()
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO user_profile (
			id, target_role, target_company_type, current_stage, application_deadline,
			tech_stacks_json, primary_projects_json, self_reported_weaknesses_json,
			active_job_target_id, created_at, updated_at
		) VALUES (1, '', '', '', '', '[]', '[]', '[]', '', ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			active_job_target_id = '',
			updated_at = excluded.updated_at
	`, now, now); err != nil {
		return nil, fmt.Errorf("clear active job target: %w", err)
	}
	return s.GetUserProfile(ctx)
}

func (s *Store) getStoredActiveJobTargetID(ctx context.Context) (string, error) {
	var activeJobTargetID string
	err := s.db.QueryRowContext(ctx, `
		SELECT active_job_target_id
		FROM user_profile
		WHERE id = 1
	`).Scan(&activeJobTargetID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get stored active job target: %w", err)
	}
	return activeJobTargetID, nil
}
