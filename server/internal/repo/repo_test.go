package repo

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
)

func TestOpenInitializesSQLiteFTS5Schema(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "practicehelper.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM question_templates`).Scan(&count); err != nil {
		t.Fatalf("seeded question_templates query failed: %v", err)
	}

	if count == 0 {
		t.Fatal("expected seeded question templates, got 0")
	}
}

func TestSaveUserProfileParsesDateOnlyDeadline(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "practicehelper.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	deadline := "2026-04-01"
	profile, err := store.SaveUserProfile(context.Background(), domain.UserProfileInput{
		TargetRole:           "Go Backend Engineer",
		TargetCompanyType:    "Startup",
		CurrentStage:         "Interview prep",
		ApplicationDeadline:  &deadline,
		TechStacks:           []string{"Go"},
		PrimaryProjects:      []string{"PracticeHelper"},
		SelfReportedWeakness: []string{"Kafka"},
	})
	if err != nil {
		t.Fatalf("SaveUserProfile() error = %v", err)
	}

	if profile.ApplicationDeadline == nil {
		t.Fatal("expected application deadline to be parsed")
	}

	got := profile.ApplicationDeadline.UTC().Format(time.DateOnly)
	if got != deadline {
		t.Fatalf("expected deadline %s, got %s", deadline, got)
	}
}

func TestProjectImportJobLifecycle(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "practicehelper.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	job, err := store.CreateProjectImportJob(ctx, "https://github.com/octocat/Hello-World")
	if err != nil {
		t.Fatalf("CreateProjectImportJob() error = %v", err)
	}

	if job.Status != domain.ProjectImportStatusQueued {
		t.Fatalf("expected queued status, got %s", job.Status)
	}

	startedAt := time.Now().UTC()
	if err := store.UpdateProjectImportJobStatus(
		ctx,
		job.ID,
		domain.ProjectImportStatusRunning,
		domain.ProjectImportStageAnalyzing,
		"running",
		"",
		"",
		&startedAt,
		nil,
	); err != nil {
		t.Fatalf("UpdateProjectImportJobStatus() running error = %v", err)
	}

	activeJob, err := store.FindActiveProjectImportJobByRepoURL(ctx, job.RepoURL)
	if err != nil {
		t.Fatalf("FindActiveProjectImportJobByRepoURL() error = %v", err)
	}
	if activeJob == nil || activeJob.ID != job.ID {
		t.Fatalf("expected active job %s, got %+v", job.ID, activeJob)
	}

	finishedAt := startedAt.Add(2 * time.Minute)
	if err := store.UpdateProjectImportJobStatus(
		ctx,
		job.ID,
		domain.ProjectImportStatusFailed,
		domain.ProjectImportStageFailed,
		"failed",
		"boom",
		"",
		nil,
		&finishedAt,
	); err != nil {
		t.Fatalf("UpdateProjectImportJobStatus() failed error = %v", err)
	}

	saved, err := store.GetProjectImportJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetProjectImportJob() error = %v", err)
	}
	if saved == nil {
		t.Fatal("expected saved import job")
	}
	if saved.Status != domain.ProjectImportStatusFailed {
		t.Fatalf("expected failed status, got %s", saved.Status)
	}
	if saved.ErrorMessage != "boom" {
		t.Fatalf("expected error message boom, got %s", saved.ErrorMessage)
	}
	if saved.FinishedAt == nil {
		t.Fatal("expected finished_at to be set")
	}
}
