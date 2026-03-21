package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
)

func (s *Store) GetMemoryIndexEntriesByRefs(
	ctx context.Context,
	refs []domain.MemoryRef,
) ([]domain.MemoryIndexEntry, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	clauses := make([]string, 0, len(refs))
	args := make([]any, 0, len(refs)*2)
	for _, ref := range refs {
		if ref.RefTable == "" || ref.RefID == "" {
			continue
		}
		clauses = append(clauses, "(ref_table = ? AND ref_id = ?)")
		args = append(args, ref.RefTable, ref.RefID)
	}
	if len(clauses) == 0 {
		return nil, nil
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, memory_type, scope_type, scope_id, topic, project_id, session_id, job_target_id,
			tags_json, entities_json, summary, salience, confidence, freshness,
			ref_table, ref_id, created_at, updated_at
		FROM memory_index
		WHERE %s
	`, strings.Join(clauses, " OR ")), args...)
	if err != nil {
		return nil, fmt.Errorf("get memory index entries by refs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	byKey := make(map[string]domain.MemoryIndexEntry, len(refs))
	for rows.Next() {
		entry, err := scanMemoryIndexEntry(rows)
		if err != nil {
			return nil, err
		}
		byKey[memoryRefKey(entry.RefTable, entry.RefID)] = *entry
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory index entries by refs: %w", err)
	}

	ordered := make([]domain.MemoryIndexEntry, 0, len(refs))
	for _, ref := range refs {
		entry, ok := byKey[memoryRefKey(ref.RefTable, ref.RefID)]
		if !ok {
			continue
		}
		ordered = append(ordered, entry)
	}
	return ordered, nil
}

func (s *Store) GetMemoryIndexEntriesByIDs(
	ctx context.Context,
	ids []string,
) ([]domain.MemoryIndexEntry, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, memory_type, scope_type, scope_id, topic, project_id, session_id, job_target_id,
			tags_json, entities_json, summary, salience, confidence, freshness,
			ref_table, ref_id, created_at, updated_at
		FROM memory_index
		WHERE id IN (%s)
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("get memory index entries by ids: %w", err)
	}
	defer func() { _ = rows.Close() }()

	byID := make(map[string]domain.MemoryIndexEntry, len(ids))
	for rows.Next() {
		entry, err := scanMemoryIndexEntry(rows)
		if err != nil {
			return nil, err
		}
		byID[entry.ID] = *entry
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory index entries by ids: %w", err)
	}

	ordered := make([]domain.MemoryIndexEntry, 0, len(ids))
	for _, id := range ids {
		entry, ok := byID[id]
		if !ok {
			continue
		}
		ordered = append(ordered, entry)
	}
	return ordered, nil
}

func (s *Store) GetMemoryEmbeddingRecordsByMemoryIndexIDs(
	ctx context.Context,
	memoryIndexIDs []string,
) ([]domain.MemoryEmbeddingRecord, error) {
	if len(memoryIndexIDs) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(memoryIndexIDs)), ",")
	args := make([]any, 0, len(memoryIndexIDs))
	for _, id := range memoryIndexIDs {
		args = append(args, id)
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, memory_index_id, memory_type, ref_table, ref_id,
			content_hash, model_name, vector_store_id, vector_dim, status,
			last_error, last_indexed_at, created_at, updated_at
		FROM memory_embedding_records
		WHERE memory_index_id IN (%s)
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("get memory embedding records: %w", err)
	}
	defer func() { _ = rows.Close() }()

	byID := make(map[string]domain.MemoryEmbeddingRecord, len(memoryIndexIDs))
	for rows.Next() {
		record, err := scanMemoryEmbeddingRecord(rows)
		if err != nil {
			return nil, err
		}
		byID[record.MemoryIndexID] = *record
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory embedding records: %w", err)
	}

	ordered := make([]domain.MemoryEmbeddingRecord, 0, len(memoryIndexIDs))
	for _, id := range memoryIndexIDs {
		record, ok := byID[id]
		if !ok {
			continue
		}
		ordered = append(ordered, record)
	}
	return ordered, nil
}

func (s *Store) UpsertMemoryEmbeddingRecords(
	ctx context.Context,
	records []domain.MemoryEmbeddingRecord,
) error {
	for _, record := range records {
		if record.MemoryIndexID == "" {
			continue
		}
		if record.ID == "" {
			record.ID = newID("memb")
		}
		if record.Status == "" {
			record.Status = domain.MemoryEmbeddingStatusPending
		}

		now := nowUTC()
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO memory_embedding_records (
				id, memory_index_id, memory_type, ref_table, ref_id,
				content_hash, model_name, vector_store_id, vector_dim, status,
				last_error, last_indexed_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(memory_index_id) DO UPDATE SET
				memory_type = excluded.memory_type,
				ref_table = excluded.ref_table,
				ref_id = excluded.ref_id,
				content_hash = excluded.content_hash,
				model_name = excluded.model_name,
				vector_store_id = excluded.vector_store_id,
				vector_dim = excluded.vector_dim,
				status = excluded.status,
				last_error = excluded.last_error,
				last_indexed_at = excluded.last_indexed_at,
				updated_at = excluded.updated_at
		`,
			record.ID,
			record.MemoryIndexID,
			record.MemoryType,
			record.RefTable,
			record.RefID,
			record.ContentHash,
			record.ModelName,
			record.VectorStoreID,
			record.VectorDim,
			record.Status,
			record.LastError,
			toNullableTimeString(record.LastIndexedAt),
			now,
			now,
		); err != nil {
			return fmt.Errorf("upsert memory embedding record %s: %w", record.MemoryIndexID, err)
		}
	}
	return nil
}

