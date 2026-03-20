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
		{name: "expression", kind: "expression", want: "围绕「x」做表达方式专项"},
		{name: "followup", kind: "followup_breakdown", want: "围绕「x」做追问抗压专项"},
		{name: "depth", kind: "depth", want: "围绕「x」做展开深挖专项"},
		{name: "detail", kind: "detail", want: "围绕「x」做细节补强专项"},
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

func TestSubmitAnswerStreamEmitsStatusSequenceForMainAnswer(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_stream_main",
		Mode:      domain.ModeBasics,
		Topic:     "go",
		Intensity: "standard",
		Status:    domain.StatusWaitingAnswer,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_stream_main",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 channel 和 mutex 什么时候各用什么？",
		ExpectedPoints: []string{"共享内存", "所有权转移"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/evaluate_answer/stream" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"}); err != nil {
			t.Fatalf("encode phase event: %v", err)
		}
		if err := encoder.Encode(domain.StreamEvent{
			Type: "result",
			Data: domain.EvaluationResult{
				Score:            81,
				ScoreBreakdown:   map[string]float64{"准确性": 80},
				Strengths:        []string{"有结论"},
				Gaps:             []string{"例子不够具体"},
				FollowupQuestion: "那你在项目里什么时候会避免过度用 channel？",
				FollowupPoints:   []string{"性能", "复杂度"},
			},
		}); err != nil {
			t.Fatalf("encode result event: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))

	var events []domain.StreamEvent
	updated, err := svc.SubmitAnswerStream(
		context.Background(),
		session.ID,
		domain.SubmitAnswerRequest{Answer: "channel 更适合消息和所有权转移，mutex 更适合保护共享状态。"},
		func(event domain.StreamEvent) error {
			events = append(events, event)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("SubmitAnswerStream() error = %v", err)
	}

	if updated.Status != domain.StatusFollowup {
		t.Fatalf("expected followup status, got %s", updated.Status)
	}

	assertStatusEventNames(t, events, []string{
		"answer_received",
		"evaluation_started",
		"feedback_ready",
		"answer_saved",
		"followup_ready",
	})
}

func TestSubmitAnswerStreamEmitsStatusSequenceForFollowupAnswer(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:         "sess_stream_followup",
		Mode:       domain.ModeBasics,
		Topic:      "go",
		Intensity:  "standard",
		Status:     domain.StatusFollowup,
		TotalScore: 72,
	}
	turn := &domain.TrainingTurn{
		ID:                    "turn_stream_followup",
		SessionID:             session.ID,
		TurnIndex:             1,
		Stage:                 "question",
		Question:              "Go 的 channel 和 mutex 什么时候各用什么？",
		ExpectedPoints:        []string{"共享内存", "所有权转移"},
		Answer:                "主回答已经提交",
		FollowupQuestion:      "那你在项目里什么时候会避免过度用 channel？",
		FollowupExpectedPoint: []string{"性能", "复杂度"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)

		switch r.URL.Path {
		case "/internal/evaluate_answer/stream":
			if err := encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"}); err != nil {
				t.Fatalf("encode phase event: %v", err)
			}
			if err := encoder.Encode(domain.StreamEvent{
				Type: "result",
				Data: domain.EvaluationResult{
					Score:          86,
					ScoreBreakdown: map[string]float64{"准确性": 85},
					Strengths:      []string{"解释到了取舍"},
					Gaps:           []string{"可以补线上场景"},
				},
			}); err != nil {
				t.Fatalf("encode evaluation result event: %v", err)
			}
		case "/internal/generate_review/stream":
			if err := encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"}); err != nil {
				t.Fatalf("encode review phase event: %v", err)
			}
			if err := encoder.Encode(domain.StreamEvent{
				Type: "result",
				Data: domain.ReviewCard{
					Overall:           "本轮回答整体过线，但例子还能更贴近真实线上场景。",
					Highlights:        []string{"主线判断清楚"},
					Gaps:              []string{"例子不够落地"},
					SuggestedTopics:   []string{"并发控制"},
					NextTrainingFocus: []string{"把取舍讲得更具体"},
					ScoreBreakdown:    map[string]float64{"准确性": 84},
				},
			}); err != nil {
				t.Fatalf("encode review result event: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))

	var events []domain.StreamEvent
	updated, err := svc.SubmitAnswerStream(
		context.Background(),
		session.ID,
		domain.SubmitAnswerRequest{Answer: "我会在共享状态简单且性能敏感时优先 mutex，而不是为了 channel 而 channel。"},
		func(event domain.StreamEvent) error {
			events = append(events, event)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("SubmitAnswerStream() error = %v", err)
	}

	if updated.Status != domain.StatusCompleted {
		t.Fatalf("expected completed status, got %s", updated.Status)
	}
	if updated.ReviewID == "" {
		t.Fatal("expected review id to be saved")
	}

	assertStatusEventNames(t, events, []string{
		"answer_received",
		"evaluation_started",
		"feedback_ready",
		"answer_saved",
		"review_started",
		"review_saved",
	})
}

func assertStatusEventNames(t *testing.T, events []domain.StreamEvent, want []string) {
	t.Helper()

	var got []string
	for _, event := range events {
		if event.Type == "status" {
			got = append(got, event.Name)
		}
	}

	if len(got) != len(want) {
		t.Fatalf("unexpected status event count: got %v want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected status event sequence: got %v want %v", got, want)
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
