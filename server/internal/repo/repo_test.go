package repo

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/infra/sqlite"
)

func TestNewStoreUsesBootstrappedSQLiteSchema(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM question_templates`).Scan(&count); err != nil {
		t.Fatalf("seeded question_templates query failed: %v", err)
	}

	if count == 0 {
		t.Fatal("expected seeded question templates, got 0")
	}
}

func TestSeedQuestionTemplatesProvideAtLeastFiveBasicsPromptsPerTopic(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	rows, err := store.db.Query(`
		SELECT topic, COUNT(*)
		FROM question_templates
		WHERE mode = ?
		GROUP BY topic
	`, domain.ModeBasics)
	if err != nil {
		t.Fatalf("query template coverage: %v", err)
	}
	defer func() { _ = rows.Close() }()

	counts := map[string]int{}
	for rows.Next() {
		var topic string
		var count int
		if err := rows.Scan(&topic, &count); err != nil {
			t.Fatalf("scan template coverage: %v", err)
		}
		counts[topic] = count
	}

	for _, topic := range []string{"go", "redis", "kafka"} {
		if counts[topic] < 5 {
			t.Fatalf("expected at least 5 basics templates for %s, got %d", topic, counts[topic])
		}
	}
}

func TestBootstrapBackfillsMissingQuestionTemplatesIntoExistingDatabase(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	if _, err := store.db.Exec(`
		DELETE FROM question_templates
		WHERE mode = ? AND topic = ? AND prompt = ?
	`, domain.ModeBasics, "go", "Go 的 channel 和 mutex 各适合什么场景？"); err != nil {
		t.Fatalf("delete seeded template: %v", err)
	}

	if err := sqlite.Bootstrap(store.db); err != nil {
		t.Fatalf("Bootstrap() backfill error = %v", err)
	}

	var count int
	if err := store.db.QueryRow(`
		SELECT COUNT(*)
		FROM question_templates
		WHERE mode = ? AND topic = ? AND prompt = ?
	`, domain.ModeBasics, "go", "Go 的 channel 和 mutex 各适合什么场景？").Scan(&count); err != nil {
		t.Fatalf("query backfilled template: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected deleted template to be backfilled exactly once, got %d", count)
	}
}

func TestBootstrapMigratesLegacyFollowupSessionStatusAndPendingTurn(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "practicehelper.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
		CREATE TABLE training_sessions (
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
		);
		CREATE TABLE training_turns (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
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
		);
	`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`
		INSERT INTO training_sessions (
			id, mode, topic, project_id, intensity, status, total_score, started_at, ended_at, review_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "sess_legacy_followup", domain.ModeBasics, "go", "", "standard", "followup", 72.5, now, "", "", now, now); err != nil {
		t.Fatalf("insert legacy session: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO training_turns (
			id, session_id, turn_index, stage, question, expected_points_json, answer, evaluation_json,
			followup_question, followup_expected_points_json, followup_answer, followup_evaluation_json,
			weakness_hits_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"turn_legacy_followup",
		"sess_legacy_followup",
		1,
		"question",
		"Go 的 channel 和 mutex 什么时候各用什么？",
		`["共享内存","所有权转移"]`,
		"主回答已经提交",
		`{"score":72.5,"headline":"主回答基本过线。","strengths":["回答有主干"],"gaps":["还缺性能取舍"],"followup_intent":"继续确认性能与复杂度取舍。","followup_question":"如果你发现 channel 用多了，你会怎么收敛？","followup_expected_points":["性能","复杂度"]}`,
		"如果你发现 channel 用多了，你会怎么收敛？",
		`["性能","复杂度"]`,
		"",
		`{}`,
		"[]",
		now,
		now,
	); err != nil {
		t.Fatalf("insert legacy turn: %v", err)
	}

	if err := sqlite.Bootstrap(db); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	store := New(db)
	session, err := store.GetSession(context.Background(), "sess_legacy_followup")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("expected migrated session, got nil")
	}
	if session.Status != domain.StatusWaitingAnswer {
		t.Fatalf("expected waiting_answer after migration, got %s", session.Status)
	}
	if session.MaxTurns != 2 {
		t.Fatalf("expected default max_turns 2, got %d", session.MaxTurns)
	}
	if len(session.Turns) != 2 {
		t.Fatalf("expected 2 turns after migration, got %d", len(session.Turns))
	}
	if session.Turns[1].TurnIndex != 2 {
		t.Fatalf("expected migrated turn index 2, got %d", session.Turns[1].TurnIndex)
	}
	if session.Turns[1].Question != "如果你发现 channel 用多了，你会怎么收敛？" {
		t.Fatalf("unexpected migrated followup question: %q", session.Turns[1].Question)
	}
	if session.Turns[1].Answer != "" {
		t.Fatalf("expected pending migrated followup answer to stay empty, got %q", session.Turns[1].Answer)
	}
	if session.Turns[1].Evaluation != nil {
		t.Fatalf("expected pending migrated followup evaluation to be nil, got %+v", session.Turns[1].Evaluation)
	}

	resumable, err := store.GetLatestResumableSession(context.Background())
	if err != nil {
		t.Fatalf("GetLatestResumableSession() error = %v", err)
	}
	if resumable == nil || resumable.ID != session.ID {
		t.Fatalf("expected migrated session to be resumable, got %#v", resumable)
	}
}

func TestBootstrapMigratesLegacyAnsweredFollowupIntoSecondTurn(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "practicehelper.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
		CREATE TABLE training_sessions (
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
		);
		CREATE TABLE training_turns (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
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
		);
	`); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := db.Exec(`
		INSERT INTO training_sessions (
			id, mode, topic, project_id, intensity, status, total_score, started_at, ended_at, review_id, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "sess_legacy_completed", domain.ModeBasics, "redis", "", "standard", domain.StatusCompleted, 68, now, now, "review_legacy", now, now); err != nil {
		t.Fatalf("insert legacy session: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO training_turns (
			id, session_id, turn_index, stage, question, expected_points_json, answer, evaluation_json,
			followup_question, followup_expected_points_json, followup_answer, followup_evaluation_json,
			weakness_hits_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"turn_legacy_completed",
		"sess_legacy_completed",
		1,
		"question",
		"Redis 为什么快？",
		`["内存访问","事件循环"]`,
		"因为数据在内存里。",
		`{"score":63,"headline":"主回答有方向，但细节不足。","strengths":["答到了内存"],"gaps":["没讲线程模型"],"followup_intent":"确认你是否理解单线程模型的边界。","followup_question":"单线程为什么还能扛住高并发？","followup_expected_points":["IO 多路复用","避免锁竞争"]}`,
		"单线程为什么还能扛住高并发？",
		`["IO 多路复用","避免锁竞争"]`,
		"因为它把并发放在 IO 多路复用和事件循环上，减少了锁竞争。",
		`{"score":81,"headline":"追问回答补上了关键机制。","strengths":["提到了 IO 多路复用"],"gaps":[],"suggestion":"补上单线程边界。","followup_intent":"","followup_question":"","followup_expected_points":[]}`,
		"[]",
		now,
		now,
	); err != nil {
		t.Fatalf("insert legacy turn: %v", err)
	}

	if err := sqlite.Bootstrap(db); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	store := New(db)
	session, err := store.GetSession(context.Background(), "sess_legacy_completed")
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if session == nil {
		t.Fatal("expected migrated session, got nil")
	}
	if len(session.Turns) != 2 {
		t.Fatalf("expected 2 turns after migration, got %d", len(session.Turns))
	}
	if session.Turns[1].Answer != "因为它把并发放在 IO 多路复用和事件循环上，减少了锁竞争。" {
		t.Fatalf("unexpected migrated followup answer: %q", session.Turns[1].Answer)
	}
	if session.Turns[1].Evaluation == nil {
		t.Fatal("expected migrated followup evaluation, got nil")
	}
	if session.Turns[1].Evaluation.Score != 81 {
		t.Fatalf("expected migrated followup score 81, got %v", session.Turns[1].Evaluation.Score)
	}
}

func TestSaveUserProfileParsesDateOnlyDeadline(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	deadline := "2026-04-01"
	profile, err := store.SaveUserProfile(context.Background(), domain.UserProfileInput{
		TargetRole:           "Go Backend Engineer",
		TargetCompanyType:    "Startup",
		CurrentStage:         "Interview prep",
		ApplicationDeadline:  &deadline,
		TechStacks:           []string{"Go"},
		PrimaryProjects:      []string{"PracticeHelper"},
		SelfReportedWeakness: []string{"Kafka"},
	})
	if err != nil {
		t.Fatalf("SaveUserProfile() error = %v", err)
	}

	if profile.ApplicationDeadline == nil {
		t.Fatal("expected application deadline to be parsed")
	}

	got := profile.ApplicationDeadline.UTC().Format(time.DateOnly)
	if got != deadline {
		t.Fatalf("expected deadline %s, got %s", deadline, got)
	}
}

func TestProjectImportJobLifecycle(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	job, err := store.CreateProjectImportJob(ctx, "https://github.com/octocat/Hello-World")
	if err != nil {
		t.Fatalf("CreateProjectImportJob() error = %v", err)
	}

	if job.Status != domain.ProjectImportStatusQueued {
		t.Fatalf("expected queued status, got %s", job.Status)
	}

	startedAt := time.Now().UTC()
	if err := store.UpdateProjectImportJobStatus(
		ctx,
		job.ID,
		domain.ProjectImportStatusRunning,
		domain.ProjectImportStageAnalyzing,
		"running",
		"",
		"",
		&startedAt,
		nil,
	); err != nil {
		t.Fatalf("UpdateProjectImportJobStatus() running error = %v", err)
	}

	activeJob, err := store.FindActiveProjectImportJobByRepoURL(ctx, job.RepoURL)
	if err != nil {
		t.Fatalf("FindActiveProjectImportJobByRepoURL() error = %v", err)
	}
	if activeJob == nil || activeJob.ID != job.ID {
		t.Fatalf("expected active job %s, got %+v", job.ID, activeJob)
	}

	finishedAt := startedAt.Add(2 * time.Minute)
	if err := store.UpdateProjectImportJobStatus(
		ctx,
		job.ID,
		domain.ProjectImportStatusFailed,
		domain.ProjectImportStageFailed,
		"failed",
		"boom",
		"",
		nil,
		&finishedAt,
	); err != nil {
		t.Fatalf("UpdateProjectImportJobStatus() failed error = %v", err)
	}

	saved, err := store.GetProjectImportJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetProjectImportJob() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved import job")
	}
	if saved.Status != domain.ProjectImportStatusFailed {
		t.Fatalf("expected failed status, got %s", saved.Status)
	}
	if saved.ErrorMessage != "boom" {
		t.Fatalf("expected error message boom, got %s", saved.ErrorMessage)
	}
	if saved.FinishedAt == nil {
		t.Fatal("expected finished_at to be set")
	}

	if err := store.RetryProjectImportJob(ctx, job.ID, "retrying"); err != nil {
		t.Fatalf("RetryProjectImportJob() error = %v", err)
	}

	retried, err := store.GetProjectImportJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetProjectImportJob() after retry error = %v", err)
	}
	if retried == nil {
		t.Fatal("expected retried job")
	}
	if retried.Status != domain.ProjectImportStatusQueued {
		t.Fatalf("expected queued after retry, got %s", retried.Status)
	}
	if retried.ErrorMessage != "" {
		t.Fatalf("expected cleared error message, got %s", retried.ErrorMessage)
	}
	if retried.FinishedAt != nil {
		t.Fatal("expected finished_at to be cleared after retry")
	}
}

func TestTransitionSessionStatusRequiresCurrentStatusMatch(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	session := &domain.TrainingSession{
		ID:        "sess_transition",
		Mode:      domain.ModeBasics,
		Topic:     "go",
		Intensity: "standard",
		Status:    domain.StatusWaitingAnswer,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_transition",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 goroutine 为什么轻量？",
		ExpectedPoints: []string{"调度", "栈"},
	}
	if err := store.CreateSession(ctx, session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	claimed, err := store.TransitionSessionStatus(
		ctx,
		session.ID,
		[]string{domain.StatusWaitingAnswer},
		domain.StatusEvaluating,
	)
	if err != nil {
		t.Fatalf("TransitionSessionStatus() first call error = %v", err)
	}
	if !claimed {
		t.Fatal("expected first transition to succeed")
	}

	claimed, err = store.TransitionSessionStatus(
		ctx,
		session.ID,
		[]string{domain.StatusWaitingAnswer},
		domain.StatusReviewPending,
	)
	if err != nil {
		t.Fatalf("TransitionSessionStatus() second call error = %v", err)
	}
	if claimed {
		t.Fatal("expected second transition to fail because status already changed")
	}
}

func TestCreateReviewUpdatesExistingRowIDAndRecommendedNext(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	session := &domain.TrainingSession{
		ID:        "sess_review_roundtrip",
		Mode:      domain.ModeBasics,
		Topic:     "go",
		Intensity: "standard",
		Status:    domain.StatusReviewPending,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_review_roundtrip",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 goroutine 为什么轻量？",
		ExpectedPoints: []string{"调度", "栈"},
		Answer:         "因为调度和栈更轻",
	}
	if err := store.CreateSession(ctx, session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	first := &domain.ReviewCard{
		ID:                "review_first",
		SessionID:         session.ID,
		Overall:           "first",
		Highlights:        []string{"结构清楚"},
		Gaps:              []string{"缺细节"},
		SuggestedTopics:   []string{"redis"},
		NextTrainingFocus: []string{"补案例"},
		ScoreBreakdown:    map[string]float64{"表达": 70},
	}
	if err := store.CreateReview(ctx, first); err != nil {
		t.Fatalf("CreateReview() first error = %v", err)
	}

	second := &domain.ReviewCard{
		ID:                "review_second",
		SessionID:         first.SessionID,
		Overall:           "second",
		TopFix:            "先把 trade-off 讲清楚",
		TopFixReason:      "这是项目题说服力的核心缺口",
		Highlights:        []string{"能先给结论"},
		Gaps:              []string{"缺具体代价"},
		SuggestedTopics:   []string{"kafka"},
		NextTrainingFocus: []string{"补设计取舍"},
		RecommendedNext: &domain.NextSession{
			Mode:   domain.ModeBasics,
			Topic:  "kafka",
			Reason: "先补缓存一致性与消息幂等表述",
		},
		ScoreBreakdown: map[string]float64{"表达": 82},
	}
	if err := store.CreateReview(ctx, second); err != nil {
		t.Fatalf("CreateReview() second error = %v", err)
	}

	saved, err := store.GetReview(ctx, second.ID)
	if err != nil {
		t.Fatalf("GetReview() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved review")
	}
	if saved.ID != second.ID {
		t.Fatalf("expected review id %q, got %q", second.ID, saved.ID)
	}
	if saved.TopFix != second.TopFix {
		t.Fatalf("expected top fix %q, got %q", second.TopFix, saved.TopFix)
	}
	if saved.RecommendedNext == nil {
		t.Fatal("expected recommended next session to be persisted")
	}
	if saved.RecommendedNext.Topic != "kafka" {
		t.Fatalf("expected recommended topic kafka, got %q", saved.RecommendedNext.Topic)
	}
}

func TestListWeaknessesAppliesTimeDecayBeforeSorting(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now().UTC()
	older := now.AddDate(0, 0, -45).Format(time.RFC3339Nano)
	recent := now.AddDate(0, 0, -2).Format(time.RFC3339Nano)

	if _, err := store.db.Exec(`
		INSERT INTO weakness_tags (id, kind, label, severity, frequency, last_seen_at, evidence_session_id)
		VALUES
			('weak_old', 'depth', '旧问题', 1.20, 4, ?, 'sess_old'),
			('weak_recent', 'detail', '新问题', 0.92, 1, ?, 'sess_recent')
	`, older, recent); err != nil {
		t.Fatalf("insert weakness tags: %v", err)
	}

	items, err := store.ListWeaknesses(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListWeaknesses() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 weakness items, got %d", len(items))
	}
	if items[0].Label != "新问题" {
		t.Fatalf("expected recent weakness to rank first after decay, got %s", items[0].Label)
	}
	if items[1].Severity >= 1.2 {
		t.Fatalf("expected stale weakness severity to decay, got %.2f", items[1].Severity)
	}
}

func TestUpsertWeaknessesUsesDecayedSeverityAsWriteBaseline(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	oldLastSeenAt := time.Now().UTC().AddDate(0, 0, -42).Format(time.RFC3339Nano)
	if _, err := store.db.Exec(`
		INSERT INTO weakness_tags (id, kind, label, severity, frequency, last_seen_at, evidence_session_id)
		VALUES ('weak_old_baseline', 'depth', '并发排查', 1.20, 3, ?, 'sess_old')
	`, oldLastSeenAt); err != nil {
		t.Fatalf("insert weakness tag: %v", err)
	}

	if err := store.UpsertWeaknesses(context.Background(), "sess_new", []domain.WeaknessHit{
		{Kind: "depth", Label: "并发排查", Severity: 0.60},
	}); err != nil {
		t.Fatalf("UpsertWeaknesses() error = %v", err)
	}

	var severity float64
	var frequency int
	if err := store.db.QueryRow(`
		SELECT severity, frequency
		FROM weakness_tags
		WHERE kind = 'depth' AND label = '并发排查'
	`).Scan(&severity, &frequency); err != nil {
		t.Fatalf("query updated weakness: %v", err)
	}

	if severity >= 1.20 {
		t.Fatalf("expected upsert to use decayed baseline, got %.2f", severity)
	}
	if frequency != 4 {
		t.Fatalf("expected frequency to increment to 4, got %d", frequency)
	}
}

func TestRelieveWeaknessesMatchingTextUsesNormalizedQuestionText(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := store.db.Exec(`
		INSERT INTO weakness_tags (id, kind, label, severity, frequency, last_seen_at, evidence_session_id)
		VALUES
			('weak_match_target', 'topic', 'Redis分布式锁', 1.20, 2, ?, 'sess_old'),
			('weak_match_other', 'topic', 'Kafka监控与排查', 1.00, 1, ?, 'sess_other')
	`, now, now); err != nil {
		t.Fatalf("insert weakness tags: %v", err)
	}

	if err := store.RelieveWeaknessesMatchingText(
		context.Background(),
		[]string{"topic"},
		"Redis 分布式锁为什么不能只靠 SETNX？",
		0.22,
	); err != nil {
		t.Fatalf("RelieveWeaknessesMatchingText() error = %v", err)
	}

	var redisSeverity float64
	if err := store.db.QueryRow(`
		SELECT severity
		FROM weakness_tags
		WHERE id = 'weak_match_target'
	`).Scan(&redisSeverity); err != nil {
		t.Fatalf("query redis severity: %v", err)
	}
	if redisSeverity >= 1.20 {
		t.Fatalf("expected redis weakness to be relieved, got %.2f", redisSeverity)
	}

	var kafkaSeverity float64
	if err := store.db.QueryRow(`
		SELECT severity
		FROM weakness_tags
		WHERE id = 'weak_match_other'
	`).Scan(&kafkaSeverity); err != nil {
		t.Fatalf("query kafka severity: %v", err)
	}
	if kafkaSeverity != 1.00 {
		t.Fatalf("expected unmatched weakness to stay unchanged, got %.2f", kafkaSeverity)
	}
}

func TestJobTargetAnalysisLifecycleAndStaleStatus(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	target, err := store.CreateJobTarget(ctx, domain.JobTargetInput{
		Title:       "后端工程师 - Example",
		CompanyName: "Example",
		SourceText:  "负责高并发后端服务开发，要求 Go、Redis、Kafka 经验。",
	})
	if err != nil {
		t.Fatalf("CreateJobTarget() error = %v", err)
	}

	run, err := store.StartJobTargetAnalysis(ctx, target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() error = %v", err)
	}
	if run.Status != domain.JobTargetAnalysisRunning {
		t.Fatalf("expected running status, got %s", run.Status)
	}

	if err := store.CompleteJobTargetAnalysis(ctx, target.ID, run.ID, &domain.AnalyzeJobTargetResponse{
		Summary:          "核心是在招能独立推进高并发后端系统的人。",
		MustHaveSkills:   []string{"Go", "Redis", "Kafka"},
		BonusSkills:      []string{"Kubernetes"},
		Responsibilities: []string{"负责核心服务设计"},
		EvaluationFocus:  []string{"并发设计取舍"},
	}); err != nil {
		t.Fatalf("CompleteJobTargetAnalysis() error = %v", err)
	}

	saved, err := store.GetJobTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetJobTarget() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved job target")
	}
	if saved.LatestAnalysisStatus != domain.JobTargetAnalysisSucceeded {
		t.Fatalf("expected succeeded status, got %s", saved.LatestAnalysisStatus)
	}
	if saved.LatestSuccessfulAnalysis == nil {
		t.Fatal("expected latest successful analysis to be attached")
	}
	if saved.LatestSuccessfulAnalysis.Summary == "" {
		t.Fatal("expected latest successful analysis summary to be populated")
	}

	updated, err := store.UpdateJobTarget(ctx, target.ID, domain.JobTargetInput{
		Title:       target.Title,
		CompanyName: target.CompanyName,
		SourceText:  target.SourceText + "\n额外补充：需要强故障排查能力。",
	})
	if err != nil {
		t.Fatalf("UpdateJobTarget() error = %v", err)
	}
	if updated.LatestAnalysisStatus != domain.JobTargetAnalysisStale {
		t.Fatalf("expected stale status after source change, got %s", updated.LatestAnalysisStatus)
	}
	if updated.LatestSuccessfulAnalysis == nil {
		t.Fatal("expected latest successful analysis to remain attached after stale update")
	}
	if updated.LatestSuccessfulAnalysis.ID != run.ID {
		t.Fatalf("expected latest successful analysis id %q, got %q", run.ID, updated.LatestSuccessfulAnalysis.ID)
	}

	runs, err := store.ListJobTargetAnalysisRuns(ctx, target.ID)
	if err != nil {
		t.Fatalf("ListJobTargetAnalysisRuns() error = %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 analysis run, got %d", len(runs))
	}
}

func TestCompleteJobTargetAnalysisDoesNotOverwriteStaleTargetState(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	target, err := store.CreateJobTarget(ctx, domain.JobTargetInput{
		Title:       "后端工程师 - Example",
		CompanyName: "Example",
		SourceText:  "要求 Go、Redis、Kafka 经验。",
	})
	if err != nil {
		t.Fatalf("CreateJobTarget() error = %v", err)
	}

	run, err := store.StartJobTargetAnalysis(ctx, target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() error = %v", err)
	}

	if _, err := store.UpdateJobTarget(ctx, target.ID, domain.JobTargetInput{
		Title:       target.Title,
		CompanyName: target.CompanyName,
		SourceText:  target.SourceText + "\n额外补充：需要强故障排查能力。",
	}); err != nil {
		t.Fatalf("UpdateJobTarget() error = %v", err)
	}

	if err := store.CompleteJobTargetAnalysis(ctx, target.ID, run.ID, &domain.AnalyzeJobTargetResponse{
		Summary:          "偏高并发后端。",
		MustHaveSkills:   []string{"Go"},
		Responsibilities: []string{"负责核心服务设计"},
		EvaluationFocus:  []string{"缓存一致性"},
	}); err != nil {
		t.Fatalf("CompleteJobTargetAnalysis() error = %v", err)
	}

	saved, err := store.GetJobTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetJobTarget() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved job target")
	}
	if saved.LatestAnalysisStatus != domain.JobTargetAnalysisStale {
		t.Fatalf("expected stale status to be preserved, got %s", saved.LatestAnalysisStatus)
	}
	if saved.LatestSuccessfulAnalysis == nil || saved.LatestSuccessfulAnalysis.ID != run.ID {
		t.Fatal("expected stale target to keep latest successful analysis attached")
	}
}

func TestOlderFailedJobTargetAnalysisDoesNotClobberNewerRunState(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	target, err := store.CreateJobTarget(ctx, domain.JobTargetInput{
		Title:       "后端工程师 - Example",
		CompanyName: "Example",
		SourceText:  "要求 Go、Redis、Kafka 经验。",
	})
	if err != nil {
		t.Fatalf("CreateJobTarget() error = %v", err)
	}

	firstRun, err := store.StartJobTargetAnalysis(ctx, target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() first error = %v", err)
	}
	secondRun, err := store.StartJobTargetAnalysis(ctx, target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() second error = %v", err)
	}

	if err := store.FailJobTargetAnalysis(ctx, target.ID, firstRun.ID, "boom"); err != nil {
		t.Fatalf("FailJobTargetAnalysis() error = %v", err)
	}

	saved, err := store.GetJobTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetJobTarget() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved job target")
	}
	if saved.LatestAnalysisID != secondRun.ID {
		t.Fatalf("expected newer analysis id %q to remain current, got %q", secondRun.ID, saved.LatestAnalysisID)
	}
	if saved.LatestAnalysisStatus != domain.JobTargetAnalysisRunning {
		t.Fatalf("expected newer running status to remain current, got %s", saved.LatestAnalysisStatus)
	}
}

func TestFailedJobTargetAnalysisKeepsLatestSuccessfulSnapshotAttached(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	target, err := store.CreateJobTarget(ctx, domain.JobTargetInput{
		Title:       "后端工程师 - Example",
		CompanyName: "Example",
		SourceText:  "要求 Go、Redis、Kafka 经验。",
	})
	if err != nil {
		t.Fatalf("CreateJobTarget() error = %v", err)
	}

	firstRun, err := store.StartJobTargetAnalysis(ctx, target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() first error = %v", err)
	}
	if err := store.CompleteJobTargetAnalysis(ctx, target.ID, firstRun.ID, &domain.AnalyzeJobTargetResponse{
		Summary:          "偏高并发后端。",
		MustHaveSkills:   []string{"Go"},
		Responsibilities: []string{"负责核心服务设计"},
		EvaluationFocus:  []string{"缓存一致性"},
	}); err != nil {
		t.Fatalf("CompleteJobTargetAnalysis() error = %v", err)
	}

	secondRun, err := store.StartJobTargetAnalysis(ctx, target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() second error = %v", err)
	}
	if err := store.FailJobTargetAnalysis(ctx, target.ID, secondRun.ID, "boom"); err != nil {
		t.Fatalf("FailJobTargetAnalysis() error = %v", err)
	}

	saved, err := store.GetJobTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("GetJobTarget() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved job target")
	}
	if saved.LatestAnalysisStatus != domain.JobTargetAnalysisFailed {
		t.Fatalf("expected failed status, got %s", saved.LatestAnalysisStatus)
	}
	if saved.LatestSuccessfulAnalysis == nil {
		t.Fatal("expected latest successful analysis to remain attached after failed rerun")
	}
	if saved.LatestSuccessfulAnalysis.ID != firstRun.ID {
		t.Fatalf("expected latest successful analysis id %q, got %q", firstRun.ID, saved.LatestSuccessfulAnalysis.ID)
	}
}

func TestGetReviewIncludesBoundJobTargetRef(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	target, err := store.CreateJobTarget(ctx, domain.JobTargetInput{
		Title:       "后端工程师 - Example",
		CompanyName: "Example",
		SourceText:  "要求 Go、Redis、Kafka 经验。",
	})
	if err != nil {
		t.Fatalf("CreateJobTarget() error = %v", err)
	}
	run, err := store.StartJobTargetAnalysis(ctx, target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() error = %v", err)
	}
	if err := store.CompleteJobTargetAnalysis(ctx, target.ID, run.ID, &domain.AnalyzeJobTargetResponse{
		Summary:          "偏高并发后端。",
		MustHaveSkills:   []string{"Go"},
		Responsibilities: []string{"负责核心服务设计"},
		EvaluationFocus:  []string{"并发设计取舍"},
	}); err != nil {
		t.Fatalf("CompleteJobTargetAnalysis() error = %v", err)
	}

	session := &domain.TrainingSession{
		ID:                  "sess_review_bound_target",
		Mode:                domain.ModeBasics,
		Topic:               "go",
		JobTargetID:         target.ID,
		JobTargetAnalysisID: run.ID,
		Intensity:           "standard",
		Status:              domain.StatusReviewPending,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_review_bound_target",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 goroutine 为什么轻量？",
		ExpectedPoints: []string{"调度", "栈"},
		Answer:         "因为调度和栈更轻。",
	}
	if err := store.CreateSession(ctx, session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	review := &domain.ReviewCard{
		ID:                "review_bound_target",
		SessionID:         session.ID,
		Overall:           "整体还行。",
		Highlights:        []string{"先给结论"},
		Gaps:              []string{"缺取舍"},
		SuggestedTopics:   []string{"go"},
		NextTrainingFocus: []string{"补 trade-off"},
		ScoreBreakdown:    map[string]float64{"准确性": 80},
	}
	if err := store.CreateReview(ctx, review); err != nil {
		t.Fatalf("CreateReview() error = %v", err)
	}

	saved, err := store.GetReview(ctx, review.ID)
	if err != nil {
		t.Fatalf("GetReview() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved review")
	}
	if saved.JobTarget == nil {
		t.Fatal("expected bound job target ref on review")
	}
	if saved.JobTarget.Title != target.Title {
		t.Fatalf("expected job target title %q, got %q", target.Title, saved.JobTarget.Title)
	}
}

func TestSetAndClearActiveJobTargetPersistsOnProfile(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	target, err := store.CreateJobTarget(ctx, domain.JobTargetInput{
		Title:       "后端工程师 - Example",
		CompanyName: "Example",
		SourceText:  "要求 Go、Redis、Kafka 经验。",
	})
	if err != nil {
		t.Fatalf("CreateJobTarget() error = %v", err)
	}

	profile, err := store.SetActiveJobTarget(ctx, target.ID)
	if err != nil {
		t.Fatalf("SetActiveJobTarget() error = %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile after setting active job target")
	}
	if profile.ActiveJobTargetID != target.ID {
		t.Fatalf("expected active job target id %q, got %q", target.ID, profile.ActiveJobTargetID)
	}
	if profile.ActiveJobTarget == nil || profile.ActiveJobTarget.Title != target.Title {
		t.Fatal("expected active job target ref to be hydrated")
	}

	profile, err = store.ClearActiveJobTarget(ctx)
	if err != nil {
		t.Fatalf("ClearActiveJobTarget() error = %v", err)
	}
	if profile == nil {
		t.Fatal("expected profile after clearing active job target")
	}
	if profile.ActiveJobTargetID != "" {
		t.Fatalf("expected active job target id to be cleared, got %q", profile.ActiveJobTargetID)
	}
	if profile.ActiveJobTarget != nil {
		t.Fatal("expected active job target ref to be cleared")
	}
}

func openTestStore(t *testing.T) (*Store, error) {
	t.Helper()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "practicehelper.db"))
	if err != nil {
		return nil, err
	}

	if err := sqlite.Bootstrap(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return New(db), nil
}
