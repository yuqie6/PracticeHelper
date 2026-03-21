package controller

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

func TestExportSessionRouteReturnsJSONAttachment(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedExportRouteSession(t, store, "sess_route_export_json", true)
	router := NewRouter(service.New(store, nil))

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/sessions/"+session.ID+"/export?format=json",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("unexpected content type: %s", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("Content-Disposition") != `attachment; filename="practicehelper-session-sess_route_export_json.json"` {
		t.Fatalf("unexpected content disposition: %s", recorder.Header().Get("Content-Disposition"))
	}

	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if payload["format"] != "json" {
		t.Fatalf("unexpected format payload: %#v", payload["format"])
	}
}

func TestExportSessionRouteReturnsPDFAttachment(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := seedExportRouteSession(t, store, "sess_route_export_pdf", true)
	router := NewRouter(service.New(store, nil))

	request := httptest.NewRequest(
		http.MethodGet,
		"/api/sessions/"+session.ID+"/export?format=pdf",
		nil,
	)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("unexpected content type: %s", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("Content-Disposition") != `attachment; filename="practicehelper-session-sess_route_export_pdf.pdf"` {
		t.Fatalf("unexpected content disposition: %s", recorder.Header().Get("Content-Disposition"))
	}
	if !bytes.HasPrefix(recorder.Body.Bytes(), []byte("%PDF-1.4")) {
		t.Fatalf("expected pdf payload, got: %q", recorder.Body.Bytes()[:8])
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
		"/api/sessions/"+session.ID+"/export?format=csv",
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

func TestDueReviewsRouteReturnsWeaknessFields(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	session := &domain.TrainingSession{
		ID:         "sess_route_due",
		Mode:       domain.ModeBasics,
		Topic:      domain.BasicsTopicRedis,
		Intensity:  "standard",
		Status:     domain.StatusCompleted,
		TotalScore: 82,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_route_due",
		SessionID:      session.ID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "Redis 为什么快？",
		ExpectedPoints: []string{"内存访问"},
		Answer:         "因为热点数据主要在内存中。",
	}
	if err := store.CreateSession(context.Background(), session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := store.UpsertWeaknesses(context.Background(), session.ID, []domain.WeaknessHit{
		{Kind: "topic", Label: "缓存一致性", Severity: 1.0},
	}); err != nil {
		t.Fatalf("UpsertWeaknesses() error = %v", err)
	}
	tag, err := store.GetWeaknessTag(context.Background(), "topic", "缓存一致性")
	if err != nil {
		t.Fatalf("GetWeaknessTag() error = %v", err)
	}
	if tag == nil {
		t.Fatal("expected seeded weakness tag")
	}
	if err := store.CreateReviewSchedule(context.Background(), &domain.ReviewScheduleItem{
		SessionID:     session.ID,
		ReviewCardID:  "review_route_due",
		WeaknessTagID: tag.ID,
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
	if payload.Data[0].WeaknessKind != "topic" {
		t.Fatalf("expected weakness kind topic, got %q", payload.Data[0].WeaknessKind)
	}
	if payload.Data[0].WeaknessLabel != "缓存一致性" {
		t.Fatalf("expected weakness label 缓存一致性, got %q", payload.Data[0].WeaknessLabel)
	}
	if payload.Data[0].Topic != domain.BasicsTopicRedis {
		t.Fatalf("expected topic redis, got %q", payload.Data[0].Topic)
	}
}

func TestExportSessionsRouteReturnsZipAttachment(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	first := seedExportRouteSession(t, store, "sess_route_batch_1", true)
	second := seedExportRouteSession(t, store, "sess_route_batch_2", false)
	router := NewRouter(service.New(store, nil))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/sessions/export",
		strings.NewReader(
			fmt.Sprintf(
				`{"session_ids":["%s","%s"],"format":"markdown"}`,
				first.ID,
				second.ID,
			),
		),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Disposition") != `attachment; filename="practicehelper-sessions-2-markdown.zip"` {
		t.Fatalf("unexpected content disposition: %s", recorder.Header().Get("Content-Disposition"))
	}

	reader, err := zip.NewReader(bytes.NewReader(recorder.Body.Bytes()), int64(recorder.Body.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}
	if len(reader.File) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(reader.File))
	}
}

func TestExportSessionsRouteReturnsJSONZipAttachment(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	first := seedExportRouteSession(t, store, "sess_route_json_zip_1", true)
	second := seedExportRouteSession(t, store, "sess_route_json_zip_2", false)
	router := NewRouter(service.New(store, nil))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/sessions/export",
		strings.NewReader(
			fmt.Sprintf(
				`{"session_ids":["%s","%s"],"format":"json"}`,
				first.ID,
				second.ID,
			),
		),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Disposition") != `attachment; filename="practicehelper-sessions-2-json.zip"` {
		t.Fatalf("unexpected content disposition: %s", recorder.Header().Get("Content-Disposition"))
	}

	reader, err := zip.NewReader(bytes.NewReader(recorder.Body.Bytes()), int64(recorder.Body.Len()))
	if err != nil {
		t.Fatalf("zip.NewReader() error = %v", err)
	}
	if len(reader.File) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(reader.File))
	}
	if !strings.HasSuffix(reader.File[0].Name, ".json") {
		t.Fatalf("expected json entry, got %s", reader.File[0].Name)
	}
}

func TestExportSessionsRouteRejectsEmptySelection(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	router := NewRouter(service.New(store, nil))

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/sessions/export",
		strings.NewReader(`{"session_ids":[],"format":"markdown"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}

	payload := decodeErrorPayload(t, recorder.Body.Bytes())
	if payload.Error.Code != "empty_export_selection" {
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
