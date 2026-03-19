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
