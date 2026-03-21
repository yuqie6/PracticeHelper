package sidecar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
)

func TestEvaluateAnswerSupportsEnvelopedResultPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/evaluate_answer" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set(promptSetHeader, "stable-v1")
		w.Header().Set(promptHashHeader, "hash-answer")
		w.Header().Set(modelNameHeader, "gpt-test")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"result": map[string]any{
				"score":                    82,
				"score_breakdown":          map[string]float64{"准确性": 82},
				"strengths":                []string{"主线完整"},
				"gaps":                     []string{"例子不够具体"},
				"followup_question":        "如果线上抖动，你先看什么？",
				"followup_expected_points": []string{"先止血", "再定位"},
				"weakness_hits":            []map[string]any{},
			},
			"raw_output": `{"score":82}`,
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	result, meta, err := client.EvaluateAnswer(context.Background(), domain.EvaluateAnswerRequest{
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicRedis,
		Question:  "Redis 为什么快？",
		Answer:    "因为主要走内存访问。",
		TurnIndex: 1,
		MaxTurns:  2,
	})
	if err != nil {
		t.Fatalf("EvaluateAnswer() error = %v", err)
	}

	if result.Score != 82 {
		t.Fatalf("expected score 82, got %.1f", result.Score)
	}
	if meta == nil {
		t.Fatal("expected prompt meta")
	}
	if meta.PromptSetID != "stable-v1" {
		t.Fatalf("expected prompt set stable-v1, got %q", meta.PromptSetID)
	}
	if meta.PromptHash != "hash-answer" {
		t.Fatalf("expected prompt hash hash-answer, got %q", meta.PromptHash)
	}
	if meta.ModelName != "gpt-test" {
		t.Fatalf("expected model name gpt-test, got %q", meta.ModelName)
	}
	if meta.RawOutput != `{"score":82}` {
		t.Fatalf("expected raw output to be preserved, got %q", meta.RawOutput)
	}
}

func TestGenerateReviewStreamReadsRawOutputFromResultEventEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/generate_review/stream" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set(promptSetHeader, "stable-v1")
		w.Header().Set(promptHashHeader, "hash-review")
		w.Header().Set(modelNameHeader, "gpt-test")
		w.Header().Set("Content-Type", "application/x-ndjson")

		lines := []map[string]any{
			{"type": "phase", "phase": "prepare_context"},
			{
				"type": "result",
				"data": map[string]any{
					"result": map[string]any{
						"overall":             "整体过线，但还可以再补强。",
						"top_fix":             "把取舍讲清楚",
						"top_fix_reason":      "这是说服力最缺的一刀",
						"highlights":          []string{"主线完整"},
						"gaps":                []string{"trade-off 还不够具体"},
						"suggested_topics":    []string{"redis"},
						"next_training_focus": []string{"围绕缓存一致性做专项"},
						"score_breakdown":     map[string]float64{"准确性": 84},
					},
					"raw_output": `{"overall":"整体过线，但还可以再补强。"}`,
				},
			},
		}

		for _, line := range lines {
			if err := json.NewEncoder(w).Encode(line); err != nil {
				t.Fatalf("encode stream line: %v", err)
			}
		}
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	var phases []string
	result, meta, err := client.GenerateReviewStream(
		context.Background(),
		domain.GenerateReviewRequest{
			Session: &domain.TrainingSession{
				ID:     "sess_1",
				Mode:   domain.ModeBasics,
				Topic:  domain.BasicsTopicRedis,
				Status: domain.StatusReviewPending,
			},
			Turns: []domain.TrainingTurn{
				{
					ID:        "turn_1",
					SessionID: "sess_1",
					TurnIndex: 1,
					Question:  "Redis 为什么快？",
					Answer:    "因为数据主要在内存里。",
				},
			},
		},
		func(event domain.StreamEvent) error {
			if event.Phase != "" {
				phases = append(phases, event.Phase)
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("GenerateReviewStream() error = %v", err)
	}

	if len(phases) != 1 || phases[0] != "prepare_context" {
		t.Fatalf("expected prepare_context phase, got %v", phases)
	}
	if result.Overall == "" {
		t.Fatal("expected decoded review result")
	}
	if meta == nil || meta.RawOutput == "" {
		t.Fatal("expected raw output to be captured from result event")
	}
	if meta.RawOutput != `{"overall":"整体过线，但还可以再补强。"}` {
		t.Fatalf("unexpected raw output: %q", meta.RawOutput)
	}
}
