package sqlite

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"practicehelper/server/internal/domain"
)

func Bootstrap(db *sql.DB) error {
	if err := migrate(db); err != nil {
		return err
	}

	if err := seedQuestionTemplates(db); err != nil {
		return err
	}

	return nil
}

func migrate(db *sql.DB) error {
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
			active_job_target_id TEXT NOT NULL DEFAULT '',
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
		`CREATE TABLE IF NOT EXISTS project_import_jobs (
			id TEXT PRIMARY KEY,
			repo_url TEXT NOT NULL,
			status TEXT NOT NULL,
			stage TEXT NOT NULL,
			message TEXT NOT NULL,
			error_message TEXT NOT NULL DEFAULT '',
			project_id TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			started_at TEXT NOT NULL DEFAULT '',
			finished_at TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS job_targets (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			company_name TEXT NOT NULL DEFAULT '',
			source_text TEXT NOT NULL,
			latest_analysis_id TEXT NOT NULL DEFAULT '',
			latest_analysis_status TEXT NOT NULL DEFAULT 'idle',
			last_used_at TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS job_target_analysis_runs (
			id TEXT PRIMARY KEY,
			job_target_id TEXT NOT NULL REFERENCES job_targets(id) ON DELETE CASCADE,
			source_text_snapshot TEXT NOT NULL,
			status TEXT NOT NULL,
			error_message TEXT NOT NULL DEFAULT '',
			summary TEXT NOT NULL DEFAULT '',
			must_have_skills_json TEXT NOT NULL DEFAULT '[]',
			bonus_skills_json TEXT NOT NULL DEFAULT '[]',
			responsibilities_json TEXT NOT NULL DEFAULT '[]',
			evaluation_focus_json TEXT NOT NULL DEFAULT '[]',
			created_at TEXT NOT NULL,
			finished_at TEXT NOT NULL DEFAULT ''
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
			job_target_id TEXT NOT NULL DEFAULT '',
			job_target_analysis_id TEXT NOT NULL DEFAULT '',
			intensity TEXT NOT NULL,
			status TEXT NOT NULL,
			max_turns INTEGER NOT NULL DEFAULT 2,
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
		`CREATE TABLE IF NOT EXISTS weakness_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			weakness_id TEXT NOT NULL,
			session_id TEXT NOT NULL DEFAULT '',
			severity REAL NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS review_schedule (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL,
			review_card_id TEXT NOT NULL DEFAULT '',
			weakness_tag_id TEXT NOT NULL DEFAULT '',
			topic TEXT NOT NULL DEFAULT '',
			next_review_at TEXT NOT NULL,
			interval_days INTEGER NOT NULL DEFAULT 1,
			ease_factor REAL NOT NULL DEFAULT 2.5,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS evaluation_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			session_id TEXT NOT NULL DEFAULT '',
			turn_id TEXT NOT NULL DEFAULT '',
			flow_name TEXT NOT NULL,
			model_name TEXT NOT NULL DEFAULT '',
			latency_ms REAL NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("run migration: %w", err)
		}
	}

	if err := ensureColumn(db, "review_cards", "top_fix", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "review_cards", "top_fix_reason", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "review_cards", "recommended_next_json", "TEXT NOT NULL DEFAULT 'null'"); err != nil {
		return err
	}
	if err := ensureColumn(db, "training_sessions", "job_target_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "training_sessions", "job_target_analysis_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "user_profile", "active_job_target_id", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "training_sessions", "max_turns", "INTEGER NOT NULL DEFAULT 2"); err != nil {
		return err
	}

	return nil
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("query table info for %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			primaryKey int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &primaryKey); err != nil {
			return fmt.Errorf("scan table info for %s: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table info for %s: %w", table, err)
	}

	if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}
	return nil
}

func seedQuestionTemplates(db *sql.DB) error {
	templates, err := loadQuestionTemplates()
	if err != nil {
		return err
	}

	for _, template := range templates {
		if template.ID == "" {
			template.ID = newID("qt")
		}
		result, err := db.Exec(
			`INSERT INTO question_templates (
				id, mode, topic, prompt, focus_points_json, bad_answers_json, followup_templates_json, score_weights_json
			)
			SELECT ?, ?, ?, ?, ?, ?, ?, ?
			WHERE NOT EXISTS (
				SELECT 1
				FROM question_templates
				WHERE mode = ? AND topic = ? AND prompt = ?
			)`,
			template.ID,
			template.Mode,
			template.Topic,
			template.Prompt,
			mustJSON(template.FocusPoints),
			mustJSON(template.BadAnswers),
			mustJSON(template.FollowupTemplates),
			mustJSON(template.ScoreWeights),
			template.Mode,
			template.Topic,
			template.Prompt,
		)
		if err != nil {
			return fmt.Errorf("insert question template: %w", err)
		}
		if affected, err := result.RowsAffected(); err != nil {
			return fmt.Errorf("count inserted question template rows: %w", err)
		} else if affected > 1 {
			return fmt.Errorf("unexpected inserted rows for question template %s/%s", template.Topic, template.Prompt)
		}
	}

	return nil
}

var seedFilePaths = []string{
	"data/seed/question_templates.json",
	"../data/seed/question_templates.json",
	"../../data/seed/question_templates.json",
	"../../../data/seed/question_templates.json",
	"../../../../data/seed/question_templates.json",
}

func loadQuestionTemplates() ([]domain.QuestionTemplate, error) {
	for _, path := range seedFilePaths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var templates []domain.QuestionTemplate
		if err := json.Unmarshal(data, &templates); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		return templates, nil
	}
	return nil, nil
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
