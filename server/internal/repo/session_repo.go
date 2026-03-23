package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"practicehelper/server/internal/domain"
)

func (s *Store) CreateSession(ctx context.Context, session *domain.TrainingSession, turn *domain.TrainingTurn) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create session: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO training_sessions (id, mode, topic, project_id, job_target_id, job_target_analysis_id, prompt_set_id, prompt_set_label, prompt_set_status, prompt_overlay_json, prompt_overlay_hash, intensity, status, max_turns, total_score, started_at, ended_at, review_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		session.ID,
		session.Mode,
		session.Topic,
		session.ProjectID,
		session.JobTargetID,
		session.JobTargetAnalysisID,
		session.PromptSetID,
		promptSetLabel(session.PromptSet),
		promptSetStatus(session.PromptSet),
		mustJSON(session.PromptOverlay),
		session.PromptOverlayHash,
		session.Intensity,
		session.Status,
		session.MaxTurns,
		session.TotalScore,
		toNullableTimeString(session.StartedAt),
		toNullableTimeString(session.EndedAt),
		session.ReviewID,
		nowUTC(),
		nowUTC(),
	); err != nil {
		return fmt.Errorf("insert training session: %w", err)
	}

	if err := insertTurn(ctx, tx, turn); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) GetSession(ctx context.Context, sessionID string) (*domain.TrainingSession, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, mode, topic, project_id, job_target_id, job_target_analysis_id, prompt_set_id, prompt_set_label, prompt_set_status, prompt_overlay_json, prompt_overlay_hash, intensity, status, max_turns, total_score, started_at, ended_at, review_id, created_at, updated_at
		FROM training_sessions
		WHERE id = ?
	`, sessionID)
	session, err := scanTrainingSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	turns, err := s.ListTurns(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	session.Turns = turns

	if session.ProjectID != "" {
		project, err := s.GetProject(ctx, session.ProjectID)
		if err != nil {
			return nil, err
		}
		session.Project = project
	}
	if session.JobTargetID != "" {
		target, err := s.getJobTargetRow(ctx, session.JobTargetID)
		if err != nil {
			return nil, err
		}
		if target != nil {
			session.JobTarget = &domain.JobTargetRef{
				ID:          target.ID,
				Title:       target.Title,
				CompanyName: target.CompanyName,
			}
		}
	}

	return session, nil
}

func (s *Store) ListTurns(ctx context.Context, sessionID string) ([]domain.TrainingTurn, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, turn_index, stage, question, expected_points_json, answer, evaluation_json, weakness_hits_json, created_at, updated_at
		FROM training_turns
		WHERE session_id = ?
		ORDER BY turn_index ASC
	`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list turns: %w", err)
	}
	defer func() { _ = rows.Close() }()

	turns := make([]domain.TrainingTurn, 0)
	for rows.Next() {
		turn, err := scanTrainingTurn(rows)
		if err != nil {
			return nil, err
		}
		turns = append(turns, *turn)
	}

	return turns, nil
}

func (s *Store) SaveTurn(ctx context.Context, turn *domain.TrainingTurn) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE training_turns
		SET stage = ?, question = ?, expected_points_json = ?, answer = ?, evaluation_json = ?, weakness_hits_json = ?, updated_at = ?
		WHERE id = ?
	`,
		turn.Stage,
		turn.Question,
		mustJSON(turn.ExpectedPoints),
		turn.Answer,
		mustJSON(turn.Evaluation),
		mustJSON(turn.WeaknessHits),
		nowUTC(),
		turn.ID,
	)
	if err != nil {
		return fmt.Errorf("save turn: %w", err)
	}

	return nil
}

func (s *Store) SaveSession(ctx context.Context, session *domain.TrainingSession) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE training_sessions
		SET status = ?, max_turns = ?, total_score = ?, started_at = ?, ended_at = ?, review_id = ?, updated_at = ?
		WHERE id = ?
	`,
		session.Status,
		session.MaxTurns,
		session.TotalScore,
		toNullableTimeString(session.StartedAt),
		toNullableTimeString(session.EndedAt),
		session.ReviewID,
		nowUTC(),
		session.ID,
	)
	if err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	return nil
}

