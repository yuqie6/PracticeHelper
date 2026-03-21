package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
)

func TestDueReviewsRouteReturnsWeaknessMetadata(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	session := &domain.TrainingSession{
		ID:        "sess_route_due_review",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicRedis,
		Intensity: "standard",
		Status:    domain.StatusCompleted,
	}
	turn := &domain.TrainingTurn{
		ID:        "turn_route_due_review",
		SessionID: session.ID,
		TurnIndex: 1,
		Stage:     "question",
		Question:  "Redis 为什么快？",
		Answer:    "因为热点数据主要在内存。",
		WeaknessHits: []domain.WeaknessHit{
			{Kind: "detail", Label: "缓存一致性", Severity: 0.83},
		},
	}
	if err := store.CreateSession(ctx, session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := store.UpsertWeaknesses(ctx, session.ID, turn.WeaknessHits); err != nil {
		t.Fatalf("UpsertWeaknesses() error = %v", err)
	}
	weakness, err := store.GetWeaknessTag(ctx, "detail", "缓存一致性")
	if err != nil {
		t.Fatalf("GetWeaknessTag() error = %v", err)
	}
	if weakness == nil {
		t.Fatal("expected weakness tag to exist")
	}
	if err := store.CreateReviewSchedule(ctx, &domain.ReviewScheduleItem{
		SessionID:     session.ID,
		ReviewCardID:  "review_route_due_review",
		WeaknessTagID: weakness.ID,
		Topic:         domain.BasicsTopicRedis,
		NextReviewAt:  time.Now().UTC().Add(-time.Hour),
		IntervalDays:  1,
		EaseFactor:    2.5,
	}); err != nil {
		t.Fatalf("CreateReviewSchedule() error = %v", err)
	}

	router := NewRouter(service.New(store, nil))
	request := httptest.NewRequest(http.MethodGet, "/api/reviews/due", nil)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Data []domain.ReviewScheduleItem `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Data) != 1 {
		t.Fatalf("expected 1 due review item, got %d", len(payload.Data))
	}
	if payload.Data[0].WeaknessKind != "detail" {
		t.Fatalf("expected weakness kind detail, got %q", payload.Data[0].WeaknessKind)
	}
	if payload.Data[0].WeaknessLabel != "缓存一致性" {
		t.Fatalf("expected weakness label 缓存一致性, got %q", payload.Data[0].WeaknessLabel)
	}
	if payload.Data[0].Topic != domain.BasicsTopicRedis {
		t.Fatalf("expected topic redis, got %q", payload.Data[0].Topic)
	}
}