func (s *Store) EnqueueMemoryEmbeddingJobs(
	ctx context.Context,
	entries []domain.MemoryIndexEntry,
) error {
	for _, entry := range entries {
		if entry.ID == "" {
			continue
		}
		now := nowUTC()
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO memory_embedding_jobs (
				id, memory_index_id, memory_type, ref_table, ref_id, status,
				attempt_count, error_message, claim_token, claim_expires_at,
				created_at, updated_at, started_at, finished_at
			) VALUES (?, ?, ?, ?, ?, ?, 0, '', '', '', ?, ?, '', '')
			ON CONFLICT(memory_index_id) DO UPDATE SET
				memory_type = excluded.memory_type,
				ref_table = excluded.ref_table,
				ref_id = excluded.ref_id,
				status = CASE
					WHEN memory_embedding_jobs.status = ? THEN memory_embedding_jobs.status
					ELSE excluded.status
				END,
				error_message = CASE
					WHEN memory_embedding_jobs.status = ? THEN memory_embedding_jobs.error_message
					ELSE ''
				END,
				claim_token = CASE
					WHEN memory_embedding_jobs.status = ? THEN memory_embedding_jobs.claim_token
					ELSE ''
				END,
				claim_expires_at = CASE
					WHEN memory_embedding_jobs.status = ? THEN memory_embedding_jobs.claim_expires_at
					ELSE ''
				END,
				finished_at = CASE
					WHEN memory_embedding_jobs.status = ? THEN memory_embedding_jobs.finished_at
					ELSE ''
				END,
				updated_at = excluded.updated_at
		`,
			newID("mjob"),
			entry.ID,
			entry.MemoryType,
			entry.RefTable,
			entry.RefID,
			domain.MemoryEmbeddingJobStatusQueued,
			now,
			now,
			domain.MemoryEmbeddingJobStatusRunning,
			domain.MemoryEmbeddingJobStatusRunning,
			domain.MemoryEmbeddingJobStatusRunning,
			domain.MemoryEmbeddingJobStatusRunning,
			domain.MemoryEmbeddingJobStatusRunning,
		); err != nil {
			return fmt.Errorf("enqueue memory embedding job %s: %w", entry.ID, err)
		}
	}
	return nil
}

func (s *Store) ClaimNextMemoryEmbeddingJob(
	ctx context.Context,
	claimToken string,
	claimExpiresAt time.Time,
) (*domain.MemoryEmbeddingJob, error) {
	now := nowUTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin claim memory embedding job: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRowContext(ctx, `
		SELECT id, memory_index_id, memory_type, ref_table, ref_id, status,
			attempt_count, error_message, claim_token, claim_expires_at,
			created_at, updated_at, started_at, finished_at
		FROM memory_embedding_jobs
		WHERE status = ?
			OR (status = ? AND (claim_token = '' OR claim_expires_at = '' OR claim_expires_at <= ?))
		ORDER BY updated_at ASC
		LIMIT 1
	`, domain.MemoryEmbeddingJobStatusQueued, domain.MemoryEmbeddingJobStatusRunning, now)

	job, err := scanMemoryEmbeddingJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select memory embedding job: %w", err)
	}

	startedAt := nowUTC()
	result, err := tx.ExecContext(ctx, `
		UPDATE memory_embedding_jobs
		SET status = ?, attempt_count = attempt_count + 1, error_message = '',
			claim_token = ?, claim_expires_at = ?, started_at = ?, finished_at = '', updated_at = ?
		WHERE id = ?
	`, domain.MemoryEmbeddingJobStatusRunning, claimToken, claimExpiresAt.UTC().Format(time.RFC3339Nano), startedAt, startedAt, job.ID)
	if err != nil {
		return nil, fmt.Errorf("claim memory embedding job %s: %w", job.ID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("claim memory embedding job rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claimed memory embedding job: %w", err)
	}

	job.Status = domain.MemoryEmbeddingJobStatusRunning
	job.AttemptCount++
	job.ClaimToken = claimToken
	job.ClaimExpiresAt = &claimExpiresAt
	started := parseTime(startedAt)
	job.StartedAt = &started
	return s.GetMemoryEmbeddingJob(ctx, job.ID)
}

func (s *Store) GetMemoryEmbeddingJob(
	ctx context.Context,
	jobID string,
) (*domain.MemoryEmbeddingJob, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, memory_index_id, memory_type, ref_table, ref_id, status,
			attempt_count, error_message, claim_token, claim_expires_at,
			created_at, updated_at, started_at, finished_at
		FROM memory_embedding_jobs
		WHERE id = ?
	`, jobID)
	job, err := scanMemoryEmbeddingJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get memory embedding job %s: %w", jobID, err)
	}
	return job, nil
}

