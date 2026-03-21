package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/sidecar"
)

func TestCreateSessionRejectsUnknownPromptSet(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handlePromptSetRequest(t, w, r) {
			return
		}
		t.Fatalf("unexpected sidecar path: %s", r.URL.Path)
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	_, err = svc.CreateSession(context.Background(), domain.CreateSessionRequest{
		Mode:        domain.ModeBasics,
		Topic:       domain.BasicsTopicRedis,
		Intensity:   "standard",
		PromptSetID: "missing-v1",
	})
	if !errors.Is(err, ErrPromptSetNotFound) {
		t.Fatalf("expected ErrPromptSetNotFound, got %v", err)
	}
}

func TestGetPromptExperimentAggregatesMetricsAndSamples(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	seedPromptExperimentSession(
		t,
		store,
		"sess_prompt_stable",
		domain.PromptSetSummary{ID: "stable-v1", Label: "Stable v1", Status: "stable"},
		82,
	)
	seedPromptExperimentSession(
		t,
		store,
		"sess_prompt_candidate",
		domain.PromptSetSummary{ID: "candidate-v1", Label: "Candidate v1", Status: "candidate"},
		91,
	)

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handlePromptSetRequest(t, w, r) {
			return
		}
		t.Fatalf("unexpected sidecar path: %s", r.URL.Path)
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	report, err := svc.GetPromptExperiment(context.Background(), domain.PromptExperimentRequest{
		Left:  "stable-v1",
		Right: "candidate-v1",
		Mode:  domain.ModeBasics,
		Topic: domain.BasicsTopicRedis,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("GetPromptExperiment() error = %v", err)
	}

	if report.Left.PromptSet.ID != "stable-v1" {
		t.Fatalf("expected left prompt set stable-v1, got %s", report.Left.PromptSet.ID)
	}
	if report.Right.PromptSet.ID != "candidate-v1" {
		t.Fatalf("expected right prompt set candidate-v1, got %s", report.Right.PromptSet.ID)
	}
	if report.Left.SessionCount != 1 || report.Right.SessionCount != 1 {
		t.Fatalf("unexpected session counts: left=%d right=%d", report.Left.SessionCount, report.Right.SessionCount)
	}
	if report.Left.CompletedCount != 1 || report.Right.CompletedCount != 1 {
		t.Fatalf(
			"unexpected completed counts: left=%d right=%d",
			report.Left.CompletedCount,
			report.Right.CompletedCount,
		)
	}
	if report.Left.AvgTotalScore != 82 {
		t.Fatalf("expected left avg score 82, got %.1f", report.Left.AvgTotalScore)
	}
	if report.Right.AvgTotalScore != 91 {
		t.Fatalf("expected right avg score 91, got %.1f", report.Right.AvgTotalScore)
	}
	if report.Left.AvgGenerateQuestionLatencyMs != 11 {
		t.Fatalf(
			"expected left question latency 11, got %.1f",
			report.Left.AvgGenerateQuestionLatencyMs,
		)
	}
	if report.Right.AvgGenerateReviewLatencyMs != 33 {
		t.Fatalf(
			"expected right review latency 33, got %.1f",
			report.Right.AvgGenerateReviewLatencyMs,
		)
	}
	if len(report.RecentSamples) != 2 {
		t.Fatalf("expected 2 recent samples, got %d", len(report.RecentSamples))
	}
}

func TestGetPromptExperimentFallsBackToHistoricalPromptSetsWithoutSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	seedPromptExperimentSession(
		t,
		store,
		"sess_prompt_history_only_stable",
		domain.PromptSetSummary{ID: "stable-v1", Label: "Stable v1", Status: "stable"},
		82,
	)
	seedPromptExperimentSession(
		t,
		store,
		"sess_prompt_history_only_candidate",
		domain.PromptSetSummary{ID: "candidate-v1", Label: "Candidate v1", Status: "candidate"},
		91,
	)

	svc := New(store, nil)
	report, err := svc.GetPromptExperiment(context.Background(), domain.PromptExperimentRequest{
		Left:  "stable-v1",
		Right: "candidate-v1",
		Mode:  domain.ModeBasics,
		Topic: domain.BasicsTopicRedis,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("GetPromptExperiment() error = %v", err)
	}

	if report.Left.PromptSet.Label != "Stable v1" {
		t.Fatalf("expected historical left label, got %q", report.Left.PromptSet.Label)
	}
	if report.Right.PromptSet.Label != "Candidate v1" {
		t.Fatalf("expected historical right label, got %q", report.Right.PromptSet.Label)
	}
}

func TestGetPromptExperimentUsesHistoricalPromptSetWhenMissingFromLiveRegistry(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	seedPromptExperimentSession(
		t,
		store,
		"sess_prompt_partial_live_stable",
		domain.PromptSetSummary{ID: "stable-v1", Label: "Stable v1", Status: "stable"},
		82,
	)
	seedPromptExperimentSession(
		t,
		store,
		"sess_prompt_partial_live_candidate",
		domain.PromptSetSummary{ID: "candidate-v1", Label: "Candidate v1", Status: "candidate"},
		91,
	)

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/prompt-sets" {
			t.Fatalf("unexpected sidecar path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]domain.PromptSetSummary{
			{
				ID:          "stable-v1",
				Label:       "Stable v1",
				Description: "Current stable prompt set",
				Status:      "stable",
				IsDefault:   true,
			},
		}); err != nil {
			t.Fatalf("encode prompt sets: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	report, err := svc.GetPromptExperiment(context.Background(), domain.PromptExperimentRequest{
		Left:  "stable-v1",
		Right: "candidate-v1",
		Mode:  domain.ModeBasics,
		Topic: domain.BasicsTopicRedis,
		Limit: 10,
	})
	if err != nil {
		t.Fatalf("GetPromptExperiment() error = %v", err)
	}

	if report.Left.PromptSet.Description != "Current stable prompt set" {
		t.Fatalf("expected live description enrichment, got %q", report.Left.PromptSet.Description)
	}
	if report.Right.PromptSet.Label != "Candidate v1" {
		t.Fatalf("expected historical candidate label, got %q", report.Right.PromptSet.Label)
	}
	if report.Right.PromptSet.Description != "" {
		t.Fatalf("expected missing live prompt set to keep empty description, got %q", report.Right.PromptSet.Description)
	}
}

