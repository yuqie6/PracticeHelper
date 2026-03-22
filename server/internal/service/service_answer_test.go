package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/sidecar"
)

func TestSubmitAnswerRejectsBusySessionBeforeCallingSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_busy",
		Mode:      domain.ModeBasics,
		Topic:     "go",
		Intensity: "standard",
		Status:    domain.StatusEvaluating,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_busy",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 goroutine 为什么轻量？",
		ExpectedPoints: []string{"调度", "栈"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	_, err = svc.SubmitAnswer(context.Background(), session.ID, domain.SubmitAnswerRequest{Answer: "answer"})
	if err == nil {
		t.Fatal("expected SubmitAnswer() to fail")
	}
	if err != ErrSessionBusy {
		t.Fatalf("expected ErrSessionBusy, got %v", err)
	}
}

func TestSubmitAnswerRejectsReviewPendingSession(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_review_pending",
		Mode:      domain.ModeBasics,
		Topic:     "go",
		Intensity: "standard",
		Status:    domain.StatusReviewPending,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_review_pending",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 goroutine 为什么轻量？",
		ExpectedPoints: []string{"调度", "栈"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	_, err = svc.SubmitAnswer(context.Background(), session.ID, domain.SubmitAnswerRequest{Answer: "answer"})
	if err == nil {
		t.Fatal("expected SubmitAnswer() to fail")
	}
	if err != ErrSessionReviewPending {
		t.Fatalf("expected ErrSessionReviewPending, got %v", err)
	}
}

func TestSubmitAnswerEvaluatesAndCreatesFollowupTurn(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_submit_followup",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicRedis,
		Intensity: "standard",
		MaxTurns:  2,
		Status:    domain.StatusWaitingAnswer,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_submit_followup",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/evaluate_answer" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.EvaluationResult{
			Score:            84,
			ScoreBreakdown:   map[string]float64{"准确性": 84},
			Strengths:        []string{"主线清楚"},
			Gaps:             []string{"例子不够具体"},
			FollowupQuestion: "如果线上出现缓存击穿，你第一步怎么处理？",
			FollowupPoints:   []string{"先止血", "再定位根因"},
			WeaknessHits:     []domain.WeaknessHit{{Kind: "detail", Label: "缓存击穿处理", Severity: 0.8}},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	updated, err := svc.SubmitAnswer(context.Background(), session.ID, domain.SubmitAnswerRequest{
		Answer: "Redis 快主要因为数据在内存里，单线程事件循环减少上下文切换。",
	})
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}

	if updated.Status != domain.StatusWaitingAnswer {
		t.Fatalf("expected waiting_answer status, got %s", updated.Status)
	}
	if len(updated.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(updated.Turns))
	}
	if updated.TotalScore != 84 {
		t.Fatalf("expected total score 84, got %.1f", updated.TotalScore)
	}
	if updated.Turns[0].Answer == "" || updated.Turns[0].Evaluation == nil {
		t.Fatalf("expected first turn answer and evaluation to be saved, got %+v", updated.Turns[0])
	}
	if updated.Turns[1].Question != "如果线上出现缓存击穿，你第一步怎么处理？" {
		t.Fatalf("unexpected followup question: %q", updated.Turns[1].Question)
	}
	if !slices.Equal(updated.Turns[1].ExpectedPoints, []string{"先止血", "再定位根因"}) {
		t.Fatalf("unexpected followup points: %v", updated.Turns[1].ExpectedPoints)
	}

	tags, err := store.ListWeaknesses(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListWeaknesses() error = %v", err)
	}
	if len(tags) == 0 || tags[0].Label != "缓存击穿处理" {
		t.Fatalf("expected weakness tag to be saved, got %+v", tags)
	}
}

