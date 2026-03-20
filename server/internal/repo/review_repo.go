package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"practicehelper/server/internal/domain"
)

func (s *Store) CreateReview(ctx context.Context, review *domain.ReviewCard) error {
	// 一个 session 只保留一张 review card。冲突时覆盖内容是为了支持重试/补生成，
	// 调用方应把这里视为幂等写，而不是为同一轮训练追加多份历史版本。
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO review_cards (id, session_id, overall, highlights_json, gaps_json, suggested_topics_json, next_training_focus_json, score_breakdown_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			overall = excluded.overall,
			highlights_json = excluded.highlights_json,
			gaps_json = excluded.gaps_json,
			suggested_topics_json = excluded.suggested_topics_json,
			next_training_focus_json = excluded.next_training_focus_json,
			score_breakdown_json = excluded.score_breakdown_json
	`,
		review.ID,
		review.SessionID,
		review.Overall,
		mustJSON(review.Highlights),
		mustJSON(review.Gaps),
		mustJSON(review.SuggestedTopics),
		mustJSON(review.NextTrainingFocus),
		mustJSON(review.ScoreBreakdown),
		nowUTC(),
	)
	if err != nil {
		return fmt.Errorf("create review: %w", err)
	}

	return nil
}

func (s *Store) GetReview(ctx context.Context, reviewID string) (*domain.ReviewCard, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, session_id, overall, highlights_json, gaps_json, suggested_topics_json, next_training_focus_json, score_breakdown_json, created_at
		FROM review_cards
		WHERE id = ?
	`, reviewID)
	review, err := scanReviewCard(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return review, nil
}
