package service

import (
	"context"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
)

func TestBuildReviewScheduleItemsCreatesWeaknessLevelSchedules(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	hits := []domain.WeaknessHit{
		{Kind: "detail", Label: "缓存一致性", Severity: 0.92},
		{Kind: "topic", Label: "Kafka offset 提交", Severity: 0.81},
	}
	if err := store.UpsertWeaknesses(ctx, "sess_seed_review_schedule", hits); err != nil {
		t.Fatalf("UpsertWeaknesses() error = %v", err)
	}

	svc := New(store, nil)
	items := svc.buildReviewScheduleItems(
		ctx,
		&domain.TrainingSession{
			ID:   "sess_review_schedule_items",
			Mode: domain.ModeProject,
			Turns: []domain.TrainingTurn{
				{
					ID:           "turn_1",
					SessionID:    "sess_review_schedule_items",
					TurnIndex:    1,
					WeaknessHits: hits,
				},
			},
		},
		&domain.ReviewCard{
			ID:              "review_schedule_items",
			SuggestedTopics: []string{domain.BasicsTopicRedis},
		},
		time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC),
	)

	if len(items) != 2 {
		t.Fatalf("expected 2 schedule items, got %d", len(items))
	}

	byLabel := make(map[string]domain.ReviewScheduleItem, len(items))
	for _, item := range items {
		byLabel[item.WeaknessLabel] = item
		if item.WeaknessTagID == "" {
			t.Fatalf("expected weakness schedule to carry weakness_tag_id for %q", item.WeaknessLabel)
		}
	}

	detailItem, ok := byLabel["缓存一致性"]
	if !ok {
		t.Fatalf("expected 缓存一致性 schedule, got %#v", byLabel)
	}
	if detailItem.Topic != domain.BasicsTopicRedis {
		t.Fatalf("expected detail weakness to fall back to suggested topic redis, got %q", detailItem.Topic)
	}

	topicItem, ok := byLabel["Kafka offset 提交"]
	if !ok {
		t.Fatalf("expected Kafka weakness schedule, got %#v", byLabel)
	}
	if topicItem.Topic != domain.BasicsTopicKafka {
		t.Fatalf("expected topic weakness to resolve to kafka, got %q", topicItem.Topic)
	}
}

func TestBuildReviewScheduleItemsFallsBackToGenericSessionSchedule(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	svc := New(store, nil)
	items := svc.buildReviewScheduleItems(
		context.Background(),
		&domain.TrainingSession{
			ID:    "sess_review_schedule_fallback",
			Mode:  domain.ModeBasics,
			Topic: domain.BasicsTopicMySQL,
		},
		&domain.ReviewCard{
			ID: "review_schedule_fallback",
		},
		time.Date(2026, 3, 22, 9, 0, 0, 0, time.UTC),
	)

	if len(items) != 1 {
		t.Fatalf("expected 1 fallback schedule item, got %d", len(items))
	}
	if items[0].WeaknessTagID != "" {
		t.Fatalf("expected generic fallback schedule without weakness tag, got %q", items[0].WeaknessTagID)
	}
	if items[0].Topic != domain.BasicsTopicMySQL {
		t.Fatalf("expected fallback topic mysql, got %q", items[0].Topic)
	}
}
