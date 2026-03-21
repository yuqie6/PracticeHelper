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

	for _, topic := range []string{
		domain.BasicsTopicGo,
		domain.BasicsTopicRedis,
		domain.BasicsTopicKafka,
		domain.BasicsTopicMySQL,
		domain.BasicsTopicSystemDesign,
		domain.BasicsTopicDistributed,
		domain.BasicsTopicNetwork,
		domain.BasicsTopicMicroservice,
		domain.BasicsTopicOS,
		domain.BasicsTopicDockerK8s,
	} {
		if counts[topic] < 5 {
			t.Fatalf("expected at least 5 basics templates for %s, got %d", topic, counts[topic])
		}
	}
}

func TestListQuestionTemplatesByTopicsReturnsTemplatesAcrossSelectedTopics(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	templates, err := store.ListQuestionTemplatesByTopics(context.Background(), []string{
		domain.BasicsTopicRedis,
		domain.BasicsTopicOS,
	})
	if err != nil {
		t.Fatalf("ListQuestionTemplatesByTopics() error = %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("expected templates for selected topics")
	}

	seen := map[string]bool{}
	for _, item := range templates {
		seen[item.Topic] = true
	}
	if !seen[domain.BasicsTopicRedis] || !seen[domain.BasicsTopicOS] {
		t.Fatalf("expected redis and os templates, got topics=%v", seen)
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

func TestClaimProjectImportJobPreventsConcurrentClaims(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	job, err := store.CreateProjectImportJob(ctx, "https://github.com/octocat/claim-once")
	if err != nil {
		t.Fatalf("CreateProjectImportJob() error = %v", err)
	}

	startedAt := time.Now().UTC()
	claimed, err := store.ClaimProjectImportJob(
		ctx,
		job.ID,
		"claim_token_a",
		domain.ProjectImportStageAnalyzing,
		"running",
		startedAt,
		startedAt.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("ClaimProjectImportJob() first claim error = %v", err)
	}
	if !claimed {
		t.Fatal("expected first claim to succeed")
	}

	claimed, err = store.ClaimProjectImportJob(
		ctx,
		job.ID,
		"claim_token_b",
		domain.ProjectImportStageAnalyzing,
		"running again",
		startedAt,
		startedAt.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("ClaimProjectImportJob() second claim error = %v", err)
	}
	if claimed {
		t.Fatal("expected second claim to be rejected")
	}

	finished, err := store.FinishClaimedProjectImportJob(
		ctx,
		job.ID,
		"claim_token_b",
		domain.ProjectImportStatusCompleted,
		domain.ProjectImportStageCompleted,
		"done",
		"",
		"proj_x",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("FinishClaimedProjectImportJob() wrong token error = %v", err)
	}
	if finished {
		t.Fatal("expected finish with wrong token to fail")
	}

	finished, err = store.FinishClaimedProjectImportJob(
		ctx,
		job.ID,
		"claim_token_a",
		domain.ProjectImportStatusCompleted,
		domain.ProjectImportStageCompleted,
		"done",
		"",
		"proj_x",
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatalf("FinishClaimedProjectImportJob() owner token error = %v", err)
	}
	if !finished {
		t.Fatal("expected finish with owner token to succeed")
	}

	var (
		status         string
		claimToken     string
		claimExpiresAt string
	)
	if err := store.db.QueryRow(`
		SELECT status, claim_token, claim_expires_at
		FROM project_import_jobs
		WHERE id = ?
	`, job.ID).Scan(&status, &claimToken, &claimExpiresAt); err != nil {
		t.Fatalf("query claimed job: %v", err)
	}

	if status != domain.ProjectImportStatusCompleted {
		t.Fatalf("expected completed status, got %s", status)
	}
	if claimToken != "" || claimExpiresAt != "" {
		t.Fatalf("expected claim fields to be cleared, got token=%q expires=%q", claimToken, claimExpiresAt)
	}
}

func TestClaimProjectImportJobReclaimsExpiredRunningJob(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	job, err := store.CreateProjectImportJob(ctx, "https://github.com/octocat/reclaim-expired")
	if err != nil {
		t.Fatalf("CreateProjectImportJob() error = %v", err)
	}

	expiredAt := time.Now().UTC().Add(-time.Minute)
	if _, err := store.db.ExecContext(ctx, `
		UPDATE project_import_jobs
		SET status = ?, stage = ?, message = ?, claim_token = ?, claim_expires_at = ?,
			started_at = ?, updated_at = ?
		WHERE id = ?
	`,
		domain.ProjectImportStatusRunning,
		domain.ProjectImportStageAnalyzing,
		"stale worker",
		"stale_claim",
		expiredAt.Format(time.RFC3339Nano),
		expiredAt.Add(-time.Minute).Format(time.RFC3339Nano),
		expiredAt.Format(time.RFC3339Nano),
		job.ID,
	); err != nil {
		t.Fatalf("seed stale claimed job: %v", err)
	}

	claimed, err := store.ClaimProjectImportJob(
		ctx,
		job.ID,
		"fresh_claim",
		domain.ProjectImportStageAnalyzing,
		"reclaimed",
		time.Now().UTC(),
		time.Now().UTC().Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("ClaimProjectImportJob() reclaim error = %v", err)
	}
	if !claimed {
		t.Fatal("expected expired running job to be reclaimed")
	}

	var claimToken string
	if err := store.db.QueryRow(`
		SELECT claim_token
		FROM project_import_jobs
		WHERE id = ?
	`, job.ID).Scan(&claimToken); err != nil {
		t.Fatalf("query reclaimed token: %v", err)
	}
	if claimToken != "fresh_claim" {
		t.Fatalf("expected fresh claim token, got %q", claimToken)
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

	var snapshotCount int
	if err := store.db.QueryRow(`
		SELECT COUNT(*)
		FROM weakness_snapshots
		WHERE weakness_id = 'weak_match_target'
	`).Scan(&snapshotCount); err != nil {
		t.Fatalf("count relieved snapshots: %v", err)
	}
	if snapshotCount != 1 {
		t.Fatalf("expected 1 relieved snapshot for matched weakness, got %d", snapshotCount)
	}
}

func TestRelieveWeaknessRecordsSnapshot(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := store.db.Exec(`
		INSERT INTO weakness_tags (id, kind, label, severity, frequency, last_seen_at, evidence_session_id)
		VALUES ('weak_relieve_target', 'topic', 'Redis缓存击穿', 1.20, 2, ?, 'sess_old')
	`, now); err != nil {
		t.Fatalf("insert weakness tag: %v", err)
	}

	if err := store.RelieveWeakness(context.Background(), "topic", "Redis缓存击穿", 0.30); err != nil {
		t.Fatalf("RelieveWeakness() error = %v", err)
	}

	var severity float64
	if err := store.db.QueryRow(`
		SELECT severity
		FROM weakness_tags
		WHERE id = 'weak_relieve_target'
	`).Scan(&severity); err != nil {
		t.Fatalf("query relieved severity: %v", err)
	}
	if severity >= 1.20 {
		t.Fatalf("expected relieved severity to decrease, got %.2f", severity)
	}

	var snapshotSeverity float64
	var sessionID string
	if err := store.db.QueryRow(`
		SELECT severity, session_id
		FROM weakness_snapshots
		WHERE weakness_id = 'weak_relieve_target'
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&snapshotSeverity, &sessionID); err != nil {
		t.Fatalf("query relieved snapshot: %v", err)
	}
	if snapshotSeverity != severity {
		t.Fatalf("expected snapshot severity %.2f, got %.2f", severity, snapshotSeverity)
	}
	if sessionID != "" {
		t.Fatalf("expected relieved snapshot session id to be empty, got %q", sessionID)
	}
}

func TestGetWeaknessTrendsReturnsLatestFiftyPoints(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	if _, err := store.db.Exec(`
		INSERT INTO weakness_tags (id, kind, label, severity, frequency, last_seen_at, evidence_session_id)
		VALUES ('weak_trend_latest', 'topic', '缓存一致性', 1.40, 60, ?, 'sess_latest')
	`, base.Add(59*time.Minute).Format(time.RFC3339Nano)); err != nil {
		t.Fatalf("insert weakness tag: %v", err)
	}

	for i := 0; i < 60; i++ {
		if _, err := store.db.Exec(`
			INSERT INTO weakness_snapshots (weakness_id, session_id, severity, created_at)
			VALUES (?, ?, ?, ?)
		`,
			"weak_trend_latest",
			"",
			float64(i+1),
			base.Add(time.Duration(i)*time.Minute).Format(time.RFC3339Nano),
		); err != nil {
			t.Fatalf("insert weakness snapshot %d: %v", i, err)
		}
	}

	trends, err := store.GetWeaknessTrends(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetWeaknessTrends() error = %v", err)
	}
	if len(trends) != 1 {
		t.Fatalf("expected 1 trend, got %d", len(trends))
	}
	if len(trends[0].Points) != 50 {
		t.Fatalf("expected 50 points, got %d", len(trends[0].Points))
	}
	first := trends[0].Points[0]
	last := trends[0].Points[len(trends[0].Points)-1]
	if first.Severity != 11 {
		t.Fatalf("expected first recent point severity 11, got %.2f", first.Severity)
	}
	if last.Severity != 60 {
		t.Fatalf("expected last recent point severity 60, got %.2f", last.Severity)
	}
	if first.CreatedAt != base.Add(10*time.Minute).Format(time.RFC3339Nano) {
		t.Fatalf("expected first recent point timestamp %q, got %q", base.Add(10*time.Minute).Format(time.RFC3339Nano), first.CreatedAt)
	}
	if last.CreatedAt != base.Add(59*time.Minute).Format(time.RFC3339Nano) {
		t.Fatalf("expected last recent point timestamp %q, got %q", base.Add(59*time.Minute).Format(time.RFC3339Nano), last.CreatedAt)
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

func TestCreateReviewScheduleUpsertsExistingSessionEntry(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	first := &domain.ReviewScheduleItem{
		SessionID:     "sess_review_schedule",
		ReviewCardID:  "review_1",
		WeaknessTagID: "weak_cache_consistency",
		Topic:         "redis",
		NextReviewAt:  time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC),
		IntervalDays:  1,
		EaseFactor:    2.5,
	}
	if _, err := store.db.Exec(`
		INSERT INTO weakness_tags (id, kind, label, severity, frequency, last_seen_at, evidence_session_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, first.WeaknessTagID, "topic", "缓存一致性", 1.10, 3, nowUTC(), first.SessionID); err != nil {
		t.Fatalf("seed weakness tag: %v", err)
	}
	if err := store.CreateReviewSchedule(ctx, first); err != nil {
		t.Fatalf("CreateReviewSchedule() first error = %v", err)
	}

	second := &domain.ReviewScheduleItem{
		SessionID:     first.SessionID,
		ReviewCardID:  "review_2",
		WeaknessTagID: first.WeaknessTagID,
		Topic:         "kafka",
		NextReviewAt:  time.Date(2026, 3, 24, 10, 0, 0, 0, time.UTC),
		IntervalDays:  3,
		EaseFactor:    2.6,
	}
	if err := store.CreateReviewSchedule(ctx, second); err != nil {
		t.Fatalf("CreateReviewSchedule() second error = %v", err)
	}

	var count int
	if err := store.db.QueryRow(`
		SELECT COUNT(*)
		FROM review_schedule
		WHERE weakness_tag_id = ?
	`, first.WeaknessTagID).Scan(&count); err != nil {
		t.Fatalf("query review schedule count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 review schedule row after upsert, got %d", count)
	}

	items, err := store.ListDueReviews(ctx, second.NextReviewAt.Add(time.Second))
	if err != nil {
		t.Fatalf("ListDueReviews() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 due review item, got %d", len(items))
	}
	if items[0].ReviewCardID != second.ReviewCardID {
		t.Fatalf("expected updated review card id %q, got %q", second.ReviewCardID, items[0].ReviewCardID)
	}
	if items[0].Topic != second.Topic {
		t.Fatalf("expected updated topic %q, got %q", second.Topic, items[0].Topic)
	}
	if items[0].WeaknessLabel != "缓存一致性" {
		t.Fatalf("expected weakness label 缓存一致性, got %q", items[0].WeaknessLabel)
	}
	if items[0].IntervalDays != second.IntervalDays {
		t.Fatalf("expected updated interval %d, got %d", second.IntervalDays, items[0].IntervalDays)
	}
}

func TestCreateEvaluationLogPersistsEntry(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.CreateEvaluationLog(
		context.Background(),
		"sess_eval_log",
		"turn_eval_log",
		"evaluate_answer",
		"gpt-test",
		"stable-v1",
		"hash-evaluate-answer",
		"{\"score\":82}",
		12.5,
	); err != nil {
		t.Fatalf("CreateEvaluationLog() error = %v", err)
	}

	var (
		sessionID   string
		turnID      string
		flowName    string
		modelName   string
		promptSetID string
		promptHash  string
		rawOutput   string
		latencyMs   float64
	)
	if err := store.db.QueryRow(`
		SELECT session_id, turn_id, flow_name, model_name, prompt_set_id, prompt_hash, raw_output, latency_ms
		FROM evaluation_logs
		ORDER BY id DESC
		LIMIT 1
	`).Scan(
		&sessionID,
		&turnID,
		&flowName,
		&modelName,
		&promptSetID,
		&promptHash,
		&rawOutput,
		&latencyMs,
	); err != nil {
		t.Fatalf("query evaluation log: %v", err)
	}

	if sessionID != "sess_eval_log" || turnID != "turn_eval_log" {
		t.Fatalf("unexpected evaluation log ids: got session=%q turn=%q", sessionID, turnID)
	}
	if flowName != "evaluate_answer" {
		t.Fatalf("expected flow name evaluate_answer, got %q", flowName)
	}
	if modelName != "gpt-test" {
		t.Fatalf("expected model name gpt-test, got %q", modelName)
	}
	if promptSetID != "stable-v1" {
		t.Fatalf("expected prompt set id stable-v1, got %q", promptSetID)
	}
	if promptHash != "hash-evaluate-answer" {
		t.Fatalf("expected prompt hash hash-evaluate-answer, got %q", promptHash)
	}
	if rawOutput != "{\"score\":82}" {
		t.Fatalf("expected raw output to persist, got %q", rawOutput)
	}
	if latencyMs != 12.5 {
		t.Fatalf("expected latency 12.5, got %.2f", latencyMs)
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
