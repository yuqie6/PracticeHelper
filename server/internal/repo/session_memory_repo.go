package repo

import (
	"context"
	"database/sql"
	"fmt"

	"practicehelper/server/internal/domain"
)

func (s *Store) UpsertSessionMemorySummary(
	ctx context.Context,
	summary *domain.SessionMemorySummary,
) error {
	if summary == nil || summary.SessionID == "" {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin upsert session memory summary: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if summary.ID == "" {
		summary.ID = newID("smem")
	}
	now := nowUTC()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO session_memory_summaries (
			id, session_id, mode, topic, project_id, job_target_id, prompt_set_id, summary,
			strengths_json, gaps_json, misconceptions_json, growth_json,
			recommended_focus_json, salience, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			mode = excluded.mode,
			topic = excluded.topic,
			project_id = excluded.project_id,
			job_target_id = excluded.job_target_id,
			prompt_set_id = excluded.prompt_set_id,
			summary = excluded.summary,
			strengths_json = excluded.strengths_json,
			gaps_json = excluded.gaps_json,
			misconceptions_json = excluded.misconceptions_json,
			growth_json = excluded.growth_json,
			recommended_focus_json = excluded.recommended_focus_json,
			salience = excluded.salience,
			updated_at = excluded.updated_at
	`,
		summary.ID,
		summary.SessionID,
		summary.Mode,
		normalizeTopicLabel(summary.Topic),
		summary.ProjectID,
		summary.JobTargetID,
		summary.PromptSetID,
		summary.Summary,
		mustJSON(summary.Strengths),
		mustJSON(summary.Gaps),
		mustJSON(summary.Misconceptions),
		mustJSON(summary.GrowthSignals),
		mustJSON(summary.RecommendedFocus),
		summary.Salience,
		now,
		now,
	); err != nil {
		return fmt.Errorf("upsert session memory summary %s: %w", summary.SessionID, err)
	}

	stored, err := s.getSessionMemorySummaryTx(ctx, tx, summary.SessionID)
	if err != nil {
		return err
	}
	if stored != nil {
		*summary = *stored
		if err := s.upsertMemoryIndexEntries(ctx, tx, []domain.MemoryIndexEntry{{
			MemoryType:  "session_summary",
			ScopeType:   domain.MemoryScopeSession,
			ScopeID:     stored.SessionID,
			Topic:       stored.Topic,
			ProjectID:   stored.ProjectID,
			SessionID:   stored.SessionID,
			JobTargetID: stored.JobTargetID,
			Summary:     stored.Summary,
			Tags:        append([]string{}, stored.Strengths...),
			Entities:    append([]string{}, stored.RecommendedFocus...),
			Salience:    stored.Salience,
			Confidence:  0.75,
			Freshness:   1.0,
			RefTable:    "session_memory_summaries",
			RefID:       stored.ID,
		}}); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *Store) GetSessionMemorySummary(
	ctx context.Context,
	sessionID string,
) (*domain.SessionMemorySummary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, session_id, mode, topic, project_id, job_target_id, prompt_set_id, summary,
			strengths_json, gaps_json, misconceptions_json, growth_json,
			recommended_focus_json, salience, created_at, updated_at
		FROM session_memory_summaries
		WHERE session_id = ?
	`, sessionID)
	item, err := scanSessionMemorySummary(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session memory summary %s: %w", sessionID, err)
	}
	return item, nil
}

func (s *Store) ListRelevantSessionMemorySummaries(
	ctx context.Context,
	topic string,
	projectID string,
	jobTargetID string,
	excludeSessionID string,
	limit int,
) ([]domain.SessionMemorySummary, error) {
	if limit <= 0 {
		limit = 3
	}

	topic = normalizeTopicLabel(topic)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, mode, topic, project_id, job_target_id, prompt_set_id, summary,
			strengths_json, gaps_json, misconceptions_json, growth_json,
			recommended_focus_json, salience, created_at, updated_at
		FROM session_memory_summaries
		WHERE (? = '' OR session_id <> ?) AND (
			(? <> '' AND topic = ?) OR
			(? <> '' AND project_id = ?) OR
			(? <> '' AND job_target_id = ?) OR
			(? = '' AND ? = '' AND ? = '')
		)
		ORDER BY
			CASE
				WHEN ? <> '' AND topic = ? THEN 0
				WHEN ? <> '' AND project_id = ? THEN 1
				WHEN ? <> '' AND job_target_id = ? THEN 2
				ELSE 3
			END,
			salience DESC,
			updated_at DESC
		LIMIT ?
	`,
		excludeSessionID, excludeSessionID,
		topic, topic,
		projectID, projectID,
		jobTargetID, jobTargetID,
		topic, projectID, jobTargetID,
		topic, topic,
		projectID, projectID,
		jobTargetID, jobTargetID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list relevant session summaries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.SessionMemorySummary, 0, limit)
	for rows.Next() {
		item, err := scanSessionMemorySummary(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate relevant session summaries: %w", err)
	}

	return items, nil
}

func (s *Store) getSessionMemorySummaryTx(
	ctx context.Context,
	tx *sql.Tx,
	sessionID string,
) (*domain.SessionMemorySummary, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, session_id, mode, topic, project_id, job_target_id, prompt_set_id, summary,
			strengths_json, gaps_json, misconceptions_json, growth_json,
			recommended_focus_json, salience, created_at, updated_at
		FROM session_memory_summaries
		WHERE session_id = ?
	`, sessionID)
	item, err := scanSessionMemorySummary(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get session memory summary in tx %s: %w", sessionID, err)
	}
	return item, nil
}
