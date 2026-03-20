package repo

import (
	"fmt"

	"practicehelper/server/internal/domain"
)

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
