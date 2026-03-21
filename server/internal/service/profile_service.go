package service

import (
	"context"

	"practicehelper/server/internal/domain"
)

func (s *Service) GetProfile(ctx context.Context) (*domain.UserProfile, error) {
	return s.repo.GetUserProfile(ctx)
}

func (s *Service) SaveProfile(ctx context.Context, input domain.UserProfileInput) (*domain.UserProfile, error) {
	profile, err := s.repo.SaveUserProfile(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := s.repo.EnsureKnowledgeSeeds(ctx); err != nil {
		return nil, err
	}
	return profile, nil
}

func (s *Service) GetDashboard(ctx context.Context) (*domain.Dashboard, error) {
	profile, err := s.repo.GetUserProfile(ctx)
	if err != nil {
		return nil, err
	}

	weaknesses, err := s.repo.ListWeaknesses(ctx, 20)
	if err != nil {
		return nil, err
	}

	var activeJobTarget *domain.JobTarget
	if profile != nil && profile.ActiveJobTargetID != "" {
		activeJobTarget, err = s.repo.GetJobTarget(ctx, profile.ActiveJobTargetID)
		if err != nil {
			return nil, err
		}
	}

	recommendationScope := "generic"
	if activeJobTarget != nil && activeJobTarget.LatestAnalysisStatus == domain.JobTargetAnalysisSucceeded && activeJobTarget.LatestSuccessfulAnalysis != nil {
		weaknesses = weightWeaknessesForActiveJobTarget(weaknesses, activeJobTarget)
		recommendationScope = "job_target"
	}
	if len(weaknesses) > 5 {
		weaknesses = weaknesses[:5]
	}

	recent, err := s.repo.ListRecentSessions(ctx, 5)
	if err != nil {
		return nil, err
	}

	currentSession, err := s.repo.GetLatestResumableSession(ctx)
	if err != nil {
		return nil, err
	}

	dashboard := &domain.Dashboard{
		Profile:             profile,
		Weaknesses:          weaknesses,
		RecentSessions:      recent,
		CurrentSession:      currentSession,
		TodayFocus:          buildTodayFocus(profile, activeJobTarget, weaknesses, recommendationScope),
		RecommendedTrack:    buildRecommendedTrack(profile, activeJobTarget, weaknesses, recommendationScope),
		RecommendationScope: recommendationScope,
	}
	if profile != nil {
		dashboard.ActiveJobTarget = profile.ActiveJobTarget
	}

	if profile != nil && profile.ApplicationDeadline != nil {
		days := daysUntilDeadline(*profile.ApplicationDeadline)
		if days < 0 {
			days = 0
		}
		dashboard.DaysUntilDeadline = &days
	}

	return dashboard, nil
}
