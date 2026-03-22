package service

import (
	"context"
	"testing"

	"practicehelper/server/internal/domain"
)

func TestExecuteAgentCommandTransitionSessionReturnsDeferredExtend(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_cmd_transition",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicRedis,
		Intensity: "standard",
		MaxTurns:  1,
		Status:    domain.StatusEvaluating,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_cmd_transition",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	result, err := svc.ExecuteAgentCommand(context.Background(), domain.AgentCommandEnvelope{
		CommandID:      "cmd_transition_session_turn_1_extend",
		CommandType:    domain.AgentCommandTypeTransitionSession,
		SessionID:      session.ID,
		IdempotencyKey: "sess_cmd_transition:evaluate_answer:transition_session:1:extend",
		Payload: map[string]any{
			"decision":          "extend",
			"turn_index":        1,
			"current_max_turns": 1,
		},
	})
	if err != nil {
		t.Fatalf("ExecuteAgentCommand() error = %v", err)
	}

	if result.Status != domain.AgentCommandStatusDeferred {
		t.Fatalf("expected deferred status, got %+v", result)
	}
	if intValue(result.Data["resolved_max_turns"], 0) != 2 {
		t.Fatalf("expected resolved max turns 2, got %+v", result.Data)
	}
}

func TestExecuteAgentCommandUpsertReviewPathReturnsAppliedNormalizedPath(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_cmd_review_path",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicGo,
		Intensity: "standard",
		Status:    domain.StatusReviewPending,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_cmd_review_path",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 map 为什么并发不安全？",
		ExpectedPoints: []string{"hash map 扩容", "竞态写入"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	result, err := svc.ExecuteAgentCommand(context.Background(), domain.AgentCommandEnvelope{
		CommandID:      "cmd_upsert_review_path",
		CommandType:    domain.AgentCommandTypeUpsertReviewPath,
		SessionID:      session.ID,
		IdempotencyKey: "sess_cmd_review_path:generate_review:upsert_review_path",
		Payload: map[string]any{
			"recommended_next": map[string]any{
				"mode":  "basics",
				"topic": "redis",
			},
			"suggested_topics":    []any{"redis"},
			"next_training_focus": []any{"补缓存一致性表达"},
			"gaps":                []any{"缺缓存一致性取舍"},
			"top_fix":             "补缓存一致性取舍",
			"top_fix_reason":      "这是最影响训练效果的短板",
		},
	})
	if err != nil {
		t.Fatalf("ExecuteAgentCommand() error = %v", err)
	}

	if result.Status != domain.AgentCommandStatusApplied || !result.Applied {
		t.Fatalf("expected applied result, got %+v", result)
	}
	next, ok := nextSessionValue(result.Data["recommended_next"])
	if !ok || next.Topic != domain.BasicsTopicRedis {
		t.Fatalf("expected normalized recommended next redis, got %+v", result.Data)
	}
}
