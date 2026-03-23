package repo

import (
	"context"
	"database/sql"
	"fmt"

	"practicehelper/server/internal/domain"
)

func (s *Store) GetPromptPreferences(ctx context.Context) (*domain.PromptOverlay, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT tone, detail_level, followup_intensity, answer_language, focus_tags_json, custom_instruction
		FROM prompt_preferences
		WHERE id = 1
	`)

	var (
		tone, detailLevel, followupIntensity string
		answerLanguage, focusTagsJSON        string
		customInstruction                    string
	)
	if err := row.Scan(
		&tone,
		&detailLevel,
		&followupIntensity,
		&answerLanguage,
		&focusTagsJSON,
		&customInstruction,
	); err != nil {
		if err == sql.ErrNoRows {
			return &domain.PromptOverlay{}, nil
		}
		return nil, fmt.Errorf("get prompt preferences: %w", err)
	}

	return &domain.PromptOverlay{
		Tone:              tone,
		DetailLevel:       detailLevel,
		FollowupIntensity: followupIntensity,
		AnswerLanguage:    answerLanguage,
		FocusTags:         parseStringList(focusTagsJSON),
		CustomInstruction: customInstruction,
	}, nil
}

func (s *Store) SavePromptPreferences(ctx context.Context, overlay *domain.PromptOverlay) error {
	if overlay == nil {
		overlay = &domain.PromptOverlay{}
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO prompt_preferences (
			id, tone, detail_level, followup_intensity, answer_language,
			focus_tags_json, custom_instruction, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			tone = excluded.tone,
			detail_level = excluded.detail_level,
			followup_intensity = excluded.followup_intensity,
			answer_language = excluded.answer_language,
			focus_tags_json = excluded.focus_tags_json,
			custom_instruction = excluded.custom_instruction,
			updated_at = excluded.updated_at
	`,
		1,
		overlay.Tone,
		overlay.DetailLevel,
		overlay.FollowupIntensity,
		overlay.AnswerLanguage,
		mustJSON(overlay.FocusTags),
		overlay.CustomInstruction,
		nowUTC(),
		nowUTC(),
	)
	if err != nil {
		return fmt.Errorf("save prompt preferences: %w", err)
	}
	return nil
}
