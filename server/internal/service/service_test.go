package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/infra/sqlite"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/sidecar"
)

func TestBuildTodayFocusUsesReadableWeaknessLabels(t *testing.T) {
	profile := &domain.UserProfile{}
	weaknesses := []domain.WeaknessTag{
		{
			Kind:  "depth",
			Label: "回答缺少因果展开",
		},
	}

	focus := buildTodayFocus(profile, weaknesses)
	want := "今天优先补 展开深度：回答缺少因果展开"
	if focus != want {
		t.Fatalf("unexpected today focus: got %q want %q", focus, want)
	}
}

func TestBuildRecommendedTrackCoversNewWeaknessKinds(t *testing.T) {
	cases := []struct {
		name string
		kind string
		want string
	}{
		{name: "expression", kind: "expression", want: "表达方式专项"},
		{name: "followup", kind: "followup_breakdown", want: "追问抗压专项"},
		{name: "depth", kind: "depth", want: "展开深挖专项"},
		{name: "detail", kind: "detail", want: "细节补强专项"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildRecommendedTrack(nil, []domain.WeaknessTag{{Kind: tc.kind, Label: "x"}})
			if got != tc.want {
				t.Fatalf("unexpected recommended track: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestSubmitAnswerRejectsBusySessionBeforeCallingSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_busy",
		Mode:      domain.ModeBasics,
		Topic:     "go",
		Intensity: "standard",
		Status:    domain.StatusEvaluating,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_busy",
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
	_, err = svc.SubmitAnswer(context.Background(), session.ID, domain.SubmitAnswerRequest{Answer: "answer"})
	if err == nil {
		t.Fatal("expected SubmitAnswer() to fail")
	}
	if err != ErrSessionBusy {
		t.Fatalf("expected ErrSessionBusy, got %v", err)
	}
}

func TestSubmitAnswerRejectsReviewPendingSession(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_review_pending",
		Mode:      domain.ModeBasics,
		Topic:     "go",
		Intensity: "standard",
		Status:    domain.StatusReviewPending,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_review_pending",
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
	_, err = svc.SubmitAnswer(context.Background(), session.ID, domain.SubmitAnswerRequest{Answer: "answer"})
	if err == nil {
		t.Fatal("expected SubmitAnswer() to fail")
	}
	if err != ErrSessionReviewPending {
		t.Fatalf("expected ErrSessionReviewPending, got %v", err)
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
		FollowupAnswer: "因为开销更低",
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

func TestSubmitAnswerPassesTemplateScoreWeightsToSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_score_weights",
		Mode:      domain.ModeBasics,
		Topic:     "redis",
		Intensity: "standard",
		Status:    domain.StatusWaitingAnswer,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_score_weights",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环", "高效数据结构"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	var captured domain.EvaluateAnswerRequest
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/evaluate_answer" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.EvaluationResult{
			Score:            82,
			ScoreBreakdown:   map[string]float64{"准确性": 80},
			Strengths:        []string{"覆盖了核心点"},
			Gaps:             []string{"可以补更多线上例子"},
			FollowupQuestion: "Redis 6 的多线程多在哪里？",
			FollowupPoints:   []string{"网络读写线程", "命令执行仍是主线程"},
			WeaknessHits:     []domain.WeaknessHit{{Kind: "detail", Label: "redis", Severity: 0.4}},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	_, err = svc.SubmitAnswer(context.Background(), session.ID, domain.SubmitAnswerRequest{
		Answer: "Redis 快主要因为数据在内存、事件循环和数据结构效率。",
	})
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}

	want := map[string]float64{
		"准确性":   30,
		"完整性":   25,
		"落地感":   15,
		"表达清晰度": 15,
		"抗追问能力": 15,
	}
	if len(captured.ScoreWeights) != len(want) {
		t.Fatalf("unexpected score weights length: got %d want %d", len(captured.ScoreWeights), len(want))
	}
	for key, value := range want {
		if captured.ScoreWeights[key] != value {
			t.Fatalf("unexpected score weight for %s: got %v want %v", key, captured.ScoreWeights[key], value)
		}
	}
}

func openTestStore(t *testing.T) (*repo.Store, error) {
	t.Helper()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "practicehelper.db"))
	if err != nil {
		return nil, err
	}

	if err := sqlite.Bootstrap(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo.New(db), nil
}
