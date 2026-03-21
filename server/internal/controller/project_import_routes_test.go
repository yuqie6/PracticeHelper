package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/service"
)

func TestProjectRoutesSupportListGetAndUpdate(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	project, err := store.CreateImportedProject(context.Background(), &domain.AnalyzeRepoResponse{
		RepoURL:         "https://github.com/example/practicehelper",
		Name:            "PracticeHelper",
		DefaultBranch:   "main",
		ImportCommit:    "abc123",
		Summary:         "初始摘要",
		TechStack:       []string{"Go", "Vue"},
		Highlights:      []string{"训练闭环"},
		Challenges:      []string{"状态协调"},
		Tradeoffs:       []string{"轻量实现"},
		OwnershipPoints: []string{"全链路"},
		FollowupPoints:  []string{"恢复机制"},
		Chunks: []domain.RepoChunk{
			{
				FilePath:   "server/main.go",
				FileType:   ".go",
				Content:    "package main",
				Importance: 1,
				FTSKey:     "server/main.go#0",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateImportedProject() error = %v", err)
	}

	router := NewRouter(service.New(store, nil))

	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/api/projects", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	projects := decodeDataEnvelope[[]domain.ProjectProfile](t, listRecorder.Body.Bytes())
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		getRecorder,
		httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID, nil),
	)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getRecorder.Code, getRecorder.Body.String())
	}

	updateRequest := httptest.NewRequest(
		http.MethodPatch,
		"/api/projects/"+project.ID,
		strings.NewReader(
			`{
				"name":"PracticeHelper Reloaded",
				"summary":"更新后的摘要",
				"tech_stack":["Go","Vue","LangGraph"],
				"highlights":["训练闭环","多轮训练"],
				"challenges":["状态协调"],
				"tradeoffs":["轻量实现"],
				"ownership_points":["全链路"],
				"followup_points":["恢复机制","评估审计"]
			}`,
		),
	)
	updateRequest.Header.Set("Content-Type", "application/json")
	updateRecorder := httptest.NewRecorder()
	router.ServeHTTP(updateRecorder, updateRequest)

	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", updateRecorder.Code, updateRecorder.Body.String())
	}
	updated := decodeDataEnvelope[domain.ProjectProfile](t, updateRecorder.Body.Bytes())
	if updated.Name != "PracticeHelper Reloaded" {
		t.Fatalf("expected updated project name, got %q", updated.Name)
	}
}

func TestImportJobRoutesSupportListGetAndRetry(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	job, err := store.CreateProjectImportJob(ctx, "https://github.com/example/import-me")
	if err != nil {
		t.Fatalf("CreateProjectImportJob() error = %v", err)
	}

	startedAt := time.Now().UTC().Add(-time.Minute)
	finishedAt := time.Now().UTC()
	if err := store.UpdateProjectImportJobStatus(
		ctx,
		job.ID,
		domain.ProjectImportStatusFailed,
		domain.ProjectImportStageFailed,
		"导入失败，请稍后重试。",
		"sidecar unavailable",
		"",
		&startedAt,
		&finishedAt,
	); err != nil {
		t.Fatalf("UpdateProjectImportJobStatus() error = %v", err)
	}

	router := NewRouter(service.New(store, nil))

	listRecorder := httptest.NewRecorder()
	router.ServeHTTP(listRecorder, httptest.NewRequest(http.MethodGet, "/api/import-jobs", nil))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", listRecorder.Code, listRecorder.Body.String())
	}
	jobs := decodeDataEnvelope[[]domain.ProjectImportJob](t, listRecorder.Body.Bytes())
	if len(jobs) != 1 || jobs[0].Status != domain.ProjectImportStatusFailed {
		t.Fatalf("expected one failed import job, got %#v", jobs)
	}

	getRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		getRecorder,
		httptest.NewRequest(http.MethodGet, "/api/import-jobs/"+job.ID, nil),
	)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", getRecorder.Code, getRecorder.Body.String())
	}

	retryRecorder := httptest.NewRecorder()
	router.ServeHTTP(
		retryRecorder,
		httptest.NewRequest(http.MethodPost, "/api/import-jobs/"+job.ID+"/retry", nil),
	)
	if retryRecorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", retryRecorder.Code, retryRecorder.Body.String())
	}
	retried := decodeDataEnvelope[domain.ProjectImportJob](t, retryRecorder.Body.Bytes())
	if retried.Status != domain.ProjectImportStatusQueued {
		t.Fatalf("expected queued import job after retry, got %q", retried.Status)
	}
}
