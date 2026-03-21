package repo

import (
	"context"
	"fmt"

	"practicehelper/server/internal/domain"
)

func (s *Store) ListHistoricalPromptSets(ctx context.Context) ([]domain.PromptSetSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT prompt_set_id, COALESCE(prompt_set_label, ''), COALESCE(prompt_set_status, '')
		FROM training_sessions
		WHERE prompt_set_id != ''
		ORDER BY updated_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list historical prompt sets: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.PromptSetSummary, 0)
	seen := make(map[string]struct{})
	for rows.Next() {
		var promptSetID, label, status string
		if err := rows.Scan(&promptSetID, &label, &status); err != nil {
			return nil, fmt.Errorf("scan historical prompt set: %w", err)
		}
		if promptSetID == "" {
			continue
		}
		if _, exists := seen[promptSetID]; exists {
			continue
		}
		seen[promptSetID] = struct{}{}

		item := parsePromptSetSummary(promptSetID, label, status)
		if item == nil {
			continue
		}
		items = append(items, *item)
	}

	return items, nil
}