func TestListPromptExperimentPromptSetsReturnsHistoricalSetsWithLiveEnrichment(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	seedPromptExperimentSession(
		t,
		store,
		"sess_prompt_list_stable",
		domain.PromptSetSummary{ID: "stable-v1", Label: "Stable v1", Status: "stable"},
		82,
	)
	seedPromptExperimentSession(
		t,
		store,
		"sess_prompt_list_candidate",
		domain.PromptSetSummary{ID: "candidate-v1", Label: "Candidate v1", Status: "candidate"},
		91,
	)

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/prompt-sets" {
			t.Fatalf("unexpected sidecar path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode([]domain.PromptSetSummary{
			{
				ID:          "stable-v1",
				Label:       "Stable v1",
				Description: "Current stable prompt set",
				Status:      "stable",
				IsDefault:   true,
			},
		}); err != nil {
			t.Fatalf("encode prompt sets: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	items, err := svc.ListPromptExperimentPromptSets(context.Background())
	if err != nil {
		t.Fatalf("ListPromptExperimentPromptSets() error = %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 historical prompt sets, got %d", len(items))
	}
	if items[0].ID != "candidate-v1" && items[1].ID != "candidate-v1" {
		t.Fatalf("expected candidate-v1 in historical prompt sets, got %+v", items)
	}
	var stable domain.PromptSetSummary
	for _, item := range items {
		if item.ID == "stable-v1" {
			stable = item
		}
	}
	if stable.Description != "Current stable prompt set" {
		t.Fatalf("expected stable prompt set to be enriched, got %q", stable.Description)
	}
	if !stable.IsDefault {
		t.Fatal("expected stable prompt set to keep live default flag")
	}
}

func seedPromptExperimentSession(
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
		{flowName: "generate_question", promptHash: "hash-question-" + promptSet.ID, rawOutput: "{\"question\":\"Q\"}", latencyMs: 11},
		{flowName: "evaluate_answer", promptHash: "hash-answer-" + promptSet.ID, rawOutput: "{\"score\":92}", latencyMs: 22},
		{flowName: "generate_review", promptHash: "hash-review-" + promptSet.ID, rawOutput: "{\"overall\":\"A\"}", latencyMs: 33},
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
