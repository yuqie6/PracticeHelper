package sqlite

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	if err := ensureColumn(db, "training_turns", "followup_question", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "training_turns", "followup_expected_points_json", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
		return err
	}
	if err := ensureColumn(db, "training_turns", "followup_answer", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(db, "training_turns", "followup_evaluation_json", "TEXT NOT NULL DEFAULT '{}'"); err != nil {
		return err
	}
	if err := migrateLegacyFollowupTurns(db); err != nil {
		return err
	}

	return nil
}

type legacyFollowupTurn struct {
	sessionID          string
	baseTurnIndex      int
	question           string
	expectedPointsJSON string
	answer             string
	evaluationJSON     string
	createdAt          string
	updatedAt          string
}

func migrateLegacyFollowupTurns(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT session_id, turn_index, followup_question, followup_expected_points_json,
			followup_answer, followup_evaluation_json, created_at, updated_at
		FROM training_turns
		WHERE TRIM(followup_question) != ''
			OR TRIM(followup_answer) != ''
			OR (TRIM(followup_evaluation_json) != '' AND followup_evaluation_json NOT IN ('{}', 'null'))
	`)
	if err != nil {
		return fmt.Errorf("query legacy followup turns: %w", err)
	}
	defer func() { _ = rows.Close() }()

	legacyTurns := make([]legacyFollowupTurn, 0)
	for rows.Next() {
		var item legacyFollowupTurn
		if err := rows.Scan(
			&item.sessionID,
			&item.baseTurnIndex,
			&item.question,
			&item.expectedPointsJSON,
			&item.answer,
			&item.evaluationJSON,
			&item.createdAt,
			&item.updatedAt,
		); err != nil {
			return fmt.Errorf("scan legacy followup turn: %w", err)
		}
		legacyTurns = append(legacyTurns, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate legacy followup turns: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin legacy followup migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	for _, item := range legacyTurns {
		var existingCount int
		if err := tx.QueryRow(`
			SELECT COUNT(*)
			FROM training_turns
			WHERE session_id = ? AND turn_index = ?
		`, item.sessionID, item.baseTurnIndex+1).Scan(&existingCount); err != nil {
			return fmt.Errorf("check migrated followup turn: %w", err)
		}
		if existingCount > 0 {
			continue
		}

		expectedPointsJSON := strings.TrimSpace(item.expectedPointsJSON)
		if expectedPointsJSON == "" {
			expectedPointsJSON = "[]"
		}
		evaluationJSON := strings.TrimSpace(item.evaluationJSON)
		if evaluationJSON == "" {
			evaluationJSON = "{}"
		}

		if _, err := tx.Exec(`
			INSERT INTO training_turns (
				id, session_id, turn_index, stage, question, expected_points_json,
				answer, evaluation_json, weakness_hits_json, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			newID("turn"),
			item.sessionID,
			item.baseTurnIndex+1,
			"question",
			item.question,
			expectedPointsJSON,
			item.answer,
			evaluationJSON,
			"[]",
			item.createdAt,
			item.updatedAt,
		); err != nil {
			return fmt.Errorf("insert migrated followup turn: %w", err)
		}
	}

	if _, err := tx.Exec(`
		UPDATE training_sessions
		SET status = ?
		WHERE status = ?
	`, domain.StatusWaitingAnswer, "followup"); err != nil {
		return fmt.Errorf("migrate legacy followup status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy followup migration: %w", err)
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
	templates, err := loadTemplateSeed()
	if err != nil {
		return err
	}

	for _, template := range templates {
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

var defaultScoreWeights = map[string]float64{
	"准确性":   30,
	"完整性":   25,
	"落地感":   15,
	"表达清晰度": 15,
	"抗追问能力": 15,
}

type templateSeed struct {
	Mode              string             `json:"mode"`
	Topic             string             `json:"topic"`
	Prompt            string             `json:"prompt"`
	FocusPoints       []string           `json:"focus_points"`
	BadAnswers        []string           `json:"bad_answers"`
	FollowupTemplates []string           `json:"followup_templates"`
	ScoreWeights      map[string]float64 `json:"score_weights,omitempty"`
}

func loadTemplateSeed() ([]domain.QuestionTemplate, error) {
	seedPaths := []string{
		"data/seed/question_templates.json",
		"../data/seed/question_templates.json",
		"../../data/seed/question_templates.json",
		"../../../data/seed/question_templates.json",
		"../../../../data/seed/question_templates.json",
	}
	if _, file, _, ok := runtime.Caller(0); ok {
		seedPaths = append([]string{
			filepath.Join(filepath.Dir(file), "../../../../data/seed/question_templates.json"),
		}, seedPaths...)
	}

	var raw []byte
	var readErr error
	for _, path := range seedPaths {
		raw, readErr = os.ReadFile(path)
		if readErr == nil {
			break
		}
	}
	if readErr != nil {
		return nil, fmt.Errorf("read question templates seed: %w", readErr)
	}

	var seeds []templateSeed
	if err := json.Unmarshal(raw, &seeds); err != nil {
		return nil, fmt.Errorf("parse question templates seed: %w", err)
	}

	templates := make([]domain.QuestionTemplate, 0, len(seeds))
	for _, s := range seeds {
		weights := s.ScoreWeights
		if len(weights) == 0 {
			weights = defaultScoreWeights
		}
		templates = append(templates, domain.QuestionTemplate{
			ID:                newID("qt"),
			Mode:              s.Mode,
			Topic:             s.Topic,
			Prompt:            s.Prompt,
			FocusPoints:       s.FocusPoints,
			BadAnswers:        s.BadAnswers,
			FollowupTemplates: s.FollowupTemplates,
			ScoreWeights:      weights,
		})
	}

	return templates, nil
}