func (s *Store) CompleteMemoryEmbeddingJob(
	ctx context.Context,
	jobID string,
	claimToken string,
) error {
	if _, err := s.db.ExecContext(ctx, `
		DELETE FROM memory_embedding_jobs
		WHERE id = ? AND status = ? AND claim_token = ?
	`, jobID, domain.MemoryEmbeddingJobStatusRunning, claimToken); err != nil {
		return fmt.Errorf("complete memory embedding job %s: %w", jobID, err)
	}
	return nil
}

func (s *Store) FailMemoryEmbeddingJob(
	ctx context.Context,
	jobID string,
	claimToken string,
	errorMessage string,
	permanent bool,
) error {
	status := domain.MemoryEmbeddingJobStatusQueued
	finishedAt := ""
	if permanent {
		status = domain.MemoryEmbeddingJobStatusFailed
		finishedAt = nowUTC()
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE memory_embedding_jobs
		SET status = ?, error_message = ?, claim_token = '', claim_expires_at = '',
			finished_at = ?, updated_at = ?
		WHERE id = ? AND status = ? AND claim_token = ?
	`, status, errorMessage, finishedAt, nowUTC(), jobID, domain.MemoryEmbeddingJobStatusRunning, claimToken); err != nil {
		return fmt.Errorf("fail memory embedding job %s: %w", jobID, err)
	}
	return nil
}

func memoryRefKey(refTable string, refID string) string {
	return refTable + "::" + refID
}
