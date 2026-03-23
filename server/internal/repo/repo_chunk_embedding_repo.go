package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"practicehelper/server/internal/domain"
)

func (s *Store) GetRepoChunksByIDs(
	ctx context.Context,
	ids []string,
) ([]domain.RepoChunk, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, project_id, file_path, file_type, content, importance, fts_key, created_at
		FROM repo_chunks
		WHERE id IN (%s)
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("get repo chunks by ids: %w", err)
	}
	defer func() { _ = rows.Close() }()

	byID := make(map[string]domain.RepoChunk, len(ids))
	for rows.Next() {
		chunk, err := scanRepoChunk(rows)
		if err != nil {
			return nil, err
		}
		byID[chunk.ID] = *chunk
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate repo chunks by ids: %w", err)
	}

	ordered := make([]domain.RepoChunk, 0, len(ids))
	for _, id := range ids {
		chunk, ok := byID[id]
		if !ok {
			continue
		}
		ordered = append(ordered, chunk)
	}
	return ordered, nil
}

func (s *Store) ListRepoChunksByProject(
	ctx context.Context,
	projectID string,
) ([]domain.RepoChunk, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, file_path, file_type, content, importance, fts_key, created_at
		FROM repo_chunks
		WHERE project_id = ?
		ORDER BY importance DESC, file_path ASC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("list repo chunks by project: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return collectRepoChunks(rows, "list repo chunks by project")
}

func (s *Store) ListAllRepoChunks(ctx context.Context) ([]domain.RepoChunk, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, project_id, file_path, file_type, content, importance, fts_key, created_at
		FROM repo_chunks
		ORDER BY created_at ASC, file_path ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list all repo chunks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	return collectRepoChunks(rows, "list all repo chunks")
}

func collectRepoChunks(
	rows interface {
		Next() bool
		Scan(dest ...any) error
		Err() error
	},
	operation string,
) ([]domain.RepoChunk, error) {
	chunks := make([]domain.RepoChunk, 0)
	for rows.Next() {
		chunk, err := scanRepoChunk(rows)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, *chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s: %w", operation, err)
	}
	return chunks, nil
}

func (s *Store) GetRepoChunkEmbeddingRecordsByChunkIDs(
	ctx context.Context,
	chunkIDs []string,
) ([]domain.RepoChunkEmbeddingRecord, error) {
	if len(chunkIDs) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(chunkIDs)), ",")
	args := make([]any, 0, len(chunkIDs))
	for _, id := range chunkIDs {
		args = append(args, id)
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, repo_chunk_id, project_id, content_hash, model_name, vector_store_id,
			vector_dim, status, last_error, last_indexed_at, created_at, updated_at
		FROM repo_chunk_embedding_records
		WHERE repo_chunk_id IN (%s)
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("get repo chunk embedding records: %w", err)
	}
	defer func() { _ = rows.Close() }()

	byID := make(map[string]domain.RepoChunkEmbeddingRecord, len(chunkIDs))
	for rows.Next() {
		record, err := scanRepoChunkEmbeddingRecord(rows)
		if err != nil {
			return nil, err
		}
		byID[record.RepoChunkID] = *record
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate repo chunk embedding records: %w", err)
	}

	ordered := make([]domain.RepoChunkEmbeddingRecord, 0, len(chunkIDs))
	for _, id := range chunkIDs {
		record, ok := byID[id]
		if !ok {
			continue
		}
		ordered = append(ordered, record)
	}
	return ordered, nil
}

func (s *Store) UpsertRepoChunkEmbeddingRecords(
	ctx context.Context,
	records []domain.RepoChunkEmbeddingRecord,
) error {
	for _, record := range records {
		if record.RepoChunkID == "" {
			continue
		}
		if record.ID == "" {
			record.ID = newID("rce")
		}
		if record.Status == "" {
			record.Status = domain.RepoChunkEmbeddingStatusPending
		}

		now := nowUTC()
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO repo_chunk_embedding_records (
				id, repo_chunk_id, project_id, content_hash, model_name,
				vector_store_id, vector_dim, status, last_error,
				last_indexed_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(repo_chunk_id) DO UPDATE SET
				project_id = excluded.project_id,
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
			record.RepoChunkID,
			record.ProjectID,
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
			return fmt.Errorf("upsert repo chunk embedding record %s: %w", record.RepoChunkID, err)
		}
	}
	return nil
}

