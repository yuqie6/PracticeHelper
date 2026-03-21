package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
	"practicehelper/server/internal/sidecar"
)

func TestTrainingRoutesSmokeThroughStubbedSidecar(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	sidecarServer := newStubbedSidecarServer(t)
	defer sidecarServer.Close()

	router := NewRouter(service.New(store, sidecar.New(sidecarServer.URL, 2*time.Second)))

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/sessions/stream",
		strings.NewReader(`{"mode":"basics","topic":"go","intensity":"standard","max_turns":2}`),
	)
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)

	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createRecorder.Code, createRecorder.Body.String())
	}

	createEvents := decodeStreamEvents(t, createRecorder.Body.Bytes())
	if len(createEvents) == 0 || createEvents[0].Phase != "prepare_context" {
		t.Fatalf("expected create stream phase event, got %#v", createEvents)
	}
	session := decodeStreamResult[domain.TrainingSession](t, createEvents)
	if session.Status != domain.StatusWaitingAnswer {
		t.Fatalf("expected waiting_answer session, got %q", session.Status)
	}

	firstAnswerRecorder := httptest.NewRecorder()
	firstAnswerRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/sessions/"+session.ID+"/answer/stream",
		strings.NewReader(`{"answer":"第一轮先讲 interface 和 nil 的区别。"}`),
	)
	firstAnswerRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(firstAnswerRecorder, firstAnswerRequest)

	if firstAnswerRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", firstAnswerRecorder.Code, firstAnswerRecorder.Body.String())
	}

	firstAnswerEvents := decodeStreamEvents(t, firstAnswerRecorder.Body.Bytes())
	waitingSession := decodeStreamResult[domain.TrainingSession](t, firstAnswerEvents)
	if waitingSession.Status != domain.StatusWaitingAnswer {
		t.Fatalf("expected followup session waiting_answer, got %q", waitingSession.Status)
	}
	if !containsStatusEvent(firstAnswerEvents, "followup_ready") {
		t.Fatalf("expected followup_ready status, got %#v", firstAnswerEvents)
	}

	secondAnswerRecorder := httptest.NewRecorder()
	secondAnswerRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/sessions/"+session.ID+"/answer/stream",
		strings.NewReader(`{"answer":"第二轮补 typed nil 的真实坑点。"}`),
	)
	secondAnswerRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(secondAnswerRecorder, secondAnswerRequest)

	if secondAnswerRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", secondAnswerRecorder.Code, secondAnswerRecorder.Body.String())
	}

	secondAnswerEvents := decodeStreamEvents(t, secondAnswerRecorder.Body.Bytes())
	completedSession := decodeStreamResult[domain.TrainingSession](t, secondAnswerEvents)
	if completedSession.Status != domain.StatusCompleted {
		t.Fatalf("expected completed session, got %q", completedSession.Status)
	}
	if completedSession.ReviewID == "" {
		t.Fatal("expected review id after second answer")
	}
	if !containsStatusEvent(secondAnswerEvents, "review_saved") {
		t.Fatalf("expected review_saved status, got %#v", secondAnswerEvents)
	}

	reviewRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		reviewRecorder,
		httptest.NewRequest(http.MethodGet, "/api/reviews/"+completedSession.ReviewID, nil),
	)
	if reviewRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", reviewRecorder.Code, reviewRecorder.Body.String())
	}
	review := decodeDataEnvelope[domain.ReviewCard](t, reviewRecorder.Body.Bytes())
	if review.TopFix == "" {
		t.Fatalf("expected populated review top fix, got %#v", review)
	}

	logRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		logRecorder,
		httptest.NewRequest(http.MethodGet, "/api/sessions/"+session.ID+"/evaluation-logs", nil),
	)
	if logRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", logRecorder.Code, logRecorder.Body.String())
	}
	logs := decodeDataEnvelope[[]domain.EvaluationLogEntry](t, logRecorder.Body.Bytes())
	if len(logs) < 4 {
		t.Fatalf("expected at least 4 evaluation logs, got %d", len(logs))
	}

	sessionRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		sessionRecorder,
		httptest.NewRequest(http.MethodGet, "/api/sessions/"+session.ID, nil),
	)
	if sessionRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", sessionRecorder.Code, sessionRecorder.Body.String())
	}

	weaknessRecorder := httptest.NewRecorder()
	router.ServeHTTP(weaknessRecorder, httptest.NewRequest(http.MethodGet, "/api/weaknesses", nil))
	if weaknessRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", weaknessRecorder.Code, weaknessRecorder.Body.String())
	}
	weaknesses := decodeDataEnvelope[[]domain.WeaknessTag](t, weaknessRecorder.Body.Bytes())
	if len(weaknesses) == 0 {
		t.Fatal("expected weakness tags after answer evaluation")
	}

	if err := store.CreateReviewSchedule(context.Background(), &domain.ReviewScheduleItem{
		SessionID:     session.ID,
		ReviewCardID:  completedSession.ReviewID,
		WeaknessTagID: weaknesses[0].ID,
		WeaknessKind:  weaknesses[0].Kind,
		WeaknessLabel: weaknesses[0].Label,
		Topic:         domain.BasicsTopicGo,
		NextReviewAt:  time.Now().UTC().Add(-time.Minute),
		IntervalDays:  1,
		EaseFactor:    2.5,
	}); err != nil {
		t.Fatalf("CreateReviewSchedule() error = %v", err)
	}

	trendsRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		trendsRecorder,
		httptest.NewRequest(http.MethodGet, "/api/weaknesses/trends", nil),
	)
	if trendsRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", trendsRecorder.Code, trendsRecorder.Body.String())
	}

	if err := store.CreateReviewSchedule(context.Background(), &domain.ReviewScheduleItem{
		SessionID:    session.ID,
		ReviewCardID: completedSession.ReviewID,
		Topic:        domain.BasicsTopicGo,
		NextReviewAt: time.Now().UTC().Add(-time.Minute),
		IntervalDays: 1,
		EaseFactor:   2.5,
	}); err != nil {
		t.Fatalf("CreateReviewSchedule() error = %v", err)
	}

	dueRecorder := httptest.NewRecorder()
	router.ServeHTTP(dueRecorder, httptest.NewRequest(http.MethodGet, "/api/reviews/due", nil))
	if dueRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", dueRecorder.Code, dueRecorder.Body.String())
	}
	due := decodeDataEnvelope[[]domain.ReviewScheduleItem](t, dueRecorder.Body.Bytes())
	if len(due) == 0 {
		t.Fatal("expected due review items after review generation")
	}

	completeRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		completeRecorder,
		httptest.NewRequest(
			http.MethodPost,
			"/api/reviews/due/"+strconvFormatInt(due[0].ID)+"/complete",
			nil,
		),
	)
	if completeRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", completeRecorder.Code, completeRecorder.Body.String())
	}
	result := decodeDataEnvelope[string](t, completeRecorder.Body.Bytes())
	if result != "ok" {
		t.Fatalf("expected ok response, got %q", result)
	}
}

func newStubbedSidecarServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	mux.HandleFunc("/internal/prompt-sets", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]domain.PromptSetSummary{
			{ID: "stable-v1", Label: "Stable v1", Status: "stable", IsDefault: true},
		})
	})

	mux.HandleFunc("/internal/generate_question/stream", func(w http.ResponseWriter, _ *http.Request) {
		setStubPromptHeaders(w.Header(), "stable-v1", "hash-question", "stub-model")
		w.Header().Set("Content-Type", "application/x-ndjson")
		encoder := json.NewEncoder(w)
		_ = encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "prepare_context"})
		_ = encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"})
		_ = encoder.Encode(domain.StreamEvent{
			Type: "result",
			Data: map[string]any{
				"result": domain.GenerateQuestionResponse{
					Question:       "请解释 interface 的零值陷阱。",
					ExpectedPoints: []string{"nil 接口", "动态类型和值类型的差别"},
				},
				"raw_output": `{"question":"请解释 interface 的零值陷阱。"}`,
			},
		})
	})

	mux.HandleFunc("/internal/evaluate_answer/stream", func(w http.ResponseWriter, r *http.Request) {
		setStubPromptHeaders(w.Header(), "stable-v1", "hash-evaluate", "stub-model")
		w.Header().Set("Content-Type", "application/x-ndjson")

		var request domain.EvaluateAnswerRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		result := domain.EvaluationResult{
			Score:          86,
			ScoreBreakdown: map[string]float64{"准确性": 86},
			Headline:       "第二轮补充后明显更稳。",
			Strengths:      []string{"主线清楚"},
			Gaps:           []string{"案例还能再具体"},
			Suggestion:     "继续补一个真实线上例子。",
			WeaknessHits: []domain.WeaknessHit{
				{Kind: "depth", Label: "interface nil 语义", Severity: 0.4},
			},
		}
		rawOutput := `{"score":86,"headline":"第二轮补充后明显更稳。"}`
		if request.TurnIndex == 1 {
			result.Score = 78
			result.ScoreBreakdown = map[string]float64{"准确性": 78}
			result.Headline = "第一轮主线还行，但深度不够。"
			result.Suggestion = "补 typed nil 的实际坑点。"
			result.FollowupIntent = "depth"
			result.FollowupQuestion = "那 typed nil 在接口里为什么容易踩坑？"
			result.FollowupPoints = []string{"接口为 nil 和底层值为 nil 的差别"}
			result.WeaknessHits = []domain.WeaknessHit{
				{Kind: "depth", Label: "interface nil 语义", Severity: 0.8},
			}
			rawOutput = `{"score":78,"followup_question":"那 typed nil 在接口里为什么容易踩坑？"}`
		}

		encoder := json.NewEncoder(w)
		_ = encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "call_model"})
		_ = encoder.Encode(domain.StreamEvent{
			Type: "result",
			Data: map[string]any{
				"result":     result,
				"raw_output": rawOutput,
			},
		})
	})

	mux.HandleFunc("/internal/generate_review/stream", func(w http.ResponseWriter, _ *http.Request) {
		setStubPromptHeaders(w.Header(), "stable-v1", "hash-review", "stub-model")
		w.Header().Set("Content-Type", "application/x-ndjson")

		encoder := json.NewEncoder(w)
		_ = encoder.Encode(domain.StreamEvent{Type: "phase", Phase: "parse_result"})
		_ = encoder.Encode(domain.StreamEvent{
			Type: "result",
			Data: map[string]any{
				"result": domain.ReviewCard{
					Overall:           "主线完整，typed nil 的细节已经补上。",
					TopFix:            "再准备一个真实线上案例。",
					TopFixReason:      "这样项目题会更有说服力。",
					Highlights:        []string{"主线清楚"},
					Gaps:              []string{"线上例子还不够具体"},
					SuggestedTopics:   []string{"go"},
					NextTrainingFocus: []string{"interface 细节"},
					RecommendedNext: &domain.NextSession{
						Mode:   domain.ModeBasics,
						Topic:  domain.BasicsTopicGo,
						Reason: "继续巩固 interface 追问。",
					},
					ScoreBreakdown: map[string]float64{"准确性": 82},
				},
				"raw_output": `{"overall":"主线完整，typed nil 的细节已经补上。"}`,
			},
		})
	})

	return httptest.NewServer(mux)
}

func setStubPromptHeaders(headers http.Header, promptSetID, promptHash, modelName string) {
	headers.Set("X-PracticeHelper-Prompt-Set", promptSetID)
	headers.Set("X-PracticeHelper-Prompt-Hash", promptHash)
	headers.Set("X-PracticeHelper-Model-Name", modelName)
}

func containsStatusEvent(events []streamPayload, name string) bool {
	for _, event := range events {
		if event.Type == "status" && event.Name == name {
			return true
		}
	}
	return false
}

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}
