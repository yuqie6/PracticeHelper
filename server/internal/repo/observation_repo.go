package repo

import (
	"context"
	"database/sql"
	"fmt"

	"practicehelper/server/internal/domain"
)

const maxActiveObservations = 200

func (s *Store) CreateObservations(
	ctx context.Context,
	sessionID string,
	observations []domain.AgentObservation,
) error {
	if len(observations) == 0 {
		return nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create observations: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	indexEntries := make([]domain.MemoryIndexEntry, 0, len(observations))
	for _, observation := range observations {
		if observation.Content == "" || observation.Category == "" {
			continue
		}
		if observation.ID == "" {
			observation.ID = newID("obs")
		}
		if observation.SessionID == "" {
			observation.SessionID = sessionID
		}
		if observation.ScopeType == "" {
			observation.ScopeType = domain.MemoryScopeGlobal
		}
		if observation.Relevance <= 0 {
			observation.Relevance = 1.0
		}

		now := nowUTC()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO agent_observations (
				id, session_id, scope_type, scope_id, topic, category, content,
				tags_json, relevance, created_at, archived_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			observation.ID,
			observation.SessionID,
			normalizeMemoryScope(observation.ScopeType),
			observation.ScopeID,
			normalizeTopicLabel(observation.Topic),
			observation.Category,
			observation.Content,
			mustJSON(observation.Tags),
			observation.Relevance,
			now,
			"",
		); err != nil {
			return fmt.Errorf("insert observation %s: %w", observation.ID, err)
		}

		indexEntries = append(indexEntries, domain.MemoryIndexEntry{
			MemoryType: "observation",
			ScopeType:  observation.ScopeType,
			ScopeID:    observation.ScopeID,
			Topic:      observation.Topic,
			SessionID:  observation.SessionID,
			Summary:    observation.Content,
			Tags:       observation.Tags,
			Salience:   observation.Relevance,
			Confidence: 0.7,
			Freshness:  1.0,
			RefTable:   "agent_observations",
			RefID:      observation.ID,
		})
	}

	if err := s.archiveExcessObservationsTx(ctx, tx); err != nil {
		return err
	}
	if err := s.upsertMemoryIndexEntries(ctx, tx, indexEntries); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) ListRelevantObservations(
	ctx context.Context,
	sessionID string,
	projectID string,
	jobTargetID string,
	topic string,
	limit int,
) ([]domain.AgentObservation, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, session_id, scope_type, scope_id, topic, category, content, tags_json, relevance, created_at, archived_at
		FROM agent_observations
		WHERE archived_at = '' AND (
			(? <> '' AND session_id = ?) OR
			(? <> '' AND scope_type = 'project' AND scope_id = ?) OR
			(? <> '' AND scope_type = 'job_target' AND scope_id = ?) OR
			(? <> '' AND topic = ?) OR
			scope_type = 'global'
		)
		ORDER BY
			CASE
				WHEN ? <> '' AND session_id = ? THEN 0
				WHEN ? <> '' AND scope_type = 'project' AND scope_id = ? THEN 1
				WHEN ? <> '' AND scope_type = 'job_target' AND scope_id = ? THEN 2
				WHEN ? <> '' AND topic = ? THEN 3
				ELSE 4
			END,
			relevance DESC,
			created_at DESC
		LIMIT ?
	`,
		sessionID, sessionID,
		projectID, projectID,
		jobTargetID, jobTargetID,
		normalizeTopicLabel(topic), normalizeTopicLabel(topic),
		sessionID, sessionID,
		projectID, projectID,
		jobTargetID, jobTargetID,
		normalizeTopicLabel(topic), normalizeTopicLabel(topic),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list relevant observations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.AgentObservation, 0, limit)
	for rows.Next() {
		item, err := scanAgentObservation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate relevant observations: %w", err)
	}

	return items, nil
}

func (s *Store) archiveExcessObservationsTx(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT id
		FROM agent_observations
		WHERE archived_at = ''
		ORDER BY relevance ASC, created_at ASC
	`)
	if err != nil {
		return fmt.Errorf("query active observations for archive: %w", err)
	}
	defer func() { _ = rows.Close() }()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return fmt.Errorf("scan active observation id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate active observation ids: %w", err)
	}
	if len(ids) <= maxActiveObservations {
		return nil
	}

	archiveAt := nowUTC()
	for _, id := range ids[:len(ids)-maxActiveObservations] {
		if _, err := tx.ExecContext(ctx, `
			UPDATE agent_observations
			SET archived_at = ?
			WHERE id = ? AND archived_at = ''
		`, archiveAt, id); err != nil {
			return fmt.Errorf("archive observation %s: %w", id, err)
		}
	}

	return nil
}