func TestSubmitAnswerFinalTurnTriggersReview(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_submit_final",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicGo,
		Intensity: "standard",
		MaxTurns:  1,
		Status:    domain.StatusWaitingAnswer,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_submit_final",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 channel 和 mutex 什么时候各用什么？",
		ExpectedPoints: []string{"共享状态", "所有权转移"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/internal/evaluate_answer":
			if err := json.NewEncoder(w).Encode(domain.EvaluationResult{
				Score:          88,
				ScoreBreakdown: map[string]float64{"准确性": 88},
				Strengths:      []string{"取舍讲清楚了"},
				Gaps:           []string{"可以补真实线上例子"},
			}); err != nil {
				t.Fatalf("encode evaluation response: %v", err)
			}
		case "/internal/generate_review":
			if err := json.NewEncoder(w).Encode(domain.ReviewCard{
				Overall:           "整体过线，但还可以补更真实的线上例子。",
				Highlights:        []string{"主线完整"},
				Gaps:              []string{"例子不够落地"},
				SuggestedTopics:   []string{domain.BasicsTopicGo},
				NextTrainingFocus: []string{"补并发场景例子"},
				ScoreBreakdown:    map[string]float64{"准确性": 86},
			}); err != nil {
				t.Fatalf("encode review response: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	updated, err := svc.SubmitAnswer(context.Background(), session.ID, domain.SubmitAnswerRequest{
		Answer: "共享状态简单时优先 mutex，消息和所有权转移更适合 channel。",
	})
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}

	if updated.Status != domain.StatusCompleted {
		t.Fatalf("expected completed status, got %s", updated.Status)
	}
	if updated.ReviewID == "" {
		t.Fatal("expected review id to be saved")
	}
	review, err := store.GetReview(context.Background(), updated.ReviewID)
	if err != nil {
		t.Fatalf("GetReview() error = %v", err)
	}
	if review == nil || review.Overall == "" {
		t.Fatalf("expected persisted review, got %+v", review)
	}
}

func TestSubmitAnswerRestoresStatusOnSidecarFailure(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_submit_restore",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicRedis,
		Intensity: "standard",
		MaxTurns:  2,
		Status:    domain.StatusWaitingAnswer,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_submit_restore",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary upstream failure", http.StatusServiceUnavailable)
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	_, err = svc.SubmitAnswer(context.Background(), session.ID, domain.SubmitAnswerRequest{
		Answer: "因为主要访问内存。",
	})
	if err == nil {
		t.Fatal("expected SubmitAnswer() to fail")
	}

	saved, err := store.GetSession(context.Background(), session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved session")
	}
	if saved.Status != domain.StatusWaitingAnswer {
		t.Fatalf("expected session status to be restored, got %s", saved.Status)
	}
}

