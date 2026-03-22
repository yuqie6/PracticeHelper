package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/sidecar"
)

func TestCompleteDueReviewAdvancesScheduleUsingSessionScore(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:         "sess_due_review",
		Mode:       domain.ModeBasics,
		Topic:      "redis",
		Intensity:  "standard",
		Status:     domain.StatusCompleted,
		TotalScore: 88,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_due_review",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环"},
		Answer:         "因为数据在内存里。",
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := store.UpsertWeaknesses(context.Background(), session.ID, []domain.WeaknessHit{
		{Kind: "topic", Label: "缓存一致性", Severity: 1.0},
	}); err != nil {
		t.Fatalf("UpsertWeaknesses() error = %v", err)
	}
	tag, err := store.GetWeaknessTag(context.Background(), "topic", "缓存一致性")
	if err != nil {
		t.Fatalf("GetWeaknessTag() error = %v", err)
	}
	if tag == nil {
		t.Fatal("expected weakness tag to exist")
	}
	if err := store.CreateReviewSchedule(context.Background(), &domain.ReviewScheduleItem{
		SessionID:     session.ID,
		ReviewCardID:  "review_due_review",
		WeaknessTagID: tag.ID,
		Topic:         session.Topic,
		NextReviewAt:  time.Now().UTC().Add(-time.Hour),
		IntervalDays:  1,
		EaseFactor:    2.5,
	}); err != nil {
		t.Fatalf("CreateReviewSchedule() error = %v", err)
	}

	items, err := store.ListDueReviews(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatalf("ListDueReviews() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 due review item, got %d", len(items))
	}
	if items[0].WeaknessLabel != "缓存一致性" {
		t.Fatalf("expected weakness label 缓存一致性, got %q", items[0].WeaknessLabel)
	}

	svc := New(store, nil)
	if err := svc.CompleteDueReview(context.Background(), items[0].ID); err != nil {
		t.Fatalf("CompleteDueReview() error = %v", err)
	}

	saved, err := store.GetReviewSchedule(context.Background(), items[0].ID)
	if err != nil {
		t.Fatalf("GetReviewSchedule() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected updated review schedule")
	}
	if saved.IntervalDays != 3 {
		t.Fatalf("expected interval to advance to 3 days, got %d", saved.IntervalDays)
	}
	if saved.EaseFactor <= 2.5 {
		t.Fatalf("expected ease factor to increase, got %.2f", saved.EaseFactor)
	}
	if !saved.NextReviewAt.After(time.Now().UTC().Add(48 * time.Hour)) {
		t.Fatalf("expected next review to move into future, got %s", saved.NextReviewAt.Format(time.RFC3339))
	}

	remaining, err := store.ListDueReviews(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatalf("ListDueReviews() after complete error = %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected no due reviews after completion, got %d", len(remaining))
	}
}

func TestBuildReviewScheduleItemsUsesWeaknessLevelTopics(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.UpsertWeaknesses(ctx, "sess_schedule_seed", []domain.WeaknessHit{
		{Kind: "detail", Label: "Redis 缓存一致性", Severity: 1.1},
		{Kind: "followup_breakdown", Label: "进程线程调度容易乱", Severity: 0.9},
	}); err != nil {
		t.Fatalf("UpsertWeaknesses() error = %v", err)
	}

	svc := New(store, nil)
	nextReview := time.Date(2026, 3, 25, 9, 0, 0, 0, time.UTC)
	items := svc.buildReviewScheduleItems(ctx, &domain.TrainingSession{
		ID:    "sess_schedule_seed",
		Mode:  domain.ModeBasics,
		Topic: domain.BasicsTopicMixed,
		Turns: []domain.TrainingTurn{
			{
				ID:           "turn_1",
				SessionID:    "sess_schedule_seed",
				TurnIndex:    1,
				WeaknessHits: []domain.WeaknessHit{{Kind: "detail", Label: "Redis 缓存一致性", Severity: 1.1}},
			},
			{
				ID:           "turn_2",
				SessionID:    "sess_schedule_seed",
				TurnIndex:    2,
				WeaknessHits: []domain.WeaknessHit{{Kind: "followup_breakdown", Label: "进程线程调度容易乱", Severity: 0.9}},
			},
		},
	}, &domain.ReviewCard{
		ID:              "review_schedule_seed",
		SuggestedTopics: []string{domain.BasicsTopicKafka},
	}, nextReview)

	if len(items) != 2 {
		t.Fatalf("expected 2 weakness-level review schedule items, got %d", len(items))
	}
	if items[0].WeaknessTagID == "" || items[1].WeaknessTagID == "" {
		t.Fatalf("expected weakness tag ids to be resolved, got %+v", items)
	}
	if items[0].WeaknessLabel == "" || items[1].WeaknessLabel == "" {
		t.Fatalf("expected weakness labels to be carried to schedule items, got %+v", items)
	}

	topics := []string{items[0].Topic, items[1].Topic}
	if !slices.Contains(topics, domain.BasicsTopicRedis) {
		t.Fatalf("expected redis review topic, got %v", topics)
	}
	if !slices.Contains(topics, domain.BasicsTopicOS) {
		t.Fatalf("expected os review topic, got %v", topics)
	}
}

