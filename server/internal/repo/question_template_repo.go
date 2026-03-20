package repo

import (
	"context"
	"fmt"

	"practicehelper/server/internal/domain"
)

func (s *Store) ListQuestionTemplatesByTopic(ctx context.Context, topic string) ([]domain.QuestionTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, mode, topic, prompt, focus_points_json, bad_answers_json, followup_templates_json, score_weights_json
		FROM question_templates
		WHERE topic = ?
		ORDER BY id ASC
	`, topic)
	if err != nil {
		return nil, fmt.Errorf("list question templates: %w", err)
	}
	defer func() { _ = rows.Close() }()

	templates := make([]domain.QuestionTemplate, 0)
	for rows.Next() {
		template, err := scanQuestionTemplate(rows)
		if err != nil {
			return nil, err
		}
		templates = append(templates, *template)
	}

	return templates, nil
}
