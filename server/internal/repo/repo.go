package repo

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"practicehelper/server/internal/domain"
)

type Store struct {
	db *sql.DB
}

var ErrAlreadyImported = errors.New("project already imported")

func Open(databasePath string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(databasePath), 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_busy_timeout=5000&_foreign_keys=on", databasePath)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	store := &Store{db: db}
	if err := store.migrate(); err != nil {
		return nil, err
	}

	if err := store.seedQuestionTemplates(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS user_profile (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			target_role TEXT NOT NULL,
			target_company_type TEXT NOT NULL,
			current_stage TEXT NOT NULL,
			application_deadline TEXT NOT NULL DEFAULT '',
			tech_stacks_json TEXT NOT NULL,
			primary_projects_json TEXT NOT NULL,
			self_reported_weaknesses_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS project_profiles (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			repo_url TEXT NOT NULL UNIQUE,
			default_branch TEXT NOT NULL,
			import_commit TEXT NOT NULL,
			summary TEXT NOT NULL,
			tech_stack_json TEXT NOT NULL,
			highlights_json TEXT NOT NULL,
			challenges_json TEXT NOT NULL,
			tradeoffs_json TEXT NOT NULL,
			ownership_points_json TEXT NOT NULL,
			followup_points_json TEXT NOT NULL,
			import_status TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS repo_chunks (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL REFERENCES project_profiles(id) ON DELETE CASCADE,
			file_path TEXT NOT NULL,
			file_type TEXT NOT NULL,
			content TEXT NOT NULL,
			importance REAL NOT NULL,
			fts_key TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS repo_chunks_fts USING fts5(
			chunk_id UNINDEXED,
			project_id UNINDEXED,
			file_path,
			file_type,
			content
		);`,
		`CREATE TABLE IF NOT EXISTS question_templates (
			id TEXT PRIMARY KEY,
			mode TEXT NOT NULL,
			topic TEXT NOT NULL,
			prompt TEXT NOT NULL,
			focus_points_json TEXT NOT NULL,
			bad_answers_json TEXT NOT NULL,
			followup_templates_json TEXT NOT NULL,
			score_weights_json TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS training_sessions (
			id TEXT PRIMARY KEY,
			mode TEXT NOT NULL,
			topic TEXT NOT NULL DEFAULT '',
			project_id TEXT NOT NULL DEFAULT '',
			intensity TEXT NOT NULL,
			status TEXT NOT NULL,
			total_score REAL NOT NULL DEFAULT 0,
			started_at TEXT NOT NULL DEFAULT '',
			ended_at TEXT NOT NULL DEFAULT '',
			review_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS training_turns (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES training_sessions(id) ON DELETE CASCADE,
			turn_index INTEGER NOT NULL,
			stage TEXT NOT NULL,
			question TEXT NOT NULL,
			expected_points_json TEXT NOT NULL,
			answer TEXT NOT NULL DEFAULT '',
			evaluation_json TEXT NOT NULL DEFAULT '{}',
			followup_question TEXT NOT NULL DEFAULT '',
			followup_expected_points_json TEXT NOT NULL DEFAULT '[]',
			followup_answer TEXT NOT NULL DEFAULT '',
			followup_evaluation_json TEXT NOT NULL DEFAULT '{}',
			weakness_hits_json TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS review_cards (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL UNIQUE REFERENCES training_sessions(id) ON DELETE CASCADE,
			overall TEXT NOT NULL,
			highlights_json TEXT NOT NULL,
			gaps_json TEXT NOT NULL,
			suggested_topics_json TEXT NOT NULL,
			next_training_focus_json TEXT NOT NULL,
			score_breakdown_json TEXT NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS weakness_tags (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			label TEXT NOT NULL,
			severity REAL NOT NULL,
			frequency INTEGER NOT NULL,
			last_seen_at TEXT NOT NULL,
			evidence_session_id TEXT NOT NULL,
			UNIQUE(kind, label)
		);`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("run migration: %w", err)
		}
	}

	return nil
}

func (s *Store) seedQuestionTemplates() error {
	var count int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM question_templates`).Scan(&count); err != nil {
		return fmt.Errorf("count templates: %w", err)
	}
	if count > 0 {
		return nil
	}

	templates := []domain.QuestionTemplate{
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "go",
			Prompt:            "Go 的 goroutine 为什么比系统线程更轻量？",
			FocusPoints:       []string{"GMP 调度", "栈扩缩容", "协作式与抢占式调度", "上下文切换成本"},
			BadAnswers:        []string{"只说 goroutine 很轻，不解释为什么", "只背概念，不提调度器和栈"},
			FollowupTemplates: []string{"如果 goroutine 很多，调度器怎么避免完全失控？", "goroutine 泄漏在线上怎么排查？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "redis",
			Prompt:            "Redis 为什么快？",
			FocusPoints:       []string{"内存访问", "事件循环", "高效数据结构", "网络模型", "避免大 key"},
			BadAnswers:        []string{"只说因为在内存里", "只说单线程快，不说 IO 模型和数据结构"},
			FollowupTemplates: []string{"Redis 6 的多线程多在哪里？", "大 key 和持久化会带来什么问题？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
		{
			ID:                newID("qt"),
			Mode:              domain.ModeBasics,
			Topic:             "kafka",
			Prompt:            "Kafka 消费幂等一般怎么做？",
			FocusPoints:       []string{"业务幂等键", "重复消费", "事务边界", "offset 提交时机", "失败重试策略"},
			BadAnswers:        []string{"只说 Kafka 自己保证", "只背 at least once，不落到业务实现"},
			FollowupTemplates: []string{"为什么 offset 不能先提交？", "如果重试和 DLQ 设计不好会怎样？"},
			ScoreWeights: map[string]float64{
				"准确性":   30,
				"完整性":   25,
				"落地感":   15,
				"表达清晰度": 15,
				"抗追问能力": 15,
			},
		},
	}

	for _, template := range templates {
		if _, err := s.db.Exec(
			`INSERT INTO question_templates (
				id, mode, topic, prompt, focus_points_json, bad_answers_json, followup_templates_json, score_weights_json
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			template.ID,
			template.Mode,
			template.Topic,
			template.Prompt,
			mustJSON(template.FocusPoints),
			mustJSON(template.BadAnswers),
			mustJSON(template.FollowupTemplates),
			mustJSON(template.ScoreWeights),
		); err != nil {
			return fmt.Errorf("insert question template: %w", err)
		}
	}

	return nil
}

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

func (s *Store) CreateImportedProject(ctx context.Context, analysis *domain.AnalyzeRepoResponse) (*domain.ProjectProfile, error) {
	projectID := newID("proj")
	now := nowUTC()

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

func (s *Store) ListQuestionTemplatesByTopic(ctx context.Context, topic string) ([]domain.QuestionTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, mode, topic, prompt, focus_points_json, bad_answers_json, followup_templates_json, score_weights_json
		FROM question_templates
		WHERE topic = ?
		ORDER BY id ASC
	`, topic)
	if err != nil {
		return nil, fmt.Errorf("list question templates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	templates := make([]domain.QuestionTemplate, 0)
	for rows.Next() {
		template, err := scanQuestionTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *template)
	}

	return templates, nil
}

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

func (s *Store) CreateReview(ctx context.Context, review *domain.ReviewCard) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO review_cards (id, session_id, overall, highlights_json, gaps_json, suggested_topics_json, next_training_focus_json, score_breakdown_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			overall = excluded.overall,
			highlights_json = excluded.highlights_json,
			gaps_json = excluded.gaps_json,
			suggested_topics_json = excluded.suggested_topics_json,
			next_training_focus_json = excluded.next_training_focus_json,
			score_breakdown_json = excluded.score_breakdown_json
	`,
		review.ID,
		review.SessionID,
		review.Overall,
		mustJSON(review.Highlights),
		mustJSON(review.Gaps),
		mustJSON(review.SuggestedTopics),
		mustJSON(review.NextTrainingFocus),
		mustJSON(review.ScoreBreakdown),
		nowUTC(),
	)
	if err != nil {
		return fmt.Errorf("create review: %w", err)
	}

	return nil
}

func (s *Store) GetReview(ctx context.Context, reviewID string) (*domain.ReviewCard, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, session_id, overall, highlights_json, gaps_json, suggested_topics_json, next_training_focus_json, score_breakdown_json, created_at
		FROM review_cards
		WHERE id = ?
	`, reviewID)
	review, err := scanReviewCard(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return review, nil
}

func (s *Store) UpsertWeaknesses(ctx context.Context, sessionID string, hits []domain.WeaknessHit) error {
	now := nowUTC()
	for _, hit := range dedupeWeaknessHits(hits) {
		if hit.Label == "" || hit.Kind == "" {
			continue
		}

		var existingID string
		var currentSeverity float64
		var frequency int
		err := s.db.QueryRowContext(ctx, `SELECT id, severity, frequency FROM weakness_tags WHERE kind = ? AND label = ?`, hit.Kind, hit.Label).Scan(&existingID, &currentSeverity, &frequency)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			_, err = s.db.ExecContext(ctx, `
				INSERT INTO weakness_tags (id, kind, label, severity, frequency, last_seen_at, evidence_session_id)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, newID("weak"), hit.Kind, hit.Label, math.Min(1.0, hit.Severity), 1, now, sessionID)
		case err == nil:
			_, err = s.db.ExecContext(ctx, `
				UPDATE weakness_tags
				SET severity = ?, frequency = ?, last_seen_at = ?, evidence_session_id = ?
				WHERE id = ?
			`, math.Min(1.5, currentSeverity+(hit.Severity*0.35)), frequency+1, now, sessionID, existingID)
		}
		if err != nil {
			return fmt.Errorf("upsert weakness: %w", err)
		}
	}

	return nil
}

func (s *Store) RelieveWeakness(ctx context.Context, kind, label string, amount float64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE weakness_tags
		SET severity = CASE
			WHEN severity - ? < 0 THEN 0
			ELSE severity - ?
		END
		WHERE kind = ? AND label = ?
	`, amount, amount, kind, label)
	if err != nil {
		return fmt.Errorf("relieve weakness: %w", err)
	}

	return nil
}

func (s *Store) ListWeaknesses(ctx context.Context, limit int) ([]domain.WeaknessTag, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, label, severity, frequency, last_seen_at, evidence_session_id
		FROM weakness_tags
		ORDER BY severity DESC, frequency DESC, last_seen_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list weaknesses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.WeaknessTag, 0)
	for rows.Next() {
		item, err := scanWeaknessTag(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}

	return items, nil
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

func scanUserProfile(scanner interface{ Scan(dest ...any) error }) (*domain.UserProfile, error) {
	var (
		id                       int64
		targetRole               string
		targetCompanyType        string
		currentStage             string
		applicationDeadline      string
		techStacksJSON           string
		primaryProjectsJSON      string
		selfReportedWeaknessJSON string
		createdAt                string
		updatedAt                string
	)

	if err := scanner.Scan(&id, &targetRole, &targetCompanyType, &currentStage, &applicationDeadline, &techStacksJSON, &primaryProjectsJSON, &selfReportedWeaknessJSON, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	profile := &domain.UserProfile{
		ID:                   id,
		TargetRole:           targetRole,
		TargetCompanyType:    targetCompanyType,
		CurrentStage:         currentStage,
		ApplicationDeadline:  parseNullableTime(applicationDeadline),
		TechStacks:           parseStringList(techStacksJSON),
		PrimaryProjects:      parseStringList(primaryProjectsJSON),
		SelfReportedWeakness: parseStringList(selfReportedWeaknessJSON),
		CreatedAt:            parseTime(createdAt),
		UpdatedAt:            parseTime(updatedAt),
	}

	return profile, nil
}

func scanProjectProfile(scanner interface{ Scan(dest ...any) error }) (*domain.ProjectProfile, error) {
	var (
		id, name, repoURL, defaultBranch, importCommit, summary, techStackJSON, highlightsJSON, challengesJSON string
		tradeoffsJSON, ownershipJSON, followupJSON, importStatus, createdAt, updatedAt                         string
	)

	if err := scanner.Scan(&id, &name, &repoURL, &defaultBranch, &importCommit, &summary, &techStackJSON, &highlightsJSON, &challengesJSON, &tradeoffsJSON, &ownershipJSON, &followupJSON, &importStatus, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	project := &domain.ProjectProfile{
		ID:              id,
		Name:            name,
		RepoURL:         repoURL,
		DefaultBranch:   defaultBranch,
		ImportCommit:    importCommit,
		Summary:         summary,
		TechStack:       parseStringList(techStackJSON),
		Highlights:      parseStringList(highlightsJSON),
		Challenges:      parseStringList(challengesJSON),
		Tradeoffs:       parseStringList(tradeoffsJSON),
		OwnershipPoints: parseStringList(ownershipJSON),
		FollowupPoints:  parseStringList(followupJSON),
		ImportStatus:    importStatus,
		CreatedAt:       parseTime(createdAt),
		UpdatedAt:       parseTime(updatedAt),
	}

	return project, nil
}

func scanRepoChunk(scanner interface{ Scan(dest ...any) error }) (*domain.RepoChunk, error) {
	var id, projectID, filePath, fileType, content, ftsKey, createdAt string
	var importance float64
	if err := scanner.Scan(&id, &projectID, &filePath, &fileType, &content, &importance, &ftsKey, &createdAt); err != nil {
		return nil, err
	}

	return &domain.RepoChunk{
		ID:         id,
		ProjectID:  projectID,
		FilePath:   filePath,
		FileType:   fileType,
		Content:    content,
		Importance: importance,
		FTSKey:     ftsKey,
		CreatedAt:  parseTime(createdAt),
	}, nil
}

func scanQuestionTemplate(scanner interface{ Scan(dest ...any) error }) (*domain.QuestionTemplate, error) {
	var id, mode, topic, prompt, focusPointsJSON, badAnswersJSON, followupTemplatesJSON, scoreWeightsJSON string
	if err := scanner.Scan(&id, &mode, &topic, &prompt, &focusPointsJSON, &badAnswersJSON, &followupTemplatesJSON, &scoreWeightsJSON); err != nil {
		return nil, err
	}

	weights := map[string]float64{}
	_ = json.Unmarshal([]byte(scoreWeightsJSON), &weights)

	return &domain.QuestionTemplate{
		ID:                id,
		Mode:              mode,
		Topic:             topic,
		Prompt:            prompt,
		FocusPoints:       parseStringList(focusPointsJSON),
		BadAnswers:        parseStringList(badAnswersJSON),
		FollowupTemplates: parseStringList(followupTemplatesJSON),
		ScoreWeights:      weights,
	}, nil
}

func scanTrainingSession(scanner interface{ Scan(dest ...any) error }) (*domain.TrainingSession, error) {
	var id, mode, topic, projectID, intensity, status, startedAt, endedAt, reviewID, createdAt, updatedAt string
	var totalScore float64
	if err := scanner.Scan(&id, &mode, &topic, &projectID, &intensity, &status, &totalScore, &startedAt, &endedAt, &reviewID, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	return &domain.TrainingSession{
		ID:         id,
		Mode:       mode,
		Topic:      topic,
		ProjectID:  projectID,
		Intensity:  intensity,
		Status:     status,
		TotalScore: totalScore,
		StartedAt:  parseNullableTime(startedAt),
		EndedAt:    parseNullableTime(endedAt),
		ReviewID:   reviewID,
		CreatedAt:  parseTime(createdAt),
		UpdatedAt:  parseTime(updatedAt),
	}, nil
}

func scanTrainingTurn(scanner interface{ Scan(dest ...any) error }) (*domain.TrainingTurn, error) {
	var (
		id, sessionID, stage, question, expectedPointsJSON, answer, evaluationJSON, followupQuestion       string
		followupPointsJSON, followupAnswer, followupEvaluationJSON, weaknessHitsJSON, createdAt, updatedAt string
		turnIndex                                                                                          int
	)

	if err := scanner.Scan(&id, &sessionID, &turnIndex, &stage, &question, &expectedPointsJSON, &answer, &evaluationJSON, &followupQuestion, &followupPointsJSON, &followupAnswer, &followupEvaluationJSON, &weaknessHitsJSON, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	turn := &domain.TrainingTurn{
		ID:                    id,
		SessionID:             sessionID,
		TurnIndex:             turnIndex,
		Stage:                 stage,
		Question:              question,
		ExpectedPoints:        parseStringList(expectedPointsJSON),
		Answer:                answer,
		FollowupQuestion:      followupQuestion,
		FollowupExpectedPoint: parseStringList(followupPointsJSON),
		FollowupAnswer:        followupAnswer,
		WeaknessHits:          parseWeaknessHits(weaknessHitsJSON),
		CreatedAt:             parseTime(createdAt),
		UpdatedAt:             parseTime(updatedAt),
	}

	if strings.TrimSpace(evaluationJSON) != "" && evaluationJSON != "null" && evaluationJSON != "{}" {
		evaluation := &domain.EvaluationResult{}
		_ = json.Unmarshal([]byte(evaluationJSON), evaluation)
		turn.Evaluation = evaluation
	}

	if strings.TrimSpace(followupEvaluationJSON) != "" && followupEvaluationJSON != "null" && followupEvaluationJSON != "{}" {
		evaluation := &domain.EvaluationResult{}
		_ = json.Unmarshal([]byte(followupEvaluationJSON), evaluation)
		turn.FollowupEvaluation = evaluation
	}

	return turn, nil
}

func scanReviewCard(scanner interface{ Scan(dest ...any) error }) (*domain.ReviewCard, error) {
	var id, sessionID, overall, highlightsJSON, gapsJSON, suggestedTopicsJSON, nextTrainingFocusJSON, scoreBreakdownJSON, createdAt string
	if err := scanner.Scan(&id, &sessionID, &overall, &highlightsJSON, &gapsJSON, &suggestedTopicsJSON, &nextTrainingFocusJSON, &scoreBreakdownJSON, &createdAt); err != nil {
		return nil, err
	}

	breakdown := map[string]float64{}
	_ = json.Unmarshal([]byte(scoreBreakdownJSON), &breakdown)

	return &domain.ReviewCard{
		ID:                id,
		SessionID:         sessionID,
		Overall:           overall,
		Highlights:        parseStringList(highlightsJSON),
		Gaps:              parseStringList(gapsJSON),
		SuggestedTopics:   parseStringList(suggestedTopicsJSON),
		NextTrainingFocus: parseStringList(nextTrainingFocusJSON),
		ScoreBreakdown:    breakdown,
		CreatedAt:         parseTime(createdAt),
	}, nil
}

func scanWeaknessTag(scanner interface{ Scan(dest ...any) error }) (*domain.WeaknessTag, error) {
	var id, kind, label, lastSeenAt, evidenceSessionID string
	var severity float64
	var frequency int
	if err := scanner.Scan(&id, &kind, &label, &severity, &frequency, &lastSeenAt, &evidenceSessionID); err != nil {
		return nil, err
	}

	return &domain.WeaknessTag{
		ID:                id,
		Kind:              kind,
		Label:             label,
		Severity:          severity,
		Frequency:         frequency,
		LastSeenAt:        parseTime(lastSeenAt),
		EvidenceSessionID: evidenceSessionID,
	}, nil
}

func parseStringList(raw string) []string {
	items := make([]string, 0)
	_ = json.Unmarshal([]byte(raw), &items)
	return items
}

func parseWeaknessHits(raw string) []domain.WeaknessHit {
	items := make([]domain.WeaknessHit, 0)
	_ = json.Unmarshal([]byte(raw), &items)
	return items
}

func parseTime(raw string) time.Time {
	if raw == "" {
		return time.Time{}
	}

	for _, layout := range []string{time.RFC3339Nano, time.DateOnly} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			if layout == time.DateOnly {
				return time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
			}

			return parsed
		}
	}

	return time.Time{}
}

func parseNullableTime(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	parsed := parseTime(raw)
	if parsed.IsZero() {
		return nil
	}
	return &parsed
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func toNullableTimeString(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func normalizeDateString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	parsed := parseTime(raw)
	if parsed.IsZero() {
		return raw
	}

	return parsed.UTC().Format(time.RFC3339Nano)
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func newID(prefix string) string {
	buffer := make([]byte, 8)
	if _, err := rand.Read(buffer); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buffer))
}

func buildFTSQuery(raw string) string {
	parts := strings.FieldsFunc(strings.ToLower(raw), func(r rune) bool {
		return (r < 'a' || r > 'z') && (r < '0' || r > '9') && (r < 'A' || r > 'Z') && r <= 127
	})
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 2 {
			continue
		}
		filtered = append(filtered, fmt.Sprintf("%q", part))
	}
	return strings.Join(filtered, " OR ")
}

func dedupeWeaknessHits(hits []domain.WeaknessHit) []domain.WeaknessHit {
	type key struct {
		kind  string
		label string
	}

	result := make([]domain.WeaknessHit, 0, len(hits))
	seen := map[key]domain.WeaknessHit{}
	for _, hit := range hits {
		k := key{kind: hit.Kind, label: hit.Label}
		existing, ok := seen[k]
		if !ok || hit.Severity > existing.Severity {
			seen[k] = hit
		}
	}

	for _, hit := range seen {
		result = append(result, hit)
	}

	return result
}
