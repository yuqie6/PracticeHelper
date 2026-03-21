package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
)

func TestJobTargetRoutesSupportCrudActivationAndAnalysisHistory(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	router := NewRouter(service.New(store, nil))

	createRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/job-targets",
		strings.NewReader(`{"title":"后端负责人","company_name":"Example","source_text":"要求 Go 和 Redis"}`),
	)
	createRequest.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	router.ServeHTTP(createRecorder, createRequest)

	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", createRecorder.Code, createRecorder.Body.String())
	}

	created := decodeDataEnvelope[domain.JobTarget](t, createRecorder.Body.Bytes())
	if created.Title != "后端负责人" {
		t.Fatalf("expected created title, got %q", created.Title)
	}

	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/api/job-targets", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	listed := decodeDataEnvelope[[]domain.JobTarget](t, listRecorder.Body.Bytes())
	if len(listed) != 1 {
		t.Fatalf("expected 1 job target, got %d", len(listed))
	}

	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		getRecorder,
		httptest.NewRequest(http.MethodGet, "/api/job-targets/"+created.ID, nil),
	)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getRecorder.Code, getRecorder.Body.String())
	}

	updateRequest := httptest.NewRequest(
		http.MethodPatch,
		"/api/job-targets/"+created.ID,
		strings.NewReader(`{"title":"资深后端负责人","company_name":"Example","source_text":"要求 Go、Redis 和排障能力"}`),
	)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, updateRequest)

	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", updateRecorder.Code, updateRecorder.Body.String())
	}
	updated := decodeDataEnvelope[domain.JobTarget](t, updateRecorder.Body.Bytes())
	if updated.Title != "资深后端负责人" {
		t.Fatalf("expected updated title, got %q", updated.Title)
	}

	activateRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		activateRecorder,
		httptest.NewRequest(http.MethodPost, "/api/job-targets/"+created.ID+"/activate", nil),
	)
	if activateRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", activateRecorder.Code, activateRecorder.Body.String())
	}
	profile := decodeDataEnvelope[domain.UserProfile](t, activateRecorder.Body.Bytes())
	if profile.ActiveJobTargetID != created.ID {
		t.Fatalf("expected active job target %s, got %q", created.ID, profile.ActiveJobTargetID)
	}

	clearRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		clearRecorder,
		httptest.NewRequest(http.MethodPost, "/api/job-targets/clear-active", nil),
	)
	if clearRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", clearRecorder.Code, clearRecorder.Body.String())
	}
	cleared := decodeDataEnvelope[domain.UserProfile](t, clearRecorder.Body.Bytes())
	if cleared.ActiveJobTargetID != "" {
		t.Fatalf("expected cleared active job target, got %q", cleared.ActiveJobTargetID)
	}

	seeded := seedRouteJobTarget(t, store)
	listRunsRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		listRunsRecorder,
		httptest.NewRequest(http.MethodGet, "/api/job-targets/"+seeded.ID+"/analysis-runs", nil),
	)
	if listRunsRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRunsRecorder.Code, listRunsRecorder.Body.String())
	}
	runs := decodeDataEnvelope[[]domain.JobTargetAnalysisRun](t, listRunsRecorder.Body.Bytes())
	if len(runs) == 0 {
		t.Fatal("expected at least one analysis run")
	}

	getRunRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		getRunRecorder,
		httptest.NewRequest(http.MethodGet, "/api/job-targets/analysis-runs/"+runs[0].ID, nil),
	)
	if getRunRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getRunRecorder.Code, getRunRecorder.Body.String())
	}
}