func (s *Store) TransitionSessionStatus(
	ctx context.Context,
	sessionID string,
	from []string,
	to string,
) (bool, error) {
	if len(from) == 0 {
		return false, errors.New("transition session status requires at least one source status")
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(from)), ",")
	query := fmt.Sprintf(`
		UPDATE training_sessions
		SET status = ?, updated_at = ?
		WHERE id = ? AND status IN (%s)
	`, placeholders)

	args := make([]any, 0, 3+len(from))
	args = append(args, to, nowUTC(), sessionID)
	for _, status := range from {
		args = append(args, status)
	}

	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("transition session status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("transition session status rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (s *Store) ListSessions(ctx context.Context, req domain.ListSessionsRequest) (*domain.PaginatedList[domain.TrainingSessionSummary], error) {
	where := "1=1"
	args := make([]any, 0)
	if req.Mode != "" {
		where += " AND ts.mode = ?"
		args = append(args, req.Mode)
	}
	if req.Topic != "" {
		where += " AND ts.topic = ?"
		args = append(args, req.Topic)
	}
	if req.Status != "" {
		where += " AND ts.status = ?"
		args = append(args, req.Status)
	}

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM training_sessions ts WHERE %s`, where)
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("count sessions: %w", err)
	}

	page := req.Page
	if page < 1 {
		page = 1
	}
	perPage := req.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	totalPages := (total + perPage - 1) / perPage
	offset := (page - 1) * perPage

	query := fmt.Sprintf(`
		SELECT ts.id, ts.mode, ts.topic, COALESCE(pp.name, ''), ts.status, ts.total_score, ts.review_id, ts.updated_at,
			COALESCE(jt.id, ''), COALESCE(jt.title, ''), COALESCE(jt.company_name, ''),
			COALESCE(ts.prompt_set_id, ''), COALESCE(ts.prompt_set_label, ''), COALESCE(ts.prompt_set_status, ''),
			COALESCE(ts.prompt_overlay_hash, '')
		FROM training_sessions ts
		LEFT JOIN project_profiles pp ON ts.project_id = pp.id
		LEFT JOIN job_targets jt ON ts.job_target_id = jt.id
		WHERE %s
		ORDER BY ts.updated_at DESC
		LIMIT ? OFFSET ?
	`, where)
	queryArgs := append(args, perPage, offset)

	rows, err := s.db.QueryContext(ctx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.TrainingSessionSummary, 0)
	for rows.Next() {
		item, err := scanSessionSummaryRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}

	return &domain.PaginatedList[domain.TrainingSessionSummary]{
		Items:      items,
		Total:      total,
		Page:       page,
		PerPage:    perPage,
		TotalPages: totalPages,
	}, nil
}

func scanSessionSummaryRow(scanner interface{ Scan(dest ...any) error }) (*domain.TrainingSessionSummary, error) {
	var id, mode, topic, projectName, status, reviewID, updatedAt string
	var jobTargetID, jobTargetTitle, jobTargetCompanyName string
	var promptSetID, promptSetLabel, promptSetStatus, promptOverlayHash string
	var totalScore float64
	if err := scanner.Scan(&id, &mode, &topic, &projectName, &status, &totalScore, &reviewID, &updatedAt, &jobTargetID, &jobTargetTitle, &jobTargetCompanyName, &promptSetID, &promptSetLabel, &promptSetStatus, &promptOverlayHash); err != nil {
		return nil, fmt.Errorf("scan session summary: %w", err)
	}

	item := &domain.TrainingSessionSummary{
		ID:                id,
		Mode:              mode,
		Topic:             topic,
		ProjectName:       projectName,
		Status:            status,
		TotalScore:        totalScore,
		ReviewID:          reviewID,
		UpdatedAt:         parseTime(updatedAt),
		PromptSetID:       promptSetID,
		PromptOverlayHash: promptOverlayHash,
		PromptSet:         parsePromptSetSummary(promptSetID, promptSetLabel, promptSetStatus),
	}
	if jobTargetID != "" {
		item.JobTarget = &domain.JobTargetRef{
			ID:          jobTargetID,
			Title:       jobTargetTitle,
			CompanyName: jobTargetCompanyName,
		}
	}
	return item, nil
}

func (s *Store) ListRecentSessions(ctx context.Context, limit int) ([]domain.TrainingSessionSummary, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT ts.id, ts.mode, ts.topic, COALESCE(pp.name, ''), ts.status, ts.total_score, ts.review_id, ts.updated_at,
			COALESCE(jt.id, ''), COALESCE(jt.title, ''), COALESCE(jt.company_name, ''),
			COALESCE(ts.prompt_set_id, ''), COALESCE(ts.prompt_set_label, ''), COALESCE(ts.prompt_set_status, ''),
			COALESCE(ts.prompt_overlay_hash, '')
		FROM training_sessions ts
		LEFT JOIN project_profiles pp ON ts.project_id = pp.id
		LEFT JOIN job_targets jt ON ts.job_target_id = jt.id
		ORDER BY ts.updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.TrainingSessionSummary, 0)
	for rows.Next() {
		var id, mode, topic, projectName, status, reviewID, updatedAt string
		var jobTargetID, jobTargetTitle, jobTargetCompanyName string
		var promptSetID, promptSetLabel, promptSetStatus, promptOverlayHash string
		var totalScore float64
		if err := rows.Scan(&id, &mode, &topic, &projectName, &status, &totalScore, &reviewID, &updatedAt, &jobTargetID, &jobTargetTitle, &jobTargetCompanyName, &promptSetID, &promptSetLabel, &promptSetStatus, &promptOverlayHash); err != nil {
			return nil, fmt.Errorf("scan recent session: %w", err)
		}

		item := domain.TrainingSessionSummary{
			ID:                id,
			Mode:              mode,
			Topic:             topic,
			ProjectName:       projectName,
			Status:            status,
			TotalScore:        totalScore,
			ReviewID:          reviewID,
			UpdatedAt:         parseTime(updatedAt),
			PromptSetID:       promptSetID,
			PromptOverlayHash: promptOverlayHash,
			PromptSet:         parsePromptSetSummary(promptSetID, promptSetLabel, promptSetStatus),
		}
		if jobTargetID != "" {
			item.JobTarget = &domain.JobTargetRef{
				ID:          jobTargetID,
				Title:       jobTargetTitle,
				CompanyName: jobTargetCompanyName,
			}
		}
		items = append(items, item)
	}

	return items, nil
}

func (s *Store) GetLatestResumableSession(ctx context.Context) (*domain.TrainingSessionSummary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT ts.id, ts.mode, ts.topic, COALESCE(pp.name, ''), ts.status, ts.total_score, ts.review_id, ts.updated_at,
			COALESCE(jt.id, ''), COALESCE(jt.title, ''), COALESCE(jt.company_name, ''),
			COALESCE(ts.prompt_set_id, ''), COALESCE(ts.prompt_set_label, ''), COALESCE(ts.prompt_set_status, ''),
			COALESCE(ts.prompt_overlay_hash, '')
		FROM training_sessions ts
		LEFT JOIN project_profiles pp ON ts.project_id = pp.id
		LEFT JOIN job_targets jt ON ts.job_target_id = jt.id
		WHERE ts.status IN (?, ?, ?)
		ORDER BY ts.updated_at DESC
		LIMIT 1
	`, domain.StatusDraft, domain.StatusActive, domain.StatusWaitingAnswer)

	var id, mode, topic, projectName, status, reviewID, updatedAt string
	var jobTargetID, jobTargetTitle, jobTargetCompanyName string
	var promptSetID, promptSetLabel, promptSetStatus, promptOverlayHash string
	var totalScore float64
	if err := row.Scan(&id, &mode, &topic, &projectName, &status, &totalScore, &reviewID, &updatedAt, &jobTargetID, &jobTargetTitle, &jobTargetCompanyName, &promptSetID, &promptSetLabel, &promptSetStatus, &promptOverlayHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get latest resumable session: %w", err)
	}

	item := &domain.TrainingSessionSummary{
		ID:                id,
		Mode:              mode,
		Topic:             topic,
		ProjectName:       projectName,
		Status:            status,
		TotalScore:        totalScore,
		ReviewID:          reviewID,
		UpdatedAt:         parseTime(updatedAt),
		PromptSetID:       promptSetID,
		PromptOverlayHash: promptOverlayHash,
		PromptSet:         parsePromptSetSummary(promptSetID, promptSetLabel, promptSetStatus),
	}
	if jobTargetID != "" {
		item.JobTarget = &domain.JobTargetRef{
			ID:          jobTargetID,
			Title:       jobTargetTitle,
			CompanyName: jobTargetCompanyName,
		}
	}

	return item, nil
}

