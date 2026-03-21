package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
)

func TestProfileRoutesPersistAndDashboardIncludesActiveJobTarget(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target := seedRouteJobTarget(t, store)
	router := NewRouter(service.New(store, nil))

	saveRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/profile",
		strings.NewReader(
			`{
				"target_role":"后端工程师",
				"target_company_type":"互联网",
				"current_stage":"一面准备",
				"application_deadline":"2026-03-30",
				"tech_stacks":["Go","Redis"],
				"primary_projects":["PracticeHelper"],
				"self_reported_weaknesses":["系统设计"],
				"active_job_target_id":"`+target.ID+`"
			}`,
		),
	)
	saveRequest.Header.Set("Content-Type", "application/json")
	saveRecorder := httptest.NewRecorder()
	router.ServeHTTP(saveRecorder, saveRequest)

	if saveRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", saveRecorder.Code, saveRecorder.Body.String())
	}

	profile := decodeDataEnvelope[domain.UserProfile](t, saveRecorder.Body.Bytes())
	if profile.TargetRole != "后端工程师" {
		t.Fatalf("expected target role saved, got %q", profile.TargetRole)
	}
	if profile.ActiveJobTarget == nil || profile.ActiveJobTarget.ID != target.ID {
		t.Fatalf("expected active job target %s, got %#v", target.ID, profile.ActiveJobTarget)
	}

	getProfileRequest := httptest.NewRequest(http.MethodGet, "/api/profile", nil)
	getProfileRecorder := httptest.NewRecorder()
	router.ServeHTTP(getProfileRecorder, getProfileRequest)

	if getProfileRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getProfileRecorder.Code, getProfileRecorder.Body.String())
	}

	dashboardRequest := httptest.NewRequest(http.MethodGet, "/api/dashboard", nil)
	dashboardRecorder := httptest.NewRecorder()
	router.ServeHTTP(dashboardRecorder, dashboardRequest)

	if dashboardRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", dashboardRecorder.Code, dashboardRecorder.Body.String())
	}

	dashboard := decodeDataEnvelope[domain.Dashboard](t, dashboardRecorder.Body.Bytes())
	if dashboard.Profile == nil || dashboard.Profile.TargetRole != "后端工程师" {
		t.Fatalf("expected dashboard profile, got %#v", dashboard.Profile)
	}
	if dashboard.ActiveJobTarget == nil || dashboard.ActiveJobTarget.ID != target.ID {
		t.Fatalf("expected dashboard active job target %s, got %#v", target.ID, dashboard.ActiveJobTarget)
	}
	if dashboard.RecommendationScope != "job_target" {
		t.Fatalf("expected recommendation scope job_target, got %q", dashboard.RecommendationScope)
	}
}
