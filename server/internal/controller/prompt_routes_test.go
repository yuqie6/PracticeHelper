package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/service"
	"practicehelper/server/internal/sidecar"
)

func TestPromptSetsRouteReturnsConfiguredPromptSets(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	sidecarServer := newPromptSetSidecarServer(t)
	defer sidecarServer.Close()

	router := NewRouter(service.New(store, sidecar.New(sidecarServer.URL, time.Second)))
	request := httptest.NewRequest(http.MethodGet, "/api/prompt-sets", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data []domain.PromptSetSummary `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Data) != 2 {
		t.Fatalf("expected 2 prompt sets, got %d", len(payload.Data))
	}
	if payload.Data[0].ID != "stable-v1" || payload.Data[1].ID != "candidate-v1" {
		t.Fatalf("unexpected prompt sets: %+v", payload.Data)
	}
}

func TestPromptExperimentPromptSetsRouteReturnsHistoricalSetsWithoutSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	seedPromptRouteExperimentSession(
		t,
		store,
		"sess_route_prompt_history_stable",
		domain.PromptSetSummary{ID: "stable-v1", Label: "Stable v1", Status: "stable"},
		84,
	)
	seedPromptRouteExperimentSession(
		t,
		store,
		"sess_route_prompt_history_candidate",
		domain.PromptSetSummary{ID: "candidate-v1", Label: "Candidate v1", Status: "candidate"},
		90,
	)

	router := NewRouter(service.New(store, nil))
	request := httptest.NewRequest(http.MethodGet, "/api/prompt-experiments/prompt-sets", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data []domain.PromptSetSummary `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Data) != 2 {
		t.Fatalf("expected 2 historical prompt sets, got %d", len(payload.Data))
	}
}

func TestPromptExperimentRouteReturnsAggregatedReport(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	seedPromptRouteExperimentSession(
		t,
		store,
		"sess_route_prompt_stable",
		domain.PromptSetSummary{ID: "stable-v1", Label: "Stable v1", Status: "stable"},
		84,
	)
	seedPromptRouteExperimentSession(
		t,
		store,
		"sess_route_prompt_candidate",
		domain.PromptSetSummary{ID: "candidate-v1", Label: "Candidate v1", Status: "candidate"},
		90,
	)

	sidecarServer := newPromptSetSidecarServer(t)
	defer sidecarServer.Close()

	router := NewRouter(service.New(store, sidecar.New(sidecarServer.URL, time.Second)))
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/prompt-experiments?left=stable-v1&right=candidate-v1&mode=basics&topic=redis&limit=10",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data domain.PromptExperimentReport `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Data.Left.SessionCount != 1 || payload.Data.Right.SessionCount != 1 {
		t.Fatalf(
			"unexpected session counts: left=%d right=%d",
			payload.Data.Left.SessionCount,
			payload.Data.Right.SessionCount,
		)
	}
	if len(payload.Data.RecentSamples) != 2 {
		t.Fatalf("expected 2 recent samples, got %d", len(payload.Data.RecentSamples))
	}
}

func TestPromptExperimentRouteReturnsHistoricalReportWithoutSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	seedPromptRouteExperimentSession(
		t,
		store,
		"sess_route_prompt_history_only_stable",
		domain.PromptSetSummary{ID: "stable-v1", Label: "Stable v1", Status: "stable"},
		84,
	)
	seedPromptRouteExperimentSession(
		t,
		store,
		"sess_route_prompt_history_only_candidate",
		domain.PromptSetSummary{ID: "candidate-v1", Label: "Candidate v1", Status: "candidate"},
		90,
	)

	router := NewRouter(service.New(store, nil))
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/prompt-experiments?left=stable-v1&right=candidate-v1&mode=basics&topic=redis&limit=10",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data domain.PromptExperimentReport `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload.Data.Right.PromptSet.Label != "Candidate v1" {
		t.Fatalf("expected historical right label, got %q", payload.Data.Right.PromptSet.Label)
	}
}

func TestSessionEvaluationLogsRouteReturnsPromptAuditFields(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	seedPromptRouteExperimentSession(
		t,
		store,
		"sess_route_logs",
		domain.PromptSetSummary{ID: "stable-v1", Label: "Stable v1", Status: "stable"},
		88,
	)

	router := NewRouter(service.New(store, nil))
	request := httptest.NewRequest(http.MethodGet, "/api/sessions/sess_route_logs/evaluation-logs", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data []domain.EvaluationLogEntry `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Data) != 3 {
		t.Fatalf("expected 3 evaluation logs, got %d", len(payload.Data))
	}
	if payload.Data[0].PromptSetID != "stable-v1" {
		t.Fatalf("expected prompt set id stable-v1, got %s", payload.Data[0].PromptSetID)
	}
	if payload.Data[0].PromptHash == "" {
		t.Fatal("expected prompt hash to be present")
	}
	if payload.Data[0].RawOutput == "" {
		t.Fatal("expected raw output to be present")
	}
}

func newPromptSetSidecarServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/prompt-sets" {
			t.Fatalf("unexpected sidecar path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]domain.PromptSetSummary{
			{ID: "stable-v1", Label: "Stable v1", Status: "stable", IsDefault: true},
			{ID: "candidate-v1", Label: "Candidate v1", Status: "candidate"},
		}); err != nil {
			t.Fatalf("encode prompt sets: %v", err)
		}
	}))
}

func seedPromptRouteExperimentSession(
	t *testing.T,
	store *repo.Store,
	sessionID string,
	promptSet domain.PromptSetSummary,
	totalScore float64,
) {
	t.Helper()

	session := &domain.TrainingSession{
		ID:          sessionID,
		Mode:        domain.ModeBasics,
		Topic:       domain.BasicsTopicRedis,
		PromptSetID: promptSet.ID,
		PromptSet:   &promptSet,
		Intensity:   "standard",
		Status:      domain.StatusCompleted,
		MaxTurns:    1,
		TotalScore:  totalScore,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_" + sessionID,
		SessionID:      sessionID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问"},
		Answer:         "因为热点数据主要在内存中。",
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	logs := []struct {
		flowName   string
		promptHash string
		rawOutput  string
		latencyMs  float64
	}{
		{flowName: "generate_question", promptHash: "hash-question-" + promptSet.ID, rawOutput: "{\"question\":\"Redis 为什么快？\"}", latencyMs: 10},
		{flowName: "evaluate_answer", promptHash: "hash-answer-" + promptSet.ID, rawOutput: "{\"score\":88}", latencyMs: 20},
		{flowName: "generate_review", promptHash: "hash-review-" + promptSet.ID, rawOutput: "{\"overall\":\"还不错\"}", latencyMs: 30},
	}
	for _, item := range logs {
		if err := store.CreateEvaluationLog(
			context.Background(),
			sessionID,
			turn.ID,
			item.flowName,
			"gpt-test",
			promptSet.ID,
			item.promptHash,
			item.rawOutput,
			item.latencyMs,
		); err != nil {
			t.Fatalf("CreateEvaluationLog(%s) error = %v", item.flowName, err)
		}
	}
}