func (s *Store) EnqueueRepoChunkEmbeddingJobs(
	ctx context.Context,
	chunks []domain.RepoChunk,
) error {
	for _, chunk := range chunks {
		if chunk.ID == "" {
			continue
		}
		now := nowUTC()
		if _, err := s.db.ExecContext(ctx, `
			INSERT INTO repo_chunk_embedding_jobs (
				id, repo_chunk_id, project_id, status, attempt_count, error_message,
				claim_token, claim_expires_at, created_at, updated_at, started_at, finished_at
			) VALUES (?, ?, ?, ?, 0, '', '', '', ?, ?, '', '')
			ON CONFLICT(repo_chunk_id) DO UPDATE SET
				project_id = excluded.project_id,
				status = CASE
					WHEN repo_chunk_embedding_jobs.status = ? THEN repo_chunk_embedding_jobs.status
					ELSE excluded.status
				END,
				error_message = CASE
					WHEN repo_chunk_embedding_jobs.status = ? THEN repo_chunk_embedding_jobs.error_message
					ELSE ''
				END,
				claim_token = CASE
					WHEN repo_chunk_embedding_jobs.status = ? THEN repo_chunk_embedding_jobs.claim_token
					ELSE ''
				END,
				claim_expires_at = CASE
					WHEN repo_chunk_embedding_jobs.status = ? THEN repo_chunk_embedding_jobs.claim_expires_at
					ELSE ''
				END,
				finished_at = CASE
					WHEN repo_chunk_embedding_jobs.status = ? THEN repo_chunk_embedding_jobs.finished_at
					ELSE ''
				END,
				updated_at = excluded.updated_at
		`,
			newID("rcj"),
			chunk.ID,
			chunk.ProjectID,
			domain.RepoChunkEmbeddingJobStatusQueued,
			now,
			now,
			domain.RepoChunkEmbeddingJobStatusRunning,
			domain.RepoChunkEmbeddingJobStatusRunning,
			domain.RepoChunkEmbeddingJobStatusRunning,
			domain.RepoChunkEmbeddingJobStatusRunning,
			domain.RepoChunkEmbeddingJobStatusRunning,
		); err != nil {
			return fmt.Errorf("enqueue repo chunk embedding job %s: %w", chunk.ID, err)
		}
	}
	return nil
}

func (s *Store) ClaimNextRepoChunkEmbeddingJob(
	ctx context.Context,
	claimToken string,
	claimExpiresAt time.Time,
) (*domain.RepoChunkEmbeddingJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin claim repo chunk embedding job: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := nowUTC()
	row := tx.QueryRowContext(ctx, `
		SELECT id, repo_chunk_id, project_id, status, attempt_count, error_message,
			claim_token, claim_expires_at, created_at, updated_at, started_at, finished_at
		FROM repo_chunk_embedding_jobs
		WHERE status = ?
			OR (status = ? AND (claim_token = '' OR claim_expires_at = '' OR claim_expires_at <= ?))
		ORDER BY updated_at ASC
		LIMIT 1
	`,
		domain.RepoChunkEmbeddingJobStatusQueued,
		domain.RepoChunkEmbeddingJobStatusRunning,
		now,
	)
	job, err := scanRepoChunkEmbeddingJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("select repo chunk embedding job: %w", err)
	}

	startedAt := nowUTC()
	result, err := tx.ExecContext(ctx, `
		UPDATE repo_chunk_embedding_jobs
		SET status = ?, attempt_count = attempt_count + 1, error_message = '',
			claim_token = ?, claim_expires_at = ?, started_at = ?, updated_at = ?
		WHERE id = ? AND (claim_expires_at = '' OR claim_expires_at <= ?)
	`,
		domain.RepoChunkEmbeddingJobStatusRunning,
		claimToken,
		claimExpiresAt.UTC().Format(time.RFC3339Nano),
		startedAt,
		nowUTC(),
		job.ID,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("claim repo chunk embedding job %s: %w", job.ID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("claim repo chunk embedding job rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claimed repo chunk embedding job: %w", err)
	}

	job.Status = domain.RepoChunkEmbeddingJobStatusRunning
	job.AttemptCount++
	job.ErrorMessage = ""
	job.ClaimToken = claimToken
	job.ClaimExpiresAt = &claimExpiresAt
	startedTime := parseTime(startedAt)
	job.StartedAt = &startedTime
	return job, nil
}

func (s *Store) CompleteRepoChunkEmbeddingJob(
	ctx context.Context,
	jobID string,
	claimToken string,
) error {
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM repo_chunk_embedding_jobs
		WHERE id = ? AND claim_token = ?
	`, jobID, claimToken)
	if err != nil {
		return fmt.Errorf("complete repo chunk embedding job %s: %w", jobID, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("complete repo chunk embedding job %s rows affected: %w", jobID, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("complete repo chunk embedding job %s: claim lost", jobID)
	}
	return nil
}

func (s *Store) FailRepoChunkEmbeddingJob(
	ctx context.Context,
	jobID string,
	claimToken string,
	errorMessage string,
	permanent bool,
) error {
	status := domain.RepoChunkEmbeddingJobStatusFailed
	if !permanent {
		status = domain.RepoChunkEmbeddingJobStatusQueued
	}

	if _, err := s.db.ExecContext(ctx, `
		UPDATE repo_chunk_embedding_jobs
		SET status = ?, error_message = ?, claim_token = '', claim_expires_at = '',
			updated_at = ?, finished_at = ?
		WHERE id = ? AND claim_token = ?
	`,
		status,
		errorMessage,
		nowUTC(),
		nowUTC(),
		jobID,
		claimToken,
	); err != nil {
		return fmt.Errorf("fail repo chunk embedding job %s: %w", jobID, err)
	}
	return nil
}
