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
