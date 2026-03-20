package service

import (
	"context"

	"practicehelper/server/internal/domain"
)

func (s *Service) GetProfile(ctx context.Context) (*domain.UserProfile, error) {
	return s.repo.GetUserProfile(ctx)
}

func (s *Service) SaveProfile(ctx context.Context, input domain.UserProfileInput) (*domain.UserProfile, error) {
	return s.repo.SaveUserProfile(ctx, input)
}

func (s *Service) GetDashboard(ctx context.Context) (*domain.Dashboard, error) {
	profile, err := s.repo.GetUserProfile(ctx)
	if err != nil {
		return nil, err
	}

	weaknesses, err := s.repo.ListWeaknesses(ctx, 5)
	if err != nil {
		return nil, err
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
		Profile:          profile,
		Weaknesses:       weaknesses,
		RecentSessions:   recent,
		CurrentSession:   currentSession,
		TodayFocus:       buildTodayFocus(profile, weaknesses),
		RecommendedTrack: buildRecommendedTrack(profile, weaknesses),
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
