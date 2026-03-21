package service

import (
	"context"

	"practicehelper/server/internal/domain"
)

func (s *Service) SearchProjectChunksForAgent(
	ctx context.Context,
	projectID string,
	query string,
	limit int,
) ([]domain.RepoChunk, error) {
	return s.repo.SearchProjectChunks(ctx, projectID, query, limit)
}

func (s *Service) GetAgentSessionDetail(
	ctx context.Context,
	sessionID string,
) (*domain.AgentSessionDetail, error) {
	session, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, ErrSessionNotFound
	}

	var review *domain.ReviewCard
	if session.ReviewID != "" {
		review, err = s.repo.GetReview(ctx, session.ReviewID)
		if err != nil {
			return nil, err
		}
	}

	return &domain.AgentSessionDetail{
		Session: session,
		Review:  review,
	}, nil
}
