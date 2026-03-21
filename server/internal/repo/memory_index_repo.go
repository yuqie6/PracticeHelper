package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"practicehelper/server/internal/domain"
)

type contextExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func (s *Store) UpsertMemoryIndexEntries(
	ctx context.Context,
	entries []domain.MemoryIndexEntry,
) error {
	return s.upsertMemoryIndexEntries(ctx, s.db, entries)
}

func (s *Store) upsertMemoryIndexEntries(
	ctx context.Context,
	exec contextExec,
	entries []domain.MemoryIndexEntry,
) error {
	for _, entry := range entries {
		if entry.RefTable == "" || entry.RefID == "" {
			continue
		}
		if entry.ID == "" {
			entry.ID = newID("memidx")
		}
		if entry.ScopeType == "" {
			entry.ScopeType = domain.MemoryScopeGlobal
		}
		now := nowUTC()
		if _, err := exec.ExecContext(ctx, `
			INSERT INTO memory_index (
				id, memory_type, scope_type, scope_id, topic, project_id, session_id, job_target_id,
				tags_json, entities_json, summary, salience, confidence, freshness,
				ref_table, ref_id, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(ref_table, ref_id) DO UPDATE SET
				memory_type = excluded.memory_type,
				scope_type = excluded.scope_type,
				scope_id = excluded.scope_id,
				topic = excluded.topic,
				project_id = excluded.project_id,
				session_id = excluded.session_id,
				job_target_id = excluded.job_target_id,
				tags_json = excluded.tags_json,
				entities_json = excluded.entities_json,
				summary = excluded.summary,
				salience = excluded.salience,
				confidence = excluded.confidence,
				freshness = excluded.freshness,
				updated_at = excluded.updated_at
		`,
			entry.ID,
			entry.MemoryType,
			normalizeMemoryScope(entry.ScopeType),
			entry.ScopeID,
			normalizeTopicLabel(entry.Topic),
			entry.ProjectID,
			entry.SessionID,
			entry.JobTargetID,
			mustJSON(entry.Tags),
			mustJSON(entry.Entities),
			entry.Summary,
			entry.Salience,
			entry.Confidence,
			entry.Freshness,
			entry.RefTable,
			entry.RefID,
			now,
			now,
		); err != nil {
			return fmt.Errorf("upsert memory index %s/%s: %w", entry.RefTable, entry.RefID, err)
		}
	}

	return nil
}

func (s *Store) ListMemoryIndexEntries(
	ctx context.Context,
	scopeType string,
	scopeID string,
	memoryType string,
	limit int,
) ([]domain.MemoryIndexEntry, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, memory_type, scope_type, scope_id, topic, project_id, session_id, job_target_id,
			tags_json, entities_json, summary, salience, confidence, freshness,
			ref_table, ref_id, created_at, updated_at
		FROM memory_index
		WHERE scope_type = ? AND (? = '' OR scope_id = ?) AND (? = '' OR memory_type = ?)
		ORDER BY salience DESC, updated_at DESC
		LIMIT ?
	`, normalizeMemoryScope(scopeType), scopeID, scopeID, memoryType, memoryType, limit)
	if err != nil {
		return nil, fmt.Errorf("list memory index entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.MemoryIndexEntry, 0)
	for rows.Next() {
		item, err := scanMemoryIndexEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory index entries: %w", err)
	}

	return items, nil
}

func (s *Store) SearchMemoryIndexEntries(
	ctx context.Context,
	memoryType string,
	topic string,
	projectID string,
	jobTargetID string,
	excludeSessionID string,
	limit int,
) ([]domain.MemoryIndexEntry, error) {
	if limit <= 0 {
		limit = 20
	}

	topic = normalizeTopicLabel(topic)
	projectID = strings.TrimSpace(projectID)
	jobTargetID = strings.TrimSpace(jobTargetID)
	excludeSessionID = strings.TrimSpace(excludeSessionID)

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, memory_type, scope_type, scope_id, topic, project_id, session_id, job_target_id,
			tags_json, entities_json, summary, salience, confidence, freshness,
			ref_table, ref_id, created_at, updated_at
		FROM memory_index
		WHERE memory_type = ? AND (? = '' OR session_id <> ?) AND (
			(? <> '' AND (project_id = ? OR (scope_type = 'project' AND scope_id = ?))) OR
			(? <> '' AND (job_target_id = ? OR (scope_type = 'job_target' AND scope_id = ?))) OR
			(? <> '' AND topic = ?) OR
			(scope_type = 'global' AND (? = '' OR topic = '' OR topic = ?)) OR
			(? = '' AND ? = '' AND ? = '' AND scope_type = 'global')
		)
		ORDER BY
			CASE
				WHEN ? <> '' AND (project_id = ? OR (scope_type = 'project' AND scope_id = ?)) THEN 0
				WHEN ? <> '' AND (job_target_id = ? OR (scope_type = 'job_target' AND scope_id = ?)) THEN 1
				WHEN ? <> '' AND topic = ? THEN 2
				WHEN scope_type = 'global' AND (? = '' OR topic = '' OR topic = ?) THEN 3
				ELSE 4
			END,
			salience DESC,
			confidence DESC,
			freshness DESC,
			updated_at DESC
		LIMIT ?
	`,
		memoryType,
		excludeSessionID, excludeSessionID,
		projectID, projectID, projectID,
		jobTargetID, jobTargetID, jobTargetID,
		topic, topic,
		topic, topic,
		topic, projectID, jobTargetID,
		projectID, projectID, projectID,
		jobTargetID, jobTargetID, jobTargetID,
		topic, topic,
		topic, topic,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search memory index entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.MemoryIndexEntry, 0, limit)
	for rows.Next() {
		item, err := scanMemoryIndexEntry(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory index search results: %w", err)
	}

	return items, nil
}
