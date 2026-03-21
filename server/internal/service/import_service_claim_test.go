package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/sidecar"
)

func TestStartImportJobClaimsOnlyOneWorker(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	var analyzeCalls atomic.Int32
	sidecarServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/analyze_repo" {
			t.Fatalf("unexpected sidecar path: %s", r.URL.Path)
		}

		analyzeCalls.Add(1)
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(domain.AnalyzeRepoResponse{
			RepoURL:       "https://github.com/octocat/claim-once",
			Name:          "claim-once",
			DefaultBranch: "main",
			ImportCommit:  "abc123",
			Summary:       "test import",
			Chunks:        []domain.RepoChunk{},
		}); err != nil {
			t.Fatalf("encode analyze repo response: %v", err)
		}
	}))
	defer sidecarServer.Close()

	job, err := store.CreateProjectImportJob(context.Background(), "https://github.com/octocat/claim-once")
	if err != nil {
		t.Fatalf("CreateProjectImportJob() error = %v", err)
	}

	svc := &Service{
		repo:    store,
		sidecar: sidecar.New(sidecarServer.URL, 5*time.Second),
	}

	svc.startImportJob(*job)
	svc.startImportJob(*job)

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		saved, err := store.GetProjectImportJob(context.Background(), job.ID)
		if err != nil {
			t.Fatalf("GetProjectImportJob() error = %v", err)
		}
		if saved != nil && saved.Status == domain.ProjectImportStatusCompleted {
			if analyzeCalls.Load() != 1 {
				t.Fatalf("expected analyze repo to be called once, got %d", analyzeCalls.Load())
			}
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for import job completion; analyze_calls=%d", analyzeCalls.Load())
}
