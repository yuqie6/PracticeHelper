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
		INSERT INTO review_cards (id, session_id, overall, top_fix, top_fix_reason, highlights_json, gaps_json, suggested_topics_json, next_training_focus_json, recommended_next_json, retrieval_trace_json, score_breakdown_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			id = excluded.id,
			overall = excluded.overall,
			top_fix = excluded.top_fix,
			top_fix_reason = excluded.top_fix_reason,
			highlights_json = excluded.highlights_json,
			gaps_json = excluded.gaps_json,
			suggested_topics_json = excluded.suggested_topics_json,
			next_training_focus_json = excluded.next_training_focus_json,
			recommended_next_json = excluded.recommended_next_json,
			retrieval_trace_json = excluded.retrieval_trace_json,
			score_breakdown_json = excluded.score_breakdown_json,
			created_at = excluded.created_at
	`,
		review.ID,
		review.SessionID,
		review.Overall,
		review.TopFix,
		review.TopFixReason,
		mustJSON(review.Highlights),
		mustJSON(review.Gaps),
		mustJSON(review.SuggestedTopics),
		mustJSON(review.NextTrainingFocus),
		mustJSON(review.RecommendedNext),
		mustJSON(review.RetrievalTrace),
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
		SELECT rc.id, rc.session_id, ts.job_target_id, ts.job_target_analysis_id, ts.prompt_set_id, ts.prompt_set_label, ts.prompt_set_status,
			COALESCE(ts.prompt_overlay_json, 'null'), COALESCE(ts.prompt_overlay_hash, ''),
			rc.overall, rc.top_fix, rc.top_fix_reason,
			rc.highlights_json, rc.gaps_json, rc.suggested_topics_json, rc.next_training_focus_json,
			rc.recommended_next_json, rc.retrieval_trace_json, rc.score_breakdown_json, rc.created_at
		FROM review_cards rc
		JOIN training_sessions ts ON ts.id = rc.session_id
		WHERE rc.id = ?
	`, reviewID)
	review, err := scanReviewCard(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if review.JobTargetID != "" {
		target, err := s.getJobTargetRow(ctx, review.JobTargetID)
		if err != nil {
			return nil, err
		}
		if target != nil {
			review.JobTarget = &domain.JobTargetRef{
				ID:          target.ID,
				Title:       target.Title,
				CompanyName: target.CompanyName,
			}
		}
	}

	return review, nil
}
