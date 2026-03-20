package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"practicehelper/server/internal/domain"
)

func (s *Store) CreateImportedProject(ctx context.Context, analysis *domain.AnalyzeRepoResponse) (*domain.ProjectProfile, error) {
	projectID := newID("proj")
	now := nowUTC()

	// 项目导入必须把 profile、原始 chunk 和 FTS 副本一起写成功或一起回滚，
	// 否则后续 project 页面能看到项目，但检索不到代码片段，状态会不一致。
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin create project: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO project_profiles (
			id, name, repo_url, default_branch, import_commit, summary, tech_stack_json, highlights_json, challenges_json, tradeoffs_json, ownership_points_json, followup_points_json, import_status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		projectID,
		analysis.Name,
		analysis.RepoURL,
		analysis.DefaultBranch,
		analysis.ImportCommit,
		analysis.Summary,
		mustJSON(analysis.TechStack),
		mustJSON(analysis.Highlights),
		mustJSON(analysis.Challenges),
		mustJSON(analysis.Tradeoffs),
		mustJSON(analysis.OwnershipPoints),
		mustJSON(analysis.FollowupPoints),
		"ready",
		now,
		now,
	); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: project_profiles.repo_url") {
			return nil, ErrAlreadyImported
		}
		return nil, fmt.Errorf("insert project: %w", err)
	}

	for _, chunk := range analysis.Chunks {
		chunkID := newID("chunk")
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO repo_chunks (id, project_id, file_path, file_type, content, importance, fts_key, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, chunkID, projectID, chunk.FilePath, chunk.FileType, chunk.Content, chunk.Importance, chunk.FTSKey, now); err != nil {
			return nil, fmt.Errorf("insert repo chunk: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO repo_chunks_fts (chunk_id, project_id, file_path, file_type, content)
			VALUES (?, ?, ?, ?, ?)
		`, chunkID, projectID, chunk.FilePath, chunk.FileType, chunk.Content); err != nil {
			return nil, fmt.Errorf("insert repo chunk fts: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create project: %w", err)
	}

	return s.GetProject(ctx, projectID)
}

func (s *Store) ListProjects(ctx context.Context) ([]domain.ProjectProfile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, repo_url, default_branch, import_commit, summary, tech_stack_json, highlights_json, challenges_json, tradeoffs_json, ownership_points_json, followup_points_json, import_status, created_at, updated_at FROM project_profiles ORDER BY updated_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer func() { _ = rows.Close() }()

	projects := make([]domain.ProjectProfile, 0)
	for rows.Next() {
		project, err := scanProjectProfile(rows)
		if err != nil {
			return nil, err
		}
		projects = append(projects, *project)
	}

	return projects, nil
}

func (s *Store) GetProject(ctx context.Context, projectID string) (*domain.ProjectProfile, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, repo_url, default_branch, import_commit, summary, tech_stack_json, highlights_json, challenges_json, tradeoffs_json, ownership_points_json, followup_points_json, import_status, created_at, updated_at FROM project_profiles WHERE id = ?`, projectID)
	project, err := scanProjectProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Store) GetProjectByRepoURL(ctx context.Context, repoURL string) (*domain.ProjectProfile, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, repo_url, default_branch, import_commit, summary, tech_stack_json, highlights_json, challenges_json, tradeoffs_json, ownership_points_json, followup_points_json, import_status, created_at, updated_at FROM project_profiles WHERE repo_url = ?`, repoURL)
	project, err := scanProjectProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return project, nil
}

func (s *Store) UpdateProject(ctx context.Context, projectID string, input domain.ProjectProfileInput) (*domain.ProjectProfile, error) {
	if _, err := s.db.ExecContext(ctx, `
		UPDATE project_profiles
		SET name = ?, summary = ?, tech_stack_json = ?, highlights_json = ?, challenges_json = ?, tradeoffs_json = ?, ownership_points_json = ?, followup_points_json = ?, updated_at = ?
		WHERE id = ?
	`,
		input.Name,
		input.Summary,
		mustJSON(input.TechStack),
		mustJSON(input.Highlights),
		mustJSON(input.Challenges),
		mustJSON(input.Tradeoffs),
		mustJSON(input.OwnershipPoints),
		mustJSON(input.FollowupPoints),
		nowUTC(),
		projectID,
	); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}

	return s.GetProject(ctx, projectID)
}

func (s *Store) SearchProjectChunks(ctx context.Context, projectID, query string, limit int) ([]domain.RepoChunk, error) {
	if limit <= 0 {
		limit = 5
	}

	// 这里服务的是“为提问/追问捞少量高相关片段”，不是通用语义搜索。
	// 没有可用检索词时回退到 importance 排序，至少保证 sidecar 能拿到项目骨架；
	// 有检索词时则走保守的 SQLite FTS token 匹配，优先稳定和低成本。
	terms := buildFTSQuery(query)
	var rows *sql.Rows
	var err error
	if terms == "" {
		rows, err = s.db.QueryContext(ctx, `
			SELECT id, project_id, file_path, file_type, content, importance, fts_key, created_at
			FROM repo_chunks
			WHERE project_id = ?
			ORDER BY importance DESC, file_path ASC
			LIMIT ?
		`, projectID, limit)
	} else {
		rows, err = s.db.QueryContext(ctx, `
			SELECT rc.id, rc.project_id, rc.file_path, rc.file_type, rc.content, rc.importance, rc.fts_key, rc.created_at
			FROM repo_chunks_fts f
			JOIN repo_chunks rc ON rc.id = f.chunk_id
			WHERE f.project_id = ? AND repo_chunks_fts MATCH ?
			ORDER BY rc.importance DESC
			LIMIT ?
		`, projectID, terms, limit)
	}
	if err != nil {
		return nil, fmt.Errorf("search project chunks: %w", err)
	}
	defer func() { _ = rows.Close() }()

	chunks := make([]domain.RepoChunk, 0)
	for rows.Next() {
		chunk, err := scanRepoChunk(rows)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, *chunk)
	}

	return chunks, nil
}
