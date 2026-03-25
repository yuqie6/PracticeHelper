package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
)

func TestDeleteSessionsRouteDeletesSelectedSessions(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_route_delete",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicRedis,
		Intensity: "standard",
		Status:    domain.StatusCompleted,
		MaxTurns:  1,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_route_delete",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问"},
		Answer:         "因为热点数据在内存里。",
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	router := NewRouter(service.New(store, nil))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/sessions/delete",
		strings.NewReader(`{"session_ids":["sess_route_delete"]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	result := decodeDataEnvelope[domain.DeleteSessionsResult](t, recorder.Body.Bytes())
	if result.DeletedCount != 1 || len(result.DeletedSessionIDs) != 1 || result.DeletedSessionIDs[0] != session.ID {
		t.Fatalf("unexpected delete result: %+v", result)
	}

	if saved, err := store.GetSession(context.Background(), session.ID); err != nil {
		t.Fatalf("GetSession() error = %v", err)
	} else if saved != nil {
		t.Fatalf("expected session %s to be deleted", session.ID)
	}
}

func TestDeleteSessionsRouteRejectsEmptySelection(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	router := NewRouter(service.New(store, nil))
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/sessions/delete",
		strings.NewReader(`{"session_ids":[]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	payload := decodeErrorPayload(t, recorder.Body.Bytes())
	if payload.Error.Code != "empty_delete_selection" {
		t.Fatalf("unexpected error code: %s", payload.Error.Code)
	}
}
