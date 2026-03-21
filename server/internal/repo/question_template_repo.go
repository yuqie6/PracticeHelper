package repo

import (
	"context"
	"fmt"
	"strings"

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

func (s *Store) ListQuestionTemplatesByTopics(
	ctx context.Context,
	topics []string,
) ([]domain.QuestionTemplate, error) {
	if len(topics) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(topics)), ",")
	args := make([]any, 0, len(topics))
	for _, topic := range topics {
		args = append(args, topic)
	}

	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, mode, topic, prompt, focus_points_json, bad_answers_json, followup_templates_json, score_weights_json
		FROM question_templates
		WHERE topic IN (%s)
		ORDER BY topic ASC, id ASC
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("list question templates by topics: %w", err)
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
