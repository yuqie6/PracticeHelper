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
			"side_effects": map[string]any{
				"depth_signal": "extend",
			},
			"command_results": []map[string]any{
				{
					"command_id": "cmd_transition_session_turn_1_extend",
					"status":     "deferred",
					"data": map[string]any{
						"resolved_depth_signal": "extend",
						"resolved_max_turns":    3,
					},
				},
			},
			"raw_output": `{"score":82}`,
			"trace": map[string]any{
				"entries": []map[string]any{
					{
						"flow":    "evaluate_answer",
						"phase":   "finalize",
						"status":  "success",
						"code":    "runtime_completed",
						"message": "agent runtime 已稳定收口。",
					},
				},
			},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	result, sideEffects, commandResults, meta, err := client.EvaluateAnswer(context.Background(), domain.EvaluateAnswerRequest{
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
	if meta.RuntimeTrace == nil || len(meta.RuntimeTrace.Entries) != 1 {
		t.Fatalf("expected runtime trace to be preserved, got %+v", meta.RuntimeTrace)
	}
	if sideEffects == nil || sideEffects.DepthSignal != domain.DepthSignalExtend {
		t.Fatalf("expected depth signal extend, got %+v", sideEffects)
	}
	if len(commandResults) != 1 || commandResults[0].Status != domain.AgentCommandStatusDeferred {
		t.Fatalf("expected deferred command result, got %+v", commandResults)
	}
}

func TestEmbedMemoryDecodesPlainResponsePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/embed_memory" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.EmbedMemoryResponse{
			Items: []domain.EmbeddedMemoryVector{{
				ID:        "memidx_1",
				Vector:    []float64{0.1, 0.2, 0.3},
				ModelName: "embed-test",
			}},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	response, err := client.EmbedMemory(context.Background(), domain.EmbedMemoryRequest{
		Items: []domain.EmbedMemoryItem{{ID: "memidx_1", Text: "redis cache consistency"}},
	})
	if err != nil {
		t.Fatalf("EmbedMemory() error = %v", err)
	}
	if len(response.Items) != 1 || len(response.Items[0].Vector) != 3 {
		t.Fatalf("unexpected embed response: %+v", response)
	}
}

func TestRerankMemoryDecodesPlainResponsePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/rerank_memory" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.RerankMemoryResponse{
			Items: []domain.RerankMemoryResult{{
				ID:    "memidx_1",
				Score: 0.92,
				Rank:  1,
			}},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	response, err := client.RerankMemory(context.Background(), domain.RerankMemoryRequest{
		Query: "redis consistency tradeoff",
		Candidates: []domain.RerankMemoryCandidate{{
			ID:   "memidx_1",
			Text: "redis cache consistency answer",
		}},
		TopK: 1,
	})
	if err != nil {
		t.Fatalf("RerankMemory() error = %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].ID != "memidx_1" {
		t.Fatalf("unexpected rerank response: %+v", response)
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
				"type": "trace",
				"data": map[string]any{
					"flow":    "generate_review",
					"phase":   "fallback",
					"status":  "fallback",
					"code":    "single_shot_failed",
					"message": "工具循环没有稳定收口，正在退回单轮生成。",
				},
			},
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
					"side_effects": map[string]any{
						"recommended_next": map[string]any{
							"mode":   "basics",
							"topic":  "redis",
							"reason": "先补缓存一致性取舍",
						},
					},
					"raw_output": `{"overall":"整体过线，但还可以再补强。"}`,
					"trace": map[string]any{
						"entries": []map[string]any{
							{
								"flow":    "generate_review",
								"phase":   "finalize",
								"status":  "success",
								"code":    "runtime_completed",
								"message": "single-shot fallback 已成功收口。",
							},
						},
					},
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
	result, sideEffects, _, meta, err := client.GenerateReviewStream(
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
			if event.Type == "trace" {
				if payload, ok := event.Data.(map[string]any); ok {
					phases = append(phases, payload["phase"].(string))
				}
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("GenerateReviewStream() error = %v", err)
	}

	if len(phases) != 2 || phases[0] != "prepare_context" || phases[1] != "fallback" {
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
	if meta.RuntimeTrace == nil || len(meta.RuntimeTrace.Entries) != 1 {
		t.Fatalf("expected runtime trace, got %+v", meta.RuntimeTrace)
	}
	if sideEffects == nil || sideEffects.RecommendedNext == nil {
		t.Fatalf("expected recommended_next side effects, got %+v", sideEffects)
	}
	if sideEffects.RecommendedNext.Topic != domain.BasicsTopicRedis {
		t.Fatalf("expected side effect topic redis, got %+v", sideEffects.RecommendedNext)
	}
}

func TestEvaluateAnswerStreamReadsSideEffectsFromResultEventEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/evaluate_answer/stream" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set(promptSetHeader, "stable-v1")
		w.Header().Set(promptHashHeader, "hash-answer-stream")
		w.Header().Set(modelNameHeader, "gpt-test")
		w.Header().Set("Content-Type", "application/x-ndjson")

		lines := []map[string]any{
			{"type": "phase", "phase": "call_model"},
			{
				"type": "result",
				"data": map[string]any{
					"result": map[string]any{
						"score":                    79,
						"score_breakdown":          map[string]float64{"准确性": 79},
						"strengths":                []string{"主线完整"},
						"gaps":                     []string{"细节不够具体"},
						"followup_question":        "如果线上抖动，你先看什么？",
						"followup_expected_points": []string{"先止血", "再定位"},
						"weakness_hits":            []map[string]any{},
					},
					"side_effects": map[string]any{
						"depth_signal": "extend",
					},
					"raw_output": `{"score":79}`,
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
	result, sideEffects, _, meta, err := client.EvaluateAnswerStream(
		context.Background(),
		domain.EvaluateAnswerRequest{
			Mode:      domain.ModeBasics,
			Topic:     domain.BasicsTopicRedis,
			Question:  "Redis 为什么快？",
			Answer:    "因为数据主要在内存里。",
			TurnIndex: 1,
			MaxTurns:  2,
		},
		nil,
	)
	if err != nil {
		t.Fatalf("EvaluateAnswerStream() error = %v", err)
	}

	if result.Score != 79 {
		t.Fatalf("expected score 79, got %.1f", result.Score)
	}
	if meta == nil || meta.RawOutput != `{"score":79}` {
		t.Fatalf("unexpected prompt meta: %+v", meta)
	}
	if sideEffects == nil || sideEffects.DepthSignal != domain.DepthSignalExtend {
		t.Fatalf("expected depth signal extend, got %+v", sideEffects)
	}
}

func TestEvaluateAnswerRetriesOnRetryableStatus(t *testing.T) {
	originalBackoffs := sidecarRetryBackoffs
	originalJitter := sidecarRetryJitter
	sidecarRetryBackoffs = []time.Duration{0, 0}
	sidecarRetryJitter = 0
	defer func() {
		sidecarRetryBackoffs = originalBackoffs
		sidecarRetryJitter = originalJitter
	}()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.EvaluationResult{
			Score:          81,
			ScoreBreakdown: map[string]float64{"准确性": 81},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	result, _, _, _, err := client.EvaluateAnswer(context.Background(), domain.EvaluateAnswerRequest{
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicRedis,
		Question:  "Redis 为什么快？",
		Answer:    "因为热点数据主要在内存。",
		TurnIndex: 1,
		MaxTurns:  2,
	})
	if err != nil {
		t.Fatalf("EvaluateAnswer() error = %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if result.Score != 81 {
		t.Fatalf("expected score 81 after retry, got %.1f", result.Score)
	}
}

func TestGenerateReviewStreamRetriesBeforeConsumingBody(t *testing.T) {
	originalBackoffs := sidecarRetryBackoffs
	originalJitter := sidecarRetryJitter
	sidecarRetryBackoffs = []time.Duration{0, 0}
	sidecarRetryJitter = 0
	defer func() {
		sidecarRetryBackoffs = originalBackoffs
		sidecarRetryJitter = originalJitter
	}()

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "busy", http.StatusBadGateway)
			return
		}

		w.Header().Set("Content-Type", "application/x-ndjson")
		if err := json.NewEncoder(w).Encode(map[string]any{
			"type": "result",
			"data": map[string]any{
				"result": map[string]any{
					"overall":             "重试后拿到了复盘。",
					"highlights":          []string{"主线完整"},
					"gaps":                []string{"例子不够具体"},
					"suggested_topics":    []string{"redis"},
					"next_training_focus": []string{"补真实案例"},
					"score_breakdown":     map[string]float64{"准确性": 80},
				},
				"raw_output": `{"overall":"重试后拿到了复盘。"}`,
			},
		}); err != nil {
			t.Fatalf("encode stream result: %v", err)
		}
	}))
	defer server.Close()

	client := New(server.URL, time.Second)
	result, _, _, meta, err := client.GenerateReviewStream(
		context.Background(),
		domain.GenerateReviewRequest{
			Session: &domain.TrainingSession{
				ID:     "sess_retry_stream",
				Mode:   domain.ModeBasics,
				Topic:  domain.BasicsTopicRedis,
				Status: domain.StatusReviewPending,
			},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("GenerateReviewStream() error = %v", err)
	}

	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if result.Overall != "重试后拿到了复盘。" {
		t.Fatalf("unexpected review overall: %q", result.Overall)
	}
	if meta == nil || meta.RawOutput != `{"overall":"重试后拿到了复盘。"}` {
		t.Fatalf("unexpected prompt meta: %+v", meta)
	}
}