func TestBuildReviewScheduleItemsDeduplicatesAndSortsBySeverity(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	if err := store.UpsertWeaknesses(ctx, "sess_schedule_dedupe", []domain.WeaknessHit{
		{Kind: "detail", Label: "缓存一致性", Severity: 1.2},
		{Kind: "topic", Label: "Kafka", Severity: 0.7},
	}); err != nil {
		t.Fatalf("UpsertWeaknesses() error = %v", err)
	}

	svc := New(store, nil)
	items := svc.buildReviewScheduleItems(ctx, &domain.TrainingSession{
		ID:    "sess_schedule_dedupe",
		Mode:  domain.ModeBasics,
		Topic: domain.BasicsTopicMixed,
		Turns: []domain.TrainingTurn{
			{
				ID:        "turn_schedule_dedupe_1",
				SessionID: "sess_schedule_dedupe",
				TurnIndex: 1,
				WeaknessHits: []domain.WeaknessHit{
					{Kind: "detail", Label: "缓存一致性", Severity: 0.6},
					{Kind: "topic", Label: "Kafka", Severity: 0.7},
				},
			},
			{
				ID:        "turn_schedule_dedupe_2",
				SessionID: "sess_schedule_dedupe",
				TurnIndex: 2,
				WeaknessHits: []domain.WeaknessHit{
					{Kind: "detail", Label: "缓存一致性", Severity: 1.2},
				},
			},
		},
	}, &domain.ReviewCard{
		ID: "review_schedule_dedupe",
	}, time.Date(2026, 3, 25, 9, 0, 0, 0, time.UTC))

	if len(items) != 2 {
		t.Fatalf("expected 2 deduplicated items, got %d", len(items))
	}
	if items[0].WeaknessLabel != "缓存一致性" {
		t.Fatalf("expected highest-severity weakness first, got %+v", items)
	}
	if items[1].WeaknessLabel != "Kafka" {
		t.Fatalf("expected kafka weakness second, got %+v", items)
	}
}

func TestRetrySessionReviewRejectsBusySession(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_retry_busy",
		Mode:      domain.ModeBasics,
		Topic:     "go",
		Intensity: "standard",
		Status:    domain.StatusEvaluating,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_retry_busy",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 goroutine 为什么轻量？",
		ExpectedPoints: []string{"调度", "栈"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	_, err = svc.RetrySessionReview(context.Background(), session.ID)
	if err == nil {
		t.Fatal("expected RetrySessionReview() to fail")
	}
	if err != ErrSessionBusy {
		t.Fatalf("expected ErrSessionBusy, got %v", err)
	}
}

func TestRetrySessionReviewKeepsReviewPendingWhenGenerationFails(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_retry_failed_review",
		Mode:      domain.ModeBasics,
		Topic:     "go",
		Intensity: "standard",
		Status:    domain.StatusReviewPending,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_retry_failed_review",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 goroutine 为什么轻量？",
		ExpectedPoints: []string{"调度", "栈"},
		Answer:         "因为更轻量",
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, sidecar.New("http://127.0.0.1:1", 20*time.Millisecond))
	_, err = svc.RetrySessionReview(context.Background(), session.ID)
	if err == nil {
		t.Fatal("expected RetrySessionReview() to fail")
	}
	if !errors.Is(err, ErrReviewGenerationRetry) {
		t.Fatalf("expected ErrReviewGenerationRetry, got %v", err)
	}

	saved, err := store.GetSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved session")
	}
	if saved.Status != domain.StatusReviewPending {
		t.Fatalf("expected session to stay review_pending, got %s", saved.Status)
	}
}

func TestRetrySessionReviewRejectsCompleted(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_retry_completed",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicGo,
		Intensity: "standard",
		Status:    domain.StatusCompleted,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_retry_completed",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 goroutine 为什么轻量？",
		ExpectedPoints: []string{"调度", "栈"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	_, err = svc.RetrySessionReview(context.Background(), session.ID)
	if err == nil {
		t.Fatal("expected RetrySessionReview() to fail")
	}
	if err != ErrSessionCompleted {
		t.Fatalf("expected ErrSessionCompleted, got %v", err)
	}
}

func TestCompleteDueReviewReturnsNotFoundForMissingID(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	svc := New(store, nil)
	err = svc.CompleteDueReview(context.Background(), 999999)
	if !errors.Is(err, ErrReviewScheduleNotFound) {
		t.Fatalf("expected ErrReviewScheduleNotFound, got %v", err)
	}
}

func TestRetrySessionReviewPassesBoundJobTargetAnalysisToSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target := seedReadyJobTarget(t, store)

	session := &domain.TrainingSession{
		ID:                  "sess_review_job_target",
		Mode:                domain.ModeBasics,
		Topic:               "redis",
		JobTargetID:         target.ID,
		JobTargetAnalysisID: target.LatestAnalysisID,
		Intensity:           "standard",
		Status:              domain.StatusReviewPending,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_review_job_target",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环"},
		Answer:         "因为数据在内存，单线程模型减少锁竞争。",
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	var captured domain.GenerateReviewRequest
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/generate_review" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.ReviewCard{
			Overall:           "整体过线，但岗位要求的缓存一致性表达还不够硬。",
			Highlights:        []string{"主线完整"},
			Gaps:              []string{"缺缓存一致性取舍"},
			SuggestedTopics:   []string{"redis"},
			NextTrainingFocus: []string{"补缓存一致性表达"},
			ScoreBreakdown:    map[string]float64{"准确性": 84},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	updated, err := svc.RetrySessionReview(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("RetrySessionReview() error = %v", err)
	}

	if updated.Status != domain.StatusCompleted {
		t.Fatalf("expected completed status, got %s", updated.Status)
	}
	if captured.JobTargetAnalysis == nil {
		t.Fatal("expected generate review request to include job target analysis")
	}
	if len(captured.JobTargetAnalysis.MustHaveSkills) == 0 {
		t.Fatal("expected must-have skills to be forwarded to review generation")
	}
}
