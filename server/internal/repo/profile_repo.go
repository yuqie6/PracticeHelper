package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"practicehelper/server/internal/domain"
)

func (s *Store) GetUserProfile(ctx context.Context) (*domain.UserProfile, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, target_role, target_company_type, current_stage, application_deadline, tech_stacks_json, primary_projects_json, self_reported_weaknesses_json, created_at, updated_at FROM user_profile WHERE id = 1`)
	profile, err := scanUserProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return profile, nil
}

func (s *Store) SaveUserProfile(ctx context.Context, input domain.UserProfileInput) (*domain.UserProfile, error) {
	now := nowUTC()
	deadline := ""
	if input.ApplicationDeadline != nil {
		deadline = normalizeDateString(*input.ApplicationDeadline)
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO user_profile (
			id, target_role, target_company_type, current_stage, application_deadline, tech_stacks_json, primary_projects_json, self_reported_weaknesses_json, created_at, updated_at
		) VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			target_role = excluded.target_role,
			target_company_type = excluded.target_company_type,
			current_stage = excluded.current_stage,
			application_deadline = excluded.application_deadline,
			tech_stacks_json = excluded.tech_stacks_json,
			primary_projects_json = excluded.primary_projects_json,
			self_reported_weaknesses_json = excluded.self_reported_weaknesses_json,
			updated_at = excluded.updated_at
	`,
		input.TargetRole,
		input.TargetCompanyType,
		input.CurrentStage,
		deadline,
		mustJSON(input.TechStacks),
		mustJSON(input.PrimaryProjects),
		mustJSON(input.SelfReportedWeakness),
		now,
		now,
	); err != nil {
		return nil, fmt.Errorf("save user profile: %w", err)
	}

	return s.GetUserProfile(ctx)
}
