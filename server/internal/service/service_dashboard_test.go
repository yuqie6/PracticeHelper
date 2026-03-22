package service

import (
	"context"
	"testing"

	"practicehelper/server/internal/domain"
	"practicehelper/server/internal/repo"
)

func TestBuildTodayFocusUsesReadableWeaknessLabels(t *testing.T) {
	profile := &domain.UserProfile{}
	weaknesses := []domain.WeaknessTag{
		{
			Kind:  "depth",
			Label: "回答缺少因果展开",
		},
	}

	focus := buildTodayFocus(profile, nil, weaknesses, "generic")
	want := "今天优先补 展开深度：回答缺少因果展开"
	if focus != want {
		t.Fatalf("unexpected today focus: got %q want %q", focus, want)
	}
}

func TestBuildRecommendedTrackCoversNewWeaknessKinds(t *testing.T) {
	cases := []struct {
		name string
		kind string
		want string
	}{
		{name: "expression", kind: "expression", want: "围绕「x」做表达方式专项"},
		{name: "followup", kind: "followup_breakdown", want: "围绕「x」做追问抗压专项"},
		{name: "depth", kind: "depth", want: "围绕「x」做展开深挖专项"},
		{name: "detail", kind: "detail", want: "围绕「x」做细节补强专项"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildRecommendedTrack(nil, nil, []domain.WeaknessTag{{Kind: tc.kind, Label: "x"}}, "generic")
			if got != tc.want {
				t.Fatalf("unexpected recommended track: got %q want %q", got, tc.want)
			}
		})
	}
}

func TestWeightWeaknessesForActiveJobTargetPrioritizesMatchedLabels(t *testing.T) {
	weighted := weightWeaknessesForActiveJobTarget(
		[]domain.WeaknessTag{
			{Kind: "detail", Label: "回答缺少因果展开", Severity: 0.88, Frequency: 2},
			{Kind: "topic", Label: "缓存一致性", Severity: 0.75, Frequency: 1},
		},
		&domain.JobTarget{
			Title: "后端工程师",
			LatestSuccessfulAnalysis: &domain.JobTargetAnalysisRun{
				MustHaveSkills:   []string{"Redis", "缓存一致性"},
				Responsibilities: []string{"负责核心链路稳定性"},
				EvaluationFocus:  []string{"故障排查闭环"},
			},
		},
	)

	if len(weighted) != 2 {
		t.Fatalf("expected 2 weighted weaknesses, got %d", len(weighted))
	}
	if weighted[0].Label != "缓存一致性" {
		t.Fatalf("expected matched weakness to rank first, got %s", weighted[0].Label)
	}
}

func TestGetDashboardKeepsUnavailableActiveJobTargetVisible(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	target := seedReadyJobTarget(t, store)
	if _, err := store.SetActiveJobTarget(context.Background(), target.ID); err != nil {
		t.Fatalf("SetActiveJobTarget() error = %v", err)
	}
	if _, err := store.UpdateJobTarget(context.Background(), target.ID, domain.JobTargetInput{
		Title:       target.Title,
		CompanyName: target.CompanyName,
		SourceText:  target.SourceText + "\n新增：强调稳定性值班与故障演练。",
	}); err != nil {
		t.Fatalf("UpdateJobTarget() error = %v", err)
	}

	svc := New(store, nil)
	dashboard, err := svc.GetDashboard(context.Background())
	if err != nil {
		t.Fatalf("GetDashboard() error = %v", err)
	}
	if dashboard == nil {
		t.Fatal("expected dashboard")
	}
	if dashboard.ActiveJobTarget == nil {
		t.Fatal("expected dashboard to keep exposing active job target")
	}
	if dashboard.ActiveJobTarget.ID != target.ID {
		t.Fatalf("expected active job target id %q, got %q", target.ID, dashboard.ActiveJobTarget.ID)
	}
	if dashboard.ActiveJobTarget.LatestAnalysisStatus != domain.JobTargetAnalysisStale {
		t.Fatalf(
			"expected active job target status %q, got %q",
			domain.JobTargetAnalysisStale,
			dashboard.ActiveJobTarget.LatestAnalysisStatus,
		)
	}
	if dashboard.RecommendationScope != "generic" {
		t.Fatalf("expected recommendation scope to fall back to generic, got %q", dashboard.RecommendationScope)
	}
}

func TestGetDashboardKeepsRunningOrFailedActiveJobTargetVisible(t *testing.T) {
	cases := []struct {
		name       string
		prepare    func(*testing.T, *repo.Store, *domain.JobTarget)
		wantStatus string
	}{
		{
			name:       "running",
			prepare:    markJobTargetAnalysisRunning,
			wantStatus: domain.JobTargetAnalysisRunning,
		},
		{
			name:       "failed",
			prepare:    markJobTargetAnalysisFailed,
			wantStatus: domain.JobTargetAnalysisFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, err := openTestStore(t)
			if err != nil {
				t.Fatalf("openTestStore() error = %v", err)
			}
			defer func() { _ = store.Close() }()

			target := seedReadyJobTarget(t, store)
			if _, err := store.SetActiveJobTarget(context.Background(), target.ID); err != nil {
				t.Fatalf("SetActiveJobTarget() error = %v", err)
			}
			tc.prepare(t, store, target)

			svc := New(store, nil)
			dashboard, err := svc.GetDashboard(context.Background())
			if err != nil {
				t.Fatalf("GetDashboard() error = %v", err)
			}
			if dashboard == nil {
				t.Fatal("expected dashboard")
			}
			if dashboard.ActiveJobTarget == nil {
				t.Fatal("expected dashboard to keep exposing active job target")
			}
			if dashboard.ActiveJobTarget.ID != target.ID {
				t.Fatalf("expected active job target id %q, got %q", target.ID, dashboard.ActiveJobTarget.ID)
			}
			if dashboard.ActiveJobTarget.LatestAnalysisStatus != tc.wantStatus {
				t.Fatalf("expected active job target status %q, got %q", tc.wantStatus, dashboard.ActiveJobTarget.LatestAnalysisStatus)
			}
			if dashboard.RecommendationScope != "generic" {
				t.Fatalf("expected recommendation scope to fall back to generic, got %q", dashboard.RecommendationScope)
			}
		})
	}
}
