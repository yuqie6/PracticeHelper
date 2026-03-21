package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/infra/sqlite"
	"practicehelper/server/internal/repo"
	"practicehelper/server/internal/service"
)

func TestExportSessionRouteReturnsMarkdownAttachment(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedExportRouteSession(t, store, "sess_route_export", true)
	router := NewRouter(service.New(store, nil))

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/sessions/"+session.ID+"/export?format=markdown",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.HasPrefix(recorder.Header().Get("Content-Type"), "text/markdown") {
		t.Fatalf("unexpected content type: %s", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("Content-Disposition") != `attachment; filename="practicehelper-session-sess_route_export.md"` {
		t.Fatalf("unexpected content disposition: %s", recorder.Header().Get("Content-Disposition"))
	}
	if !strings.Contains(recorder.Body.String(), "## 最终复盘") {
		t.Fatalf("expected markdown body, got:\n%s", recorder.Body.String())
	}
}

func TestExportSessionRouteRejectsInvalidFormat(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedExportRouteSession(t, store, "sess_route_invalid", true)
	router := NewRouter(service.New(store, nil))

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/sessions/"+session.ID+"/export?format=json",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	payload := decodeErrorPayload(t, recorder.Body.Bytes())
	if payload.Error.Code != "invalid_export_format" {
		t.Fatalf("unexpected error code: %s", payload.Error.Code)
	}
}

func TestExportSessionRouteReturnsNotFoundForMissingSession(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	router := NewRouter(service.New(store, nil))

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/sessions/missing/export?format=markdown",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	payload := decodeErrorPayload(t, recorder.Body.Bytes())
	if payload.Error.Code != "session_not_found" {
		t.Fatalf("unexpected error code: %s", payload.Error.Code)
	}
}

type errorPayload struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeErrorPayload(t *testing.T, body []byte) errorPayload {
	t.Helper()

	var payload errorPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return payload
}

func seedExportRouteSession(
	t *testing.T,
	store *repo.Store,
	sessionID string,
	withReview bool,
) *domain.TrainingSession {
	t.Helper()

	ctx := context.Background()
	target := seedRouteJobTarget(t, store)
	startedAt := time.Date(2026, 3, 21, 11, 0, 0, 0, time.UTC)
	endedAt := startedAt.Add(15 * time.Minute)

	reviewID := ""
	status := domain.StatusReviewPending
	if withReview {
		reviewID = "review_" + sessionID
		status = domain.StatusCompleted
	}

	session := &domain.TrainingSession{
		ID:                  sessionID,
		Mode:                domain.ModeBasics,
		Topic:               "redis",
		JobTargetID:         target.ID,
		JobTargetAnalysisID: target.LatestSuccessfulAnalysis.ID,
		Intensity:           "standard",
		Status:              status,
		MaxTurns:            2,
		TotalScore:          80,
		StartedAt:           &startedAt,
		EndedAt:             &endedAt,
		ReviewID:            reviewID,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_1_" + sessionID,
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问", "事件循环"},
		Answer:         "因为大量操作都在内存里完成。",
		Evaluation: &domain.EvaluationResult{
			Score:          75,
			ScoreBreakdown: map[string]float64{"准确性": 30},
			Headline:       "结论基本正确，但细节不够。",
		},
	}
	if err := store.CreateSession(ctx, session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if withReview {
		review := &domain.ReviewCard{
			ID:                reviewID,
			SessionID:         session.ID,
			Overall:           "还需要把线上排障讲清楚。",
			TopFix:            "补完整排障闭环",
			TopFixReason:      "这会直接影响项目题说服力。",
			Highlights:        []string{"先说结论"},
			Gaps:              []string{"缺线上例子"},
			SuggestedTopics:   []string{"redis"},
			NextTrainingFocus: []string{"围绕 Redis 故障排查继续练"},
			ScoreBreakdown:    map[string]float64{"完整性": 74},
		}
		if err := store.CreateReview(ctx, review); err != nil {
			t.Fatalf("CreateReview() error = %v", err)
		}
	}

	saved, err := store.GetSession(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved session")
	}
	return saved
}

func seedRouteJobTarget(t *testing.T, store *repo.Store) *domain.JobTarget {
	t.Helper()

	target, err := store.CreateJobTarget(context.Background(), domain.JobTargetInput{
		Title:       "后端工程师 - Route",
		CompanyName: "Example",
		SourceText:  "要求 Go、Redis 和排障能力。",
	})
	if err != nil {
		t.Fatalf("CreateJobTarget() error = %v", err)
	}

	run, err := store.StartJobTargetAnalysis(context.Background(), target.ID, target.SourceText)
	if err != nil {
		t.Fatalf("StartJobTargetAnalysis() error = %v", err)
	}

	if err := store.CompleteJobTargetAnalysis(context.Background(), target.ID, run.ID, &domain.AnalyzeJobTargetResponse{
		Summary:          "重点看缓存与排障闭环。",
		MustHaveSkills:   []string{"Go", "Redis"},
		Responsibilities: []string{"稳定性保障"},
		EvaluationFocus:  []string{"故障定位"},
	}); err != nil {
		t.Fatalf("CompleteJobTargetAnalysis() error = %v", err)
	}

	saved, err := store.GetJobTarget(context.Background(), target.ID)
	if err != nil {
		t.Fatalf("GetJobTarget() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved job target")
	}
	return saved
}

func openTestStore(t *testing.T) (*repo.Store, error) {
	t.Helper()

	db, err := sqlite.Open(filepath.Join(t.TempDir(), "practicehelper.db"))
	if err != nil {
		return nil, err
	}

	if err := sqlite.Bootstrap(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo.New(db), nil
}
