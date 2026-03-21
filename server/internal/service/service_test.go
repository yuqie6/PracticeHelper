package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"slices"
	"strings"
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

func TestCreateSessionUsesOSTemplates(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

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
			Question:       "说说一次上下文切换为什么会拖慢服务。",
			ExpectedPoints: []string{"触发条件", "成本来源", "止血思路"},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	_, err = svc.CreateSession(context.Background(), domain.CreateSessionRequest{
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicOS,
		Intensity: "standard",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if captured.Topic != domain.BasicsTopicOS {
		t.Fatalf("expected topic %q, got %q", domain.BasicsTopicOS, captured.Topic)
	}
	if len(captured.CandidateTopics) != 1 || captured.CandidateTopics[0] != domain.BasicsTopicOS {
		t.Fatalf("expected candidate topics [%s], got %v", domain.BasicsTopicOS, captured.CandidateTopics)
	}
	if len(captured.Templates) == 0 {
		t.Fatal("expected os templates to be forwarded")
	}
	for _, item := range captured.Templates {
		if item.Topic != domain.BasicsTopicOS {
			t.Fatalf("expected only os templates, got topic %q", item.Topic)
		}
	}
}

func TestCreateSessionMixedSelectsTopicsFromWeaknesses(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.UpsertWeaknesses(context.Background(), "sess_seed", []domain.WeaknessHit{
		{Kind: "topic", Label: "Redis 缓存一致性", Severity: 1.1},
		{Kind: "detail", Label: "MySQL 索引设计说不清", Severity: 0.9},
		{Kind: "followup_breakdown", Label: "进程线程调度容易乱", Severity: 0.8},
	}); err != nil {
		t.Fatalf("UpsertWeaknesses() error = %v", err)
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
			Question:       "讲讲缓存一致性和索引设计的取舍。",
			ExpectedPoints: []string{"Redis", "MySQL", "代价"},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	session, err := svc.CreateSession(context.Background(), domain.CreateSessionRequest{
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicMixed,
		Intensity: "standard",
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session.Topic != domain.BasicsTopicMixed {
		t.Fatalf("expected session topic %q, got %q", domain.BasicsTopicMixed, session.Topic)
	}
	if captured.Topic != domain.BasicsTopicMixed {
		t.Fatalf("expected request topic %q, got %q", domain.BasicsTopicMixed, captured.Topic)
	}
	if len(captured.CandidateTopics) < 2 {
		t.Fatalf("expected mixed mode to carry multiple candidate topics, got %v", captured.CandidateTopics)
	}
	if !slices.Contains(captured.CandidateTopics, domain.BasicsTopicRedis) {
		t.Fatalf("expected candidate topics to include redis, got %v", captured.CandidateTopics)
	}
	if !slices.Contains(captured.CandidateTopics, domain.BasicsTopicMySQL) {
		t.Fatalf("expected candidate topics to include mysql, got %v", captured.CandidateTopics)
	}
	if !slices.Contains(captured.CandidateTopics, domain.BasicsTopicOS) {
		t.Fatalf("expected candidate topics to include os, got %v", captured.CandidateTopics)
	}

	seenTopics := map[string]bool{}
	for _, item := range captured.Templates {
		seenTopics[item.Topic] = true
	}
	for _, topic := range captured.CandidateTopics {
		if !seenTopics[topic] {
			t.Fatalf("expected templates for candidate topic %q, seen=%v", topic, seenTopics)
		}
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

func TestCreateSessionRejectsExplicitRunningOrFailedJobTarget(t *testing.T) {
	cases := []struct {
		name       string
		prepare    func(*testing.T, *repo.Store, *domain.JobTarget)
		wantStatus string
	}{
		{
			name:       "running",
			prepare:    markJobTargetAnalysisRunning,
			wantStatus: domain.JobTargetAnalysisRunning,
		},
		{
			name:       "failed",
			prepare:    markJobTargetAnalysisFailed,
			wantStatus: domain.JobTargetAnalysisFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, err := openTestStore(t)
			if err != nil {
				t.Fatalf("openTestStore() error = %v", err)
			}
			defer func() { _ = store.Close() }()

			target := seedReadyJobTarget(t, store)
			tc.prepare(t, store, target)

			saved, err := store.GetJobTarget(context.Background(), target.ID)
			if err != nil {
				t.Fatalf("GetJobTarget() error = %v", err)
			}
			if saved == nil {
				t.Fatal("expected saved job target")
			}
			if saved.LatestAnalysisStatus != tc.wantStatus {
				t.Fatalf("expected %s status, got %s", tc.wantStatus, saved.LatestAnalysisStatus)
			}
			if saved.LatestSuccessfulAnalysis == nil {
				t.Fatal("expected latest successful analysis to remain available")
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
		})
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

func TestGetDashboardKeepsRunningOrFailedActiveJobTargetVisible(t *testing.T) {
	cases := []struct {
		name       string
		prepare    func(*testing.T, *repo.Store, *domain.JobTarget)
		wantStatus string
	}{
		{
			name:       "running",
			prepare:    markJobTargetAnalysisRunning,
			wantStatus: domain.JobTargetAnalysisRunning,
		},
		{
			name:       "failed",
			prepare:    markJobTargetAnalysisFailed,
			wantStatus: domain.JobTargetAnalysisFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, err := openTestStore(t)
			if err != nil {
				t.Fatalf("openTestStore() error = %v", err)
			}
			defer func() { _ = store.Close() }()

			target := seedReadyJobTarget(t, store)
			if _, err := store.SetActiveJobTarget(context.Background(), target.ID); err != nil {
				t.Fatalf("SetActiveJobTarget() error = %v", err)
			}
			tc.prepare(t, store, target)

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
			if dashboard.ActiveJobTarget.LatestAnalysisStatus != tc.wantStatus {
				t.Fatalf("expected active job target status %q, got %q", tc.wantStatus, dashboard.ActiveJobTarget.LatestAnalysisStatus)
			}
			if dashboard.RecommendationScope != "generic" {
				t.Fatalf("expected recommendation scope to fall back to generic, got %q", dashboard.RecommendationScope)
			}
		})
	}
}

func TestCreateSessionWritesEvaluationLog(t *testing.T) {
	db, err := sqlite.Open(filepath.Join(t.TempDir(), "practicehelper.db"))
	if err != nil {
		t.Fatalf("sqlite.Open() error = %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := sqlite.Bootstrap(db); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	store := repo.New(db)

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/generate_question" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
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

	var (
		sessionID string
		turnID    string
		flowName  string
		modelName string
		latencyMs float64
	)
	if err := db.QueryRow(`
		SELECT session_id, turn_id, flow_name, model_name, latency_ms
		FROM evaluation_logs
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&sessionID, &turnID, &flowName, &modelName, &latencyMs); err != nil {
		t.Fatalf("query evaluation log: %v", err)
	}

	if sessionID != session.ID {
		t.Fatalf("expected evaluation log session id %q, got %q", session.ID, sessionID)
	}
	if turnID != "" {
		t.Fatalf("expected create session log turn id to be empty, got %q", turnID)
	}
	if flowName != "generate_question" {
		t.Fatalf("expected flow name generate_question, got %q", flowName)
	}
	if modelName != "" {
		t.Fatalf("expected model name to be empty, got %q", modelName)
	}
	if latencyMs < 0 {
		t.Fatalf("expected non-negative latency, got %.2f", latencyMs)
	}
}

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
	if err := store.CreateReviewSchedule(context.Background(), &domain.ReviewScheduleItem{
		SessionID:    session.ID,
		ReviewCardID: "review_due_review",
		Topic:        session.Topic,
		NextReviewAt: time.Now().UTC().Add(-time.Hour),
		IntervalDays: 1,
		EaseFactor:   2.5,
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
		MaxTurns:            2,
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
		MaxTurns:  2,
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

	if updated.Status != domain.StatusWaitingAnswer {
		t.Fatalf("expected waiting_answer status, got %s", updated.Status)
	}
	if len(updated.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(updated.Turns))
	}
	if updated.Turns[1].Question != "那你在项目里什么时候会避免过度用 channel？" {
		t.Fatalf("expected followup question as turn 2, got %q", updated.Turns[1].Question)
	}

	assertStatusEventNames(t, events, []string{
		"answer_received",
		"evaluation_started",
		"feedback_ready",
		"answer_saved",
		"followup_ready",
	})
}

func TestSubmitAnswerStreamEmitsStatusSequenceForLastTurn(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:         "sess_stream_last",
		Mode:       domain.ModeBasics,
		Topic:      "go",
		Intensity:  "standard",
		MaxTurns:   2,
		Status:     domain.StatusWaitingAnswer,
		TotalScore: 72,
	}
	turn1 := &domain.TrainingTurn{
		ID:             "turn_stream_last_1",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 channel 和 mutex 什么时候各用什么？",
		ExpectedPoints: []string{"共享内存", "所有权转移"},
		Answer:         "主回答已经提交",
	}
	if err := store.CreateSession(context.Background(), session, turn1); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	turn2 := &domain.TrainingTurn{
		ID:             "turn_stream_last_2",
		SessionID:      session.ID,
		TurnIndex:      2,
		Stage:          "question",
		Question:       "那你在项目里什么时候会避免过度用 channel？",
		ExpectedPoints: []string{"性能", "复杂度"},
	}
	if err := store.InsertTurn(context.Background(), turn2); err != nil {
		t.Fatalf("InsertTurn() error = %v", err)
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

func TestExportSessionMarkdownIncludesTurnsAndReview(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedSessionForExport(t, store, "sess_export_with_review", true)
	svc := New(store, nil)

	filename, content, err := svc.ExportSession(context.Background(), session.ID, "markdown")
	if err != nil {
		t.Fatalf("ExportSession() error = %v", err)
	}

	if filename != "practicehelper-session-sess_export_with_review.md" {
		t.Fatalf("unexpected filename: %s", filename)
	}

	body := string(content)
	for _, snippet := range []string{
		"# PracticeHelper Session 报告",
		"- Session ID: sess_export_with_review",
		"- 绑定岗位: 后端工程师 - Example (Example)",
		"### 第 1 轮",
		"Redis 为什么快？",
		"缓存淘汰策略",
		"### 回答亮点",
		"举了缓存雪崩的真实场景",
	} {
		if !strings.Contains(body, snippet) {
			t.Fatalf("expected export body to contain %q, got:\n%s", snippet, body)
		}
	}
}

func TestExportSessionMarkdownWithoutReviewShowsPendingNotice(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedSessionForExport(t, store, "sess_export_without_review", false)
	svc := New(store, nil)

	filename, content, err := svc.ExportSession(context.Background(), session.ID, "markdown")
	if err != nil {
		t.Fatalf("ExportSession() error = %v", err)
	}

	if filename != "practicehelper-session-sess_export_without_review.md" {
		t.Fatalf("unexpected filename: %s", filename)
	}

	body := string(content)
	if !strings.Contains(body, "> 复盘未生成。") {
		t.Fatalf("expected export body to mention missing review, got:\n%s", body)
	}
	if !strings.Contains(body, "### 第 2 轮") {
		t.Fatalf("expected export body to include follow-up turn, got:\n%s", body)
	}
}

func seedSessionForExport(
	t *testing.T,
	store *repo.Store,
	sessionID string,
	withReview bool,
) *domain.TrainingSession {
	t.Helper()

	ctx := context.Background()
	target := seedReadyJobTarget(t, store)
	startedAt := time.Date(2026, 3, 21, 9, 30, 0, 0, time.UTC)
	endedAt := startedAt.Add(18 * time.Minute)

	status := domain.StatusReviewPending
	reviewID := ""
	if withReview {
		status = domain.StatusCompleted
		reviewID = "review_" + sessionID
	}

	session := &domain.TrainingSession{
		ID:                  sessionID,
		Mode:                domain.ModeBasics,
		Topic:               "redis",
		JobTargetID:         target.ID,
		JobTargetAnalysisID: target.LatestSuccessfulAnalysis.ID,
		Intensity:           "pressure",
		Status:              status,
		MaxTurns:            2,
		TotalScore:          82,
		StartedAt:           &startedAt,
		EndedAt:             &endedAt,
		ReviewID:            reviewID,
	}
	turn1 := &domain.TrainingTurn{
		ID:             "turn_1_" + sessionID,
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "单线程事件循环", "合理的数据结构"},
		Answer:         "核心是内存访问快，再加上单线程减少了线程切换成本。",
		Evaluation: &domain.EvaluationResult{
			Score:            78,
			ScoreBreakdown:   map[string]float64{"准确性": 30, "完整性": 24},
			Headline:         "主结论没问题，但对代价讲得还不够。",
			Strengths:        []string{"先说了核心结论"},
			Gaps:             []string{"没有补多路复用和数据结构"},
			Suggestion:       "补上 IO 模型和常见数据结构，回答会更完整。",
			FollowupIntent:   "确认你是否真的理解性能来自哪里。",
			FollowupQuestion: "如果 QPS 上来后出现 CPU 飙高，你会怎么定位？",
			WeaknessHits: []domain.WeaknessHit{
				{Kind: "topic", Label: "缓存淘汰策略", Severity: 0.72},
			},
		},
	}
	if err := store.CreateSession(ctx, session, turn1); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	turn2 := &domain.TrainingTurn{
		ID:             "turn_2_" + sessionID,
		SessionID:      session.ID,
		TurnIndex:      2,
		Stage:          "followup",
		Question:       "如果 QPS 上来后出现 CPU 飙高，你会怎么定位？",
		ExpectedPoints: []string{"先看慢命令", "区分热点 key", "确认是否存在大 key"},
		Answer:         "我会先看 Redis 慢日志，再排查热点 key 和大 key。",
		Evaluation: &domain.EvaluationResult{
			Score:          86,
			ScoreBreakdown: map[string]float64{"准确性": 32, "完整性": 28},
			Headline:       "排查顺序是对的，但还可以补监控闭环。",
			Strengths:      []string{"排查动作顺序合理"},
			Gaps:           []string{"没有提监控和压测回放"},
			Suggestion:     "补监控指标和压测回放，答案会更像真实线上排障。",
		},
	}
	if err := store.InsertTurn(ctx, turn2); err != nil {
		t.Fatalf("InsertTurn() error = %v", err)
	}

	if withReview {
		review := &domain.ReviewCard{
			ID:                reviewID,
			SessionID:         session.ID,
			Overall:           "整体回答有框架，但还要把线上取舍说实。",
			TopFix:            "把缓存淘汰策略和排障闭环讲完整",
			TopFixReason:      "这是这轮最影响说服力的短板。",
			Highlights:        []string{"举了缓存雪崩的真实场景"},
			Gaps:              []string{"缓存淘汰策略讲得太虚"},
			SuggestedTopics:   []string{"redis", "system_design"},
			NextTrainingFocus: []string{"围绕 Redis 性能与故障排查做一轮专项训练"},
			RecommendedNext: &domain.NextSession{
				Mode:   domain.ModeBasics,
				Topic:  "redis",
				Reason: "先把 Redis 相关薄弱点补齐。",
			},
			ScoreBreakdown: map[string]float64{"表达清晰度": 78, "完整性": 76},
		}
		if err := store.CreateReview(ctx, review); err != nil {
			t.Fatalf("CreateReview() error = %v", err)
		}
	}

	saved, err := store.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved session")
	}
	return saved
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

func markJobTargetAnalysisRunning(t *testing.T, store *repo.Store, target *domain.JobTarget) {
	t.Helper()

	run, err := store.StartJobTargetAnalysis(context.Background(), target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() error = %v", err)
	}
	if run.Status != domain.JobTargetAnalysisRunning {
		t.Fatalf("expected running status, got %s", run.Status)
	}
}

func markJobTargetAnalysisFailed(t *testing.T, store *repo.Store, target *domain.JobTarget) {
	t.Helper()

	run, err := store.StartJobTargetAnalysis(context.Background(), target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() error = %v", err)
	}
	if err := store.FailJobTargetAnalysis(context.Background(), target.ID, run.ID, "boom"); err != nil {
		t.Fatalf("FailJobTargetAnalysis() error = %v", err)
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
