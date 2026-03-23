package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestPromptExperimentRouteExcludesSessionsWithPromptOverlay(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	seedPromptRouteExperimentSession(
		t,
		store,
		"sess_route_prompt_stable_plain",
		domain.PromptSetSummary{ID: "stable-v1", Label: "Stable v1", Status: "stable"},
		84,
	)
	seedPromptRouteExperimentSessionWithOverlay(
		t,
		store,
		"sess_route_prompt_stable_overlay",
		domain.PromptSetSummary{ID: "stable-v1", Label: "Stable v1", Status: "stable"},
		91,
		"overlay-hash-stable",
	)
	seedPromptRouteExperimentSession(
		t,
		store,
		"sess_route_prompt_candidate_plain",
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
	if payload.Data.Left.SessionCount != 1 {
		t.Fatalf("expected stable session count 1 after excluding overlay, got %d", payload.Data.Left.SessionCount)
	}
	if len(payload.Data.RecentSamples) != 2 {
		t.Fatalf("expected 2 recent samples after excluding overlay, got %d", len(payload.Data.RecentSamples))
	}
	for _, item := range payload.Data.RecentSamples {
		if item.SessionID == "sess_route_prompt_stable_overlay" {
			t.Fatal("expected overlay session to be excluded from prompt experiments")
		}
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

func TestPromptPreferencesRoutesRoundTripNormalizedOverlay(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	router := NewRouter(service.New(store, nil))

	patchRequest := httptest.NewRequest(
		http.MethodPatch,
		"/api/prompt-preferences",
		strings.NewReader(`{
			"tone":" direct ",
			"detail_level":"detailed",
			"followup_intensity":"pressure",
			"answer_language":"en-us",
			"focus_tags":["depth","structure"],
			"custom_instruction":"  多追问边界和取舍。  "
		}`),
	)
	patchRequest.Header.Set("Content-Type", "application/json")
	patchRecorder := httptest.NewRecorder()
	router.ServeHTTP(patchRecorder, patchRequest)

	if patchRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", patchRecorder.Code, patchRecorder.Body.String())
	}

	var patchPayload struct {
		Data domain.PromptOverlay `json:"data"`
	}
	if err := json.Unmarshal(patchRecorder.Body.Bytes(), &patchPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if patchPayload.Data.Tone != "direct" {
		t.Fatalf("expected tone direct, got %q", patchPayload.Data.Tone)
	}
	if patchPayload.Data.AnswerLanguage != "en-US" {
		t.Fatalf("expected answer language en-US, got %q", patchPayload.Data.AnswerLanguage)
	}
	if patchPayload.Data.CustomInstruction != "多追问边界和取舍。" {
		t.Fatalf("expected trimmed instruction, got %q", patchPayload.Data.CustomInstruction)
	}

	getRequest := httptest.NewRequest(http.MethodGet, "/api/prompt-preferences", nil)
	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(getRecorder, getRequest)

	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getRecorder.Code, getRecorder.Body.String())
	}

	var getPayload struct {
		Data domain.PromptOverlay `json:"data"`
	}
	if err := json.Unmarshal(getRecorder.Body.Bytes(), &getPayload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if getPayload.Data.DetailLevel != "detailed" {
		t.Fatalf("expected detail level detailed, got %q", getPayload.Data.DetailLevel)
	}
	if len(getPayload.Data.FocusTags) != 2 {
		t.Fatalf("expected 2 focus tags, got %v", getPayload.Data.FocusTags)
	}
}

func TestPromptPreferencesPatchRejectsInvalidOverlay(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	router := NewRouter(service.New(store, nil))
	request := httptest.NewRequest(
		http.MethodPatch,
		"/api/prompt-preferences",
		strings.NewReader(`{"focus_tags":["depth","structure","expression"]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
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
	seedPromptRouteExperimentSessionWithConfig(t, store, session, promptSet)
}

func seedPromptRouteExperimentSessionWithOverlay(
	t *testing.T,
	store *repo.Store,
	sessionID string,
	promptSet domain.PromptSetSummary,
	totalScore float64,
	overlayHash string,
) {
	t.Helper()

	session := &domain.TrainingSession{
		ID:                sessionID,
		Mode:              domain.ModeBasics,
		Topic:             domain.BasicsTopicRedis,
		PromptSetID:       promptSet.ID,
		PromptSet:         &promptSet,
		PromptOverlay:     &domain.PromptOverlay{Tone: "direct"},
		PromptOverlayHash: overlayHash,
		Intensity:         "standard",
		Status:            domain.StatusCompleted,
		MaxTurns:          1,
		TotalScore:        totalScore,
	}
	seedPromptRouteExperimentSessionWithConfig(t, store, session, promptSet)
}

func seedPromptRouteExperimentSessionWithConfig(
	t *testing.T,
	store *repo.Store,
	session *domain.TrainingSession,
	promptSet domain.PromptSetSummary,
) {
	t.Helper()

	turn := &domain.TrainingTurn{
		ID:             "turn_" + session.ID,
		SessionID:      session.ID,
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
			session.ID,
			turn.ID,
			item.flowName,
			"gpt-test",
			promptSet.ID,
			item.promptHash,
			item.rawOutput,
			nil,
			item.latencyMs,
		); err != nil {
			t.Fatalf("CreateEvaluationLog(%s) error = %v", item.flowName, err)
		}
	}
}
