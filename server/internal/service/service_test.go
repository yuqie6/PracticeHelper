package service

import (
	"context"
	"path/filepath"
	"testing"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/infra/sqlite"
	"practicehelper/server/internal/repo"
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
