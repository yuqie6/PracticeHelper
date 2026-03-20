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

	focus := buildTodayFocus(profile, nil, weaknesses, "generic")
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
			got := buildRecommendedTrack(nil, nil, []domain.WeaknessTag{{Kind: tc.kind, Label: "x"}}, "generic")
			if got != tc.want {
				t.Fatalf("unexpected recommended track: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestWeightWeaknessesForActiveJobTargetPrioritizesMatchedLabels(t *testing.T) {
	weighted := weightWeaknessesForActiveJobTarget(
		[]domain.WeaknessTag{
			{Kind: "detail", Label: "回答缺少因果展开", Severity: 0.88, Frequency: 2},
			{Kind: "topic", Label: "缓存一致性", Severity: 0.75, Frequency: 1},
		},
		&domain.JobTarget{
			Title: "后端工程师",
			LatestSuccessfulAnalysis: &domain.JobTargetAnalysisRun{
				MustHaveSkills:   []string{"Redis", "缓存一致性"},
				Responsibilities: []string{"负责核心链路稳定性"},
				EvaluationFocus:  []string{"故障排查闭环"},
			},
		},
	)

	if len(weighted) != 2 {
		t.Fatalf("expected 2 weighted weaknesses, got %d", len(weighted))
	}
	if weighted[0].Label != "缓存一致性" {
		t.Fatalf("expected matched weakness to rank first, got %s", weighted[0].Label)
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

func TestCreateSessionPassesBoundJobTargetAnalysisToSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target := seedReadyJobTarget(t, store)

	var captured domain.GenerateQuestionRequest
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/generate_question" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.GenerateQuestionResponse{
			Question:       "这个岗位要求缓存一致性时，你会怎么解释 Redis 写路径的取舍？",
			ExpectedPoints: []string{"一致性目标", "失败兜底", "性能取舍"},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	session, err := svc.CreateSession(context.Background(), domain.CreateSessionRequest{
		Mode:        domain.ModeBasics,
		Topic:       "redis",
		Intensity:   "standard",
		JobTargetID: target.ID,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session.JobTargetID != target.ID {
		t.Fatalf("expected session job target id %q, got %q", target.ID, session.JobTargetID)
	}
	if session.JobTargetAnalysisID == "" {
		t.Fatal("expected session to persist a job target analysis id")
	}
	if captured.JobTargetAnalysis == nil {
		t.Fatal("expected generate question request to include job target analysis")
	}
	if len(captured.JobTargetAnalysis.MustHaveSkills) == 0 {
		t.Fatal("expected must-have skills to be forwarded to sidecar")
	}
}

func TestCreateSessionFallsBackToActiveJobTargetWhenRequestOmitsOne(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target := seedReadyJobTarget(t, store)
	if _, err := store.SetActiveJobTarget(context.Background(), target.ID); err != nil {
		t.Fatalf("SetActiveJobTarget() error = %v", err)
	}

	var captured domain.GenerateQuestionRequest
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/generate_question" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.GenerateQuestionResponse{
			Question:       "这个岗位要求缓存一致性时，你会怎么解释 Redis 写路径的取舍？",
			ExpectedPoints: []string{"一致性目标", "失败兜底", "性能取舍"},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	session, err := svc.CreateSession(context.Background(), domain.CreateSessionRequest{
		Mode:      domain.ModeBasics,
		Topic:     "redis",
		Intensity: "standard",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session.JobTargetID != target.ID {
		t.Fatalf("expected active job target %q, got %q", target.ID, session.JobTargetID)
	}
	if captured.JobTargetAnalysis == nil {
		t.Fatal("expected active job target analysis to be injected")
	}
}

func TestCreateSessionIgnoresUnavailableActiveJobTarget(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target, err := store.CreateJobTarget(context.Background(), domain.JobTargetInput{
		Title:       "后端工程师 - Example",
		CompanyName: "Example",
		SourceText:  "要求 Go、Redis、Kafka 经验。",
	})
	if err != nil {
		t.Fatalf("CreateJobTarget() error = %v", err)
	}
	if _, err := store.SetActiveJobTarget(context.Background(), target.ID); err != nil {
		t.Fatalf("SetActiveJobTarget() error = %v", err)
	}

	var captured domain.GenerateQuestionRequest
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.GenerateQuestionResponse{
			Question:       "Redis 为什么快？",
			ExpectedPoints: []string{"内存访问", "事件循环"},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	session, err := svc.CreateSession(context.Background(), domain.CreateSessionRequest{
		Mode:      domain.ModeBasics,
		Topic:     "redis",
		Intensity: "standard",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session.JobTargetID != "" {
		t.Fatalf("expected generic fallback when active job target is unavailable, got %q", session.JobTargetID)
	}
	if captured.JobTargetAnalysis != nil {
		t.Fatal("expected no job target analysis when active job target is unavailable")
	}
}

func TestCreateSessionCanExplicitlyIgnoreActiveJobTarget(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target := seedReadyJobTarget(t, store)
	if _, err := store.SetActiveJobTarget(context.Background(), target.ID); err != nil {
		t.Fatalf("SetActiveJobTarget() error = %v", err)
	}

	var captured domain.GenerateQuestionRequest
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.GenerateQuestionResponse{
			Question:       "Redis 为什么快？",
			ExpectedPoints: []string{"内存访问", "事件循环"},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	session, err := svc.CreateSession(context.Background(), domain.CreateSessionRequest{
		Mode:                  domain.ModeBasics,
		Topic:                 "redis",
		Intensity:             "standard",
		IgnoreActiveJobTarget: true,
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if session.JobTargetID != "" {
		t.Fatalf("expected explicit generic training to skip active job target, got %q", session.JobTargetID)
	}
	if captured.JobTargetAnalysis != nil {
		t.Fatal("expected no job target analysis when active job target is explicitly ignored")
	}
}

func TestCreateSessionRejectsExplicitUnavailableJobTarget(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target := seedReadyJobTarget(t, store)
	updated, err := store.UpdateJobTarget(context.Background(), target.ID, domain.JobTargetInput{
		Title:       target.Title,
		CompanyName: target.CompanyName,
		SourceText:  target.SourceText + "\n新增：强调稳定性值班与故障演练。",
	})
	if err != nil {
		t.Fatalf("UpdateJobTarget() error = %v", err)
	}
	if updated.LatestAnalysisStatus != domain.JobTargetAnalysisStale {
		t.Fatalf("expected stale status, got %s", updated.LatestAnalysisStatus)
	}

	svc := New(store, nil)
	_, err = svc.CreateSession(context.Background(), domain.CreateSessionRequest{
		Mode:        domain.ModeBasics,
		Topic:       "redis",
		Intensity:   "standard",
		JobTargetID: target.ID,
	})
	if err == nil {
		t.Fatal("expected CreateSession() to fail")
	}
	if !errors.Is(err, ErrJobTargetNotReady) {
		t.Fatalf("expected ErrJobTargetNotReady, got %v", err)
	}
}

func TestGetDashboardKeepsUnavailableActiveJobTargetVisible(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target := seedReadyJobTarget(t, store)
	if _, err := store.SetActiveJobTarget(context.Background(), target.ID); err != nil {
		t.Fatalf("SetActiveJobTarget() error = %v", err)
	}
	if _, err := store.UpdateJobTarget(context.Background(), target.ID, domain.JobTargetInput{
		Title:       target.Title,
		CompanyName: target.CompanyName,
		SourceText:  target.SourceText + "\n新增：强调稳定性值班与故障演练。",
	}); err != nil {
		t.Fatalf("UpdateJobTarget() error = %v", err)
	}

	svc := New(store, nil)
	dashboard, err := svc.GetDashboard(context.Background())
	if err != nil {
		t.Fatalf("GetDashboard() error = %v", err)
	}
	if dashboard == nil {
		t.Fatal("expected dashboard")
	}
	if dashboard.ActiveJobTarget == nil {
		t.Fatal("expected dashboard to keep exposing active job target")
	}
	if dashboard.ActiveJobTarget.ID != target.ID {
		t.Fatalf("expected active job target id %q, got %q", target.ID, dashboard.ActiveJobTarget.ID)
	}
	if dashboard.ActiveJobTarget.LatestAnalysisStatus != domain.JobTargetAnalysisStale {
		t.Fatalf(
			"expected active job target status %q, got %q",
			domain.JobTargetAnalysisStale,
			dashboard.ActiveJobTarget.LatestAnalysisStatus,
		)
	}
	if dashboard.RecommendationScope != "generic" {
		t.Fatalf("expected recommendation scope to fall back to generic, got %q", dashboard.RecommendationScope)
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

	target := seedReadyJobTarget(t, store)

	session := &domain.TrainingSession{
		ID:                  "sess_score_weights",
		Mode:                domain.ModeBasics,
		Topic:               "redis",
		JobTargetID:         target.ID,
		JobTargetAnalysisID: target.LatestAnalysisID,
		Intensity:           "standard",
		Status:              domain.StatusWaitingAnswer,
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
	if captured.JobTargetAnalysis == nil {
		t.Fatal("expected evaluate answer request to include job target analysis")
	}
	if len(captured.JobTargetAnalysis.EvaluationFocus) == 0 {
		t.Fatal("expected evaluation focus to be forwarded to sidecar")
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
		FollowupAnswer: "另外高效数据结构和 IO 模型也很关键。",
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

func seedReadyJobTarget(t *testing.T, store *repo.Store) *domain.JobTarget {
	t.Helper()

	target, err := store.CreateJobTarget(context.Background(), domain.JobTargetInput{
		Title:       "后端工程师 - Example",
		CompanyName: "Example",
		SourceText:  "负责高并发后端服务开发，要求 Go、Redis、Kafka 经验。",
	})
	if err != nil {
		t.Fatalf("CreateJobTarget() error = %v", err)
	}

	run, err := store.StartJobTargetAnalysis(context.Background(), target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() error = %v", err)
	}

	if err := store.CompleteJobTargetAnalysis(context.Background(), target.ID, run.ID, &domain.AnalyzeJobTargetResponse{
		Summary:          "核心是在招能独立推进高并发后端系统的人。",
		MustHaveSkills:   []string{"Go", "Redis", "Kafka"},
		BonusSkills:      []string{"Kubernetes"},
		Responsibilities: []string{"负责核心服务设计"},
		EvaluationFocus:  []string{"缓存一致性", "故障排查闭环"},
	}); err != nil {
		t.Fatalf("CompleteJobTargetAnalysis() error = %v", err)
	}

	saved, err := store.GetJobTarget(context.Background(), target.ID)
	if err != nil {
		t.Fatalf("GetJobTarget() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved job target")
	}
	return saved
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
