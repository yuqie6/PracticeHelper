package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"practicehelper/server/internal/domain"
)

func (s *Store) CreateSession(ctx context.Context, session *domain.TrainingSession, turn *domain.TrainingTurn) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create session: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO training_sessions (id, mode, topic, project_id, intensity, status, total_score, started_at, ended_at, review_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		session.ID,
		session.Mode,
		session.Topic,
		session.ProjectID,
		session.Intensity,
		session.Status,
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
		SELECT id, mode, topic, project_id, intensity, status, total_score, started_at, ended_at, review_id, created_at, updated_at
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

	return session, nil
}

func (s *Store) ListTurns(ctx context.Context, sessionID string) ([]domain.TrainingTurn, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, turn_index, stage, question, expected_points_json, answer, evaluation_json, followup_question, followup_expected_points_json, followup_answer, followup_evaluation_json, weakness_hits_json, created_at, updated_at
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
		SET stage = ?, question = ?, expected_points_json = ?, answer = ?, evaluation_json = ?, followup_question = ?, followup_expected_points_json = ?, followup_answer = ?, followup_evaluation_json = ?, weakness_hits_json = ?, updated_at = ?
		WHERE id = ?
	`,
		turn.Stage,
		turn.Question,
		mustJSON(turn.ExpectedPoints),
		turn.Answer,
		mustJSON(turn.Evaluation),
		turn.FollowupQuestion,
		mustJSON(turn.FollowupExpectedPoint),
		turn.FollowupAnswer,
		mustJSON(turn.FollowupEvaluation),
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
		SET status = ?, total_score = ?, started_at = ?, ended_at = ?, review_id = ?, updated_at = ?
		WHERE id = ?
	`,
		session.Status,
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

func (s *Store) ListRecentSessions(ctx context.Context, limit int) ([]domain.TrainingSessionSummary, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT ts.id, ts.mode, ts.topic, COALESCE(pp.name, ''), ts.status, ts.total_score, ts.review_id, ts.updated_at
		FROM training_sessions ts
		LEFT JOIN project_profiles pp ON ts.project_id = pp.id
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
		var totalScore float64
		if err := rows.Scan(&id, &mode, &topic, &projectName, &status, &totalScore, &reviewID, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan recent session: %w", err)
		}

		items = append(items, domain.TrainingSessionSummary{
			ID:          id,
			Mode:        mode,
			Topic:       topic,
			ProjectName: projectName,
			Status:      status,
			TotalScore:  totalScore,
			ReviewID:    reviewID,
			UpdatedAt:   parseTime(updatedAt),
		})
	}

	return items, nil
}

func (s *Store) GetLatestResumableSession(ctx context.Context) (*domain.TrainingSessionSummary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT ts.id, ts.mode, ts.topic, COALESCE(pp.name, ''), ts.status, ts.total_score, ts.review_id, ts.updated_at
		FROM training_sessions ts
		LEFT JOIN project_profiles pp ON ts.project_id = pp.id
		WHERE ts.status IN (?, ?, ?, ?)
		ORDER BY ts.updated_at DESC
		LIMIT 1
	`, domain.StatusDraft, domain.StatusActive, domain.StatusWaitingAnswer, domain.StatusFollowup)

	var id, mode, topic, projectName, status, reviewID, updatedAt string
	var totalScore float64
	if err := row.Scan(&id, &mode, &topic, &projectName, &status, &totalScore, &reviewID, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get latest resumable session: %w", err)
	}

	return &domain.TrainingSessionSummary{
		ID:          id,
		Mode:        mode,
		Topic:       topic,
		ProjectName: projectName,
		Status:      status,
		TotalScore:  totalScore,
		ReviewID:    reviewID,
		UpdatedAt:   parseTime(updatedAt),
	}, nil
}

func insertTurn(ctx context.Context, tx *sql.Tx, turn *domain.TrainingTurn) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO training_turns (
			id, session_id, turn_index, stage, question, expected_points_json, answer, evaluation_json, followup_question, followup_expected_points_json, followup_answer, followup_evaluation_json, weakness_hits_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		turn.ID,
		turn.SessionID,
		turn.TurnIndex,
		turn.Stage,
		turn.Question,
		mustJSON(turn.ExpectedPoints),
		turn.Answer,
		mustJSON(turn.Evaluation),
		turn.FollowupQuestion,
		mustJSON(turn.FollowupExpectedPoint),
		turn.FollowupAnswer,
		mustJSON(turn.FollowupEvaluation),
		mustJSON(turn.WeaknessHits),
		nowUTC(),
		nowUTC(),
	); err != nil {
		return fmt.Errorf("insert turn: %w", err)
	}

	return nil
}
