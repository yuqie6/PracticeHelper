package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
)

func TestInternalSearchChunksRequiresValidToken(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project, err := store.CreateImportedProject(context.Background(), &domain.AnalyzeRepoResponse{
		RepoURL:       "https://example.com/mirror.git",
		Name:          "Mirror",
		DefaultBranch: "main",
		ImportCommit:  "abc123",
		Summary:       "Agent workflow",
		Chunks: []domain.RepoChunk{
			{
				FilePath:   "internal/runtime.go",
				FileType:   ".go",
				Content:    "redis cache consistency and retries",
				Importance: 1.0,
				FTSKey:     "internal/runtime.go#0",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateImportedProject() error = %v", err)
	}

	router := NewRouterWithInternalToken(service.New(store, nil), "secret-token")

	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(
		unauthorized,
		httptest.NewRequest(
			http.MethodGet,
			"/internal/search-chunks?project_id="+project.ID+"&query=redis&limit=2",
			nil,
		),
	)
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%s", unauthorized.Code, unauthorized.Body.String())
	}

	request := httptest.NewRequest(
		http.MethodGet,
		"/internal/search-chunks?project_id="+project.ID+"&query=redis&limit=2",
		nil,
	)
	request.Header.Set(internalTokenHeader, "secret-token")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	chunks := decodeDataEnvelope[[]domain.RepoChunk](t, recorder.Body.Bytes())
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].FilePath != "internal/runtime.go" {
		t.Fatalf("unexpected file path: %q", chunks[0].FilePath)
	}
}

func TestInternalSessionDetailReturnsSessionAndReview(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_internal_detail",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicRedis,
		Intensity: "standard",
		Status:    domain.StatusCompleted,
		MaxTurns:  2,
		ReviewID:  "review_internal_detail",
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_internal_detail",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环"},
		Answer:         "因为主要在内存里。",
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := store.CreateReview(context.Background(), &domain.ReviewCard{
		ID:                session.ReviewID,
		SessionID:         session.ID,
		Overall:           "整体过线。",
		TopFix:            "补细节",
		TopFixReason:      "案例不够具体",
		Highlights:        []string{"主线清楚"},
		Gaps:              []string{"案例不足"},
		SuggestedTopics:   []string{"redis"},
		NextTrainingFocus: []string{"补真实案例"},
		ScoreBreakdown:    map[string]float64{"准确性": 80},
	}); err != nil {
		t.Fatalf("CreateReview() error = %v", err)
	}

	router := NewRouterWithInternalToken(service.New(store, nil), "secret-token")
	request := httptest.NewRequest(
		http.MethodGet,
		"/internal/session-detail/"+session.ID,
		nil,
	)
	request.Header.Set(internalTokenHeader, "secret-token")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	detail := decodeDataEnvelope[domain.AgentSessionDetail](t, recorder.Body.Bytes())
	if detail.Session == nil || detail.Session.ID != session.ID {
		t.Fatalf("unexpected session detail: %+v", detail.Session)
	}
	if len(detail.Session.Turns) != 1 {
		t.Fatalf("expected 1 turn, got %+v", detail.Session.Turns)
	}
	if detail.Review == nil || detail.Review.Overall != "整体过线。" {
		t.Fatalf("unexpected review detail: %+v", detail.Review)
	}
}
