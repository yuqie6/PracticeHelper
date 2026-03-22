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

func TestCreateSessionPassesBoundJobTargetAnalysisToSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target := seedReadyJobTarget(t, store)

	var captured domain.GenerateQuestionRequest
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handlePromptSetRequest(t, w, r) {
			return
		}
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
	if session.PromptSetID != "stable-v1" {
		t.Fatalf("expected session prompt set id stable-v1, got %q", session.PromptSetID)
	}
	if session.JobTargetAnalysisID == "" {
		t.Fatal("expected session to persist a job target analysis id")
	}
	if captured.PromptSetID != "stable-v1" {
		t.Fatalf("expected generate question request prompt set id stable-v1, got %q", captured.PromptSetID)
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
		if handlePromptSetRequest(t, w, r) {
			return
		}
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
		if handlePromptSetRequest(t, w, r) {
			return
		}
		if r.URL.Path != "/internal/generate_question" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
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
		if handlePromptSetRequest(t, w, r) {
			return
		}
		if r.URL.Path != "/internal/generate_question" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
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
		if handlePromptSetRequest(t, w, r) {
			return
		}
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
		if handlePromptSetRequest(t, w, r) {
			return
		}
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
		if handlePromptSetRequest(t, w, r) {
			return
		}
		if r.URL.Path != "/internal/generate_question" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("X-PracticeHelper-Prompt-Set", "stable-v1")
		w.Header().Set("X-PracticeHelper-Prompt-Hash", "hash-generate-question")
		w.Header().Set("X-PracticeHelper-Model-Name", "gpt-test")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"result": domain.GenerateQuestionResponse{
				Question:       "Redis 为什么快？",
				ExpectedPoints: []string{"内存访问", "事件循环"},
			},
			"raw_output": `{"question":"Redis 为什么快？","expected_points":["内存访问","事件循环"]}`,
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
		sessionID   string
		turnID      string
		flowName    string
		modelName   string
		promptSetID string
		promptHash  string
		rawOutput   string
		latencyMs   float64
	)
	if err := db.QueryRow(`
		SELECT session_id, turn_id, flow_name, model_name, prompt_set_id, prompt_hash, raw_output, latency_ms
		FROM evaluation_logs
		ORDER BY id DESC
		LIMIT 1
	`).Scan(&sessionID, &turnID, &flowName, &modelName, &promptSetID, &promptHash, &rawOutput, &latencyMs); err != nil {
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
	if modelName != "gpt-test" {
		t.Fatalf("expected model name gpt-test, got %q", modelName)
	}
	if promptSetID != "stable-v1" {
		t.Fatalf("expected prompt set id stable-v1, got %q", promptSetID)
	}
	if promptHash != "hash-generate-question" {
		t.Fatalf("expected prompt hash hash-generate-question, got %q", promptHash)
	}
	if !strings.Contains(rawOutput, `"question":"Redis 为什么快？"`) {
		t.Fatalf("expected raw output to be persisted, got %q", rawOutput)
	}
	if latencyMs < 0 {
		t.Fatalf("expected non-negative latency, got %.2f", latencyMs)
	}
}