func promptSetLabel(promptSet *domain.PromptSetSummary) string {
	if promptSet == nil {
		return ""
	}
	return promptSet.Label
}

func promptSetStatus(promptSet *domain.PromptSetSummary) string {
	if promptSet == nil {
		return ""
	}
	return promptSet.Status
}

func (s *Store) InsertTurn(ctx context.Context, turn *domain.TrainingTurn) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO training_turns (
			id, session_id, turn_index, stage, question, expected_points_json, answer, evaluation_json, weakness_hits_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		turn.ID,
		turn.SessionID,
		turn.TurnIndex,
		turn.Stage,
		turn.Question,
		mustJSON(turn.ExpectedPoints),
		turn.Answer,
		mustJSON(turn.Evaluation),
		mustJSON(turn.WeaknessHits),
		nowUTC(),
		nowUTC(),
	)
	if err != nil {
		return fmt.Errorf("insert turn: %w", err)
	}
	return nil
}

func insertTurn(ctx context.Context, tx *sql.Tx, turn *domain.TrainingTurn) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO training_turns (
			id, session_id, turn_index, stage, question, expected_points_json, answer, evaluation_json, weakness_hits_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		turn.ID,
		turn.SessionID,
		turn.TurnIndex,
		turn.Stage,
		turn.Question,
		mustJSON(turn.ExpectedPoints),
		turn.Answer,
		mustJSON(turn.Evaluation),
		mustJSON(turn.WeaknessHits),
		nowUTC(),
		nowUTC(),
	); err != nil {
		return fmt.Errorf("insert turn: %w", err)
	}

	return nil
}