func TestSubmitAnswerRejectsCompletedSession(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_submit_completed",
		Mode:      domain.ModeBasics,
		Topic:     domain.BasicsTopicGo,
		Intensity: "standard",
		Status:    domain.StatusCompleted,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_submit_completed",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 goroutine 为什么轻量？",
		ExpectedPoints: []string{"调度", "栈"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	svc := New(store, nil)
	_, err = svc.SubmitAnswer(context.Background(), session.ID, domain.SubmitAnswerRequest{Answer: "answer"})
	if err == nil {
		t.Fatal("expected SubmitAnswer() to fail")
	}
	if err != ErrSessionCompleted {
		t.Fatalf("expected ErrSessionCompleted, got %v", err)
	}
}

func TestSubmitAnswerPassesTemplateScoreWeightsToSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target := seedReadyJobTarget(t, store)

	session := &domain.TrainingSession{
		ID:                  "sess_score_weights",
		Mode:                domain.ModeBasics,
		Topic:               "redis",
		JobTargetID:         target.ID,
		JobTargetAnalysisID: target.LatestAnalysisID,
		Intensity:           "standard",
		MaxTurns:            2,
		Status:              domain.StatusWaitingAnswer,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_score_weights",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环", "高效数据结构"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	var captured domain.EvaluateAnswerRequest
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/evaluate_answer" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.EvaluationResult{
			Score:            82,
			ScoreBreakdown:   map[string]float64{"准确性": 80},
			Strengths:        []string{"覆盖了核心点"},
			Gaps:             []string{"可以补更多线上例子"},
			FollowupQuestion: "Redis 6 的多线程多在哪里？",
			FollowupPoints:   []string{"网络读写线程", "命令执行仍是主线程"},
			WeaknessHits:     []domain.WeaknessHit{{Kind: "detail", Label: "redis", Severity: 0.4}},
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	_, err = svc.SubmitAnswer(context.Background(), session.ID, domain.SubmitAnswerRequest{
		Answer: "Redis 快主要因为数据在内存、事件循环和数据结构效率。",
	})
	if err != nil {
		t.Fatalf("SubmitAnswer() error = %v", err)
	}

	want := map[string]float64{
		"准确性":   30,
		"完整性":   25,
		"落地感":   15,
		"表达清晰度": 15,
		"抗追问能力": 15,
	}
	if len(captured.ScoreWeights) != len(want) {
		t.Fatalf("unexpected score weights length: got %d want %d", len(captured.ScoreWeights), len(want))
	}
	for key, value := range want {
		if captured.ScoreWeights[key] != value {
			t.Fatalf("unexpected score weight for %s: got %v want %v", key, captured.ScoreWeights[key], value)
		}
	}
	if captured.JobTargetAnalysis == nil {
		t.Fatal("expected evaluate answer request to include job target analysis")
	}
	if len(captured.JobTargetAnalysis.EvaluationFocus) == 0 {
		t.Fatal("expected evaluation focus to be forwarded to sidecar")
	}
}

func TestSubmitAnswerStreamEmitsStatusSequenceForMainAnswer(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_stream_main",
		Mode:      domain.ModeBasics,
		Topic:     "go",
		Intensity: "standard",
		MaxTurns:  2,
		Status:    domain.StatusWaitingAnswer,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_stream_main",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 channel 和 mutex 什么时候各用什么？",
		ExpectedPoints: []string{"共享内存", "所有权转移"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/evaluate_answer/stream" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"}); err != nil {
			t.Fatalf("encode phase event: %v", err)
		}
		if err := encoder.Encode(domain.StreamEvent{
			Type: "result",
			Data: domain.EvaluationResult{
				Score:            81,
				ScoreBreakdown:   map[string]float64{"准确性": 80},
				Strengths:        []string{"有结论"},
				Gaps:             []string{"例子不够具体"},
				FollowupQuestion: "那你在项目里什么时候会避免过度用 channel？",
				FollowupPoints:   []string{"性能", "复杂度"},
			},
		}); err != nil {
			t.Fatalf("encode result event: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))

	var events []domain.StreamEvent
	updated, err := svc.SubmitAnswerStream(
		context.Background(),
		session.ID,
		domain.SubmitAnswerRequest{Answer: "channel 更适合消息和所有权转移，mutex 更适合保护共享状态。"},
		func(event domain.StreamEvent) error {
			events = append(events, event)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("SubmitAnswerStream() error = %v", err)
	}

	if updated.Status != domain.StatusWaitingAnswer {
		t.Fatalf("expected waiting_answer status, got %s", updated.Status)
	}
	if len(updated.Turns) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(updated.Turns))
	}
	if updated.Turns[1].Question != "那你在项目里什么时候会避免过度用 channel？" {
		t.Fatalf("expected followup question as turn 2, got %q", updated.Turns[1].Question)
	}

	assertStatusEventNames(t, events, []string{
		"answer_received",
		"evaluation_started",
		"feedback_ready",
		"answer_saved",
		"followup_ready",
	})
}

func TestSubmitAnswerStreamEmitsStatusSequenceForLastTurn(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:         "sess_stream_last",
		Mode:       domain.ModeBasics,
		Topic:      "go",
		Intensity:  "standard",
		MaxTurns:   2,
		Status:     domain.StatusWaitingAnswer,
		TotalScore: 72,
	}
	turn1 := &domain.TrainingTurn{
		ID:             "turn_stream_last_1",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Go 的 channel 和 mutex 什么时候各用什么？",
		ExpectedPoints: []string{"共享内存", "所有权转移"},
		Answer:         "主回答已经提交",
	}
	if err := store.CreateSession(context.Background(), session, turn1); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	turn2 := &domain.TrainingTurn{
		ID:             "turn_stream_last_2",
		SessionID:      session.ID,
		TurnIndex:      2,
		Stage:          "question",
		Question:       "那你在项目里什么时候会避免过度用 channel？",
		ExpectedPoints: []string{"性能", "复杂度"},
	}
	if err := store.InsertTurn(context.Background(), turn2); err != nil {
		t.Fatalf("InsertTurn() error = %v", err)
	}

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)

		switch r.URL.Path {
		case "/internal/evaluate_answer/stream":
			if err := encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"}); err != nil {
				t.Fatalf("encode phase event: %v", err)
			}
			if err := encoder.Encode(domain.StreamEvent{
				Type: "result",
				Data: domain.EvaluationResult{
					Score:          86,
					ScoreBreakdown: map[string]float64{"准确性": 85},
					Strengths:      []string{"解释到了取舍"},
					Gaps:           []string{"可以补线上场景"},
				},
			}); err != nil {
				t.Fatalf("encode evaluation result event: %v", err)
			}
		case "/internal/generate_review/stream":
			if err := encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"}); err != nil {
				t.Fatalf("encode review phase event: %v", err)
			}
			if err := encoder.Encode(domain.StreamEvent{
				Type: "result",
				Data: domain.ReviewCard{
					Overall:           "本轮回答整体过线，但例子还能更贴近真实线上场景。",
					Highlights:        []string{"主线判断清楚"},
					Gaps:              []string{"例子不够落地"},
					SuggestedTopics:   []string{"并发控制"},
					NextTrainingFocus: []string{"把取舍讲得更具体"},
					ScoreBreakdown:    map[string]float64{"准确性": 84},
				},
			}); err != nil {
				t.Fatalf("encode review result event: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))

	var events []domain.StreamEvent
	updated, err := svc.SubmitAnswerStream(
		context.Background(),
		session.ID,
		domain.SubmitAnswerRequest{Answer: "我会在共享状态简单且性能敏感时优先 mutex，而不是为了 channel 而 channel。"},
		func(event domain.StreamEvent) error {
			events = append(events, event)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("SubmitAnswerStream() error = %v", err)
	}

	if updated.Status != domain.StatusCompleted {
		t.Fatalf("expected completed status, got %s", updated.Status)
	}
	if updated.ReviewID == "" {
		t.Fatal("expected review id to be saved")
	}

	assertStatusEventNames(t, events, []string{
		"answer_received",
		"evaluation_started",
		"feedback_ready",
		"answer_saved",
		"review_started",
		"review_saved",
	})
}

func TestSubmitAnswerStreamAppliesDepthSignalFromStreamSideEffects(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_stream_extend",
		Mode:      domain.ModeBasics,
		Topic:     "redis",
		Intensity: "standard",
		MaxTurns:  1,
		Status:    domain.StatusWaitingAnswer,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_stream_extend_1",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/evaluate_answer/stream" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		if err := encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"}); err != nil {
			t.Fatalf("encode phase event: %v", err)
		}
		if err := encoder.Encode(domain.StreamEvent{
			Type: "result",
			Data: map[string]any{
				"result": map[string]any{
					"score":                    82,
					"score_breakdown":          map[string]float64{"准确性": 82},
					"strengths":                []string{"主线完整"},
					"gaps":                     []string{"案例不够具体"},
					"followup_question":        "如果线上抖动，你先看什么？",
					"followup_expected_points": []string{"先止血", "再定位"},
					"weakness_hits":            []map[string]any{},
				},
				"side_effects": map[string]any{
					"depth_signal": "extend",
				},
				"raw_output": `{"score":82}`,
			},
		}); err != nil {
			t.Fatalf("encode result event: %v", err)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	updated, err := svc.SubmitAnswerStream(
		context.Background(),
		session.ID,
		domain.SubmitAnswerRequest{Answer: "因为它主要在内存里，单线程也减少了锁竞争。"},
		nil,
	)
	if err != nil {
		t.Fatalf("SubmitAnswerStream() error = %v", err)
	}

	if updated.Status != domain.StatusWaitingAnswer {
		t.Fatalf("expected waiting_answer status after extend, got %s", updated.Status)
	}
	if updated.MaxTurns != 2 {
		t.Fatalf("expected max turns to extend to 2, got %d", updated.MaxTurns)
	}
	if len(updated.Turns) != 2 {
		t.Fatalf("expected a followup turn after extend, got %d turns", len(updated.Turns))
	}
	if updated.ReviewID != "" {
		t.Fatalf("expected no review to be created after extend, got %s", updated.ReviewID)
	}
}

func TestSubmitAnswerStreamPersistsReviewSideEffects(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:        "sess_stream_review_side_effects",
		Mode:      domain.ModeBasics,
		Topic:     "redis",
		Intensity: "standard",
		MaxTurns:  1,
		Status:    domain.StatusWaitingAnswer,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_stream_review_side_effects_1",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环"},
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)

		switch r.URL.Path {
		case "/internal/evaluate_answer/stream":
			if err := encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"}); err != nil {
				t.Fatalf("encode evaluation phase event: %v", err)
			}
			if err := encoder.Encode(domain.StreamEvent{
				Type: "result",
				Data: map[string]any{
					"result": map[string]any{
						"score":           84,
						"score_breakdown": map[string]float64{"准确性": 84},
						"strengths":       []string{"主线完整"},
						"gaps":            []string{"案例不够具体"},
					},
					"raw_output": `{"score":84}`,
				},
			}); err != nil {
				t.Fatalf("encode evaluation result event: %v", err)
			}
		case "/internal/generate_review/stream":
			if err := encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"}); err != nil {
				t.Fatalf("encode review phase event: %v", err)
			}
			if err := encoder.Encode(domain.StreamEvent{
				Type: "result",
				Data: map[string]any{
					"result": map[string]any{
						"overall":             "整体过线，但缓存一致性表达还不够硬。",
						"top_fix":             "补缓存一致性取舍",
						"top_fix_reason":      "这是这轮最缺说服力的地方",
						"highlights":          []string{"主线完整"},
						"gaps":                []string{"缺缓存一致性取舍"},
						"suggested_topics":    []string{"redis"},
						"next_training_focus": []string{"补缓存一致性表达"},
						"score_breakdown":     map[string]float64{"准确性": 84},
					},
					"side_effects": map[string]any{
						"recommended_next": map[string]any{
							"mode":   "basics",
							"topic":  "redis",
							"reason": "先补缓存一致性取舍",
						},
						"observations": []map[string]any{
							{
								"category":  "pattern",
								"content":   "用户主线能站住，但 trade-off 细节还不够具体。",
								"tags":      []string{"redis", "tradeoff"},
								"relevance": 0.9,
								"topic":     "redis",
							},
						},
					},
					"raw_output": `{"overall":"整体过线，但缓存一致性表达还不够硬。"}`,
				},
			}); err != nil {
				t.Fatalf("encode review result event: %v", err)
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer sidecarServer.Close()

	svc := New(store, sidecar.New(sidecarServer.URL, time.Second))
	updated, err := svc.SubmitAnswerStream(
		context.Background(),
		session.ID,
		domain.SubmitAnswerRequest{Answer: "因为 Redis 主要走内存访问，事件循环也减少了线程切换。"},
		nil,
	)
	if err != nil {
		t.Fatalf("SubmitAnswerStream() error = %v", err)
	}

	review, err := store.GetReview(context.Background(), updated.ReviewID)
	if err != nil {
		t.Fatalf("GetReview() error = %v", err)
	}
	if review == nil || review.RecommendedNext == nil {
		t.Fatalf("expected recommended next to be persisted, got %#v", review)
	}
	if review.RecommendedNext.Topic != domain.BasicsTopicRedis {
		t.Fatalf("expected recommended next topic redis, got %#v", review.RecommendedNext)
	}

	observations, err := store.ListRelevantObservations(
		context.Background(),
		session.ID,
		"",
		"",
		domain.BasicsTopicRedis,
		5,
	)
	if err != nil {
		t.Fatalf("ListRelevantObservations() error = %v", err)
	}
	if len(observations) == 0 {
		t.Fatal("expected review observations to be persisted")
	}
	if observations[0].Content != "用户主线能站住，但 trade-off 细节还不够具体。" {
		t.Fatalf("unexpected observation content: %#v", observations[0])
	}
}
