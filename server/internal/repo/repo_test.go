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
		domain.StatusFollowup,
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
		FollowupAnswer: "因为上下文切换成本更低",
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
