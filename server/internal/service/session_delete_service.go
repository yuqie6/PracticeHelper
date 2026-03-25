package service

import (
	"context"

	"practicehelper/server/internal/domain"
)

func (s *Service) DeleteSessions(
	ctx context.Context,
	sessionIDs []string,
) (*domain.DeleteSessionsResult, error) {
	sessionIDs = normalizeDeleteSessionIDs(sessionIDs)
	if len(sessionIDs) == 0 {
		return nil, ErrEmptyDeleteSelection
	}

	var deleteVectorPoints func(context.Context, []string) error
	if s.vectorStore != nil && s.vectorStore.Enabled() {
		deleteVectorPoints = s.vectorStore.Delete
	}

	result, err := s.repo.DeleteSessions(ctx, sessionIDs, deleteVectorPoints)
	if err != nil {
		return nil, err
	}
	if result == nil || result.DeletedCount == 0 {
		return nil, ErrSessionNotFound
	}
	return result, nil
}

func normalizeDeleteSessionIDs(sessionIDs []string) []string {
	seen := make(map[string]struct{}, len(sessionIDs))
	result := make([]string, 0, len(sessionIDs))
	for _, sessionID := range sessionIDs {
		if sessionID == "" {
			continue
		}
		if _, exists := seen[sessionID]; exists {
			continue
		}
		seen[sessionID] = struct{}{}
		result = append(result, sessionID)
	}
	return result
}
