package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"practicehelper/server/internal/domain"
)

func (s *Store) CreateReviewSchedule(ctx context.Context, item *domain.ReviewScheduleItem) error {
	var existingID int64
	query := `
		SELECT id
		FROM review_schedule
		WHERE session_id = ? AND weakness_tag_id = ''
		ORDER BY id DESC
		LIMIT 1
	`
	args := []any{item.SessionID}
	if item.WeaknessTagID != "" {
		query = `
			SELECT id
			FROM review_schedule
			WHERE weakness_tag_id = ?
			ORDER BY id DESC
			LIMIT 1
		`
		args = []any{item.WeaknessTagID}
	}

	err := s.db.QueryRowContext(ctx, query, args...).Scan(&existingID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		_, err = s.db.ExecContext(ctx, `
			INSERT INTO review_schedule (session_id, review_card_id, weakness_tag_id, topic, next_review_at, interval_days, ease_factor, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
			item.SessionID,
			item.ReviewCardID,
			item.WeaknessTagID,
			item.Topic,
			item.NextReviewAt.UTC().Format(time.RFC3339),
			item.IntervalDays,
			item.EaseFactor,
			nowUTC(),
			nowUTC(),
		)
		if err != nil {
			return fmt.Errorf("create review schedule: %w", err)
		}
	case err != nil:
		return fmt.Errorf("query review schedule by session: %w", err)
	default:
		_, err = s.db.ExecContext(ctx, `
			UPDATE review_schedule
			SET review_card_id = ?, weakness_tag_id = ?, topic = ?, next_review_at = ?, interval_days = ?, ease_factor = ?, updated_at = ?
			WHERE id = ?
		`,
			item.ReviewCardID,
			item.WeaknessTagID,
			item.Topic,
			item.NextReviewAt.UTC().Format(time.RFC3339),
			item.IntervalDays,
			item.EaseFactor,
			nowUTC(),
			existingID,
		)
		if err != nil {
			return fmt.Errorf("update review schedule: %w", err)
		}
		dedupeQuery := `
			DELETE FROM review_schedule
			WHERE session_id = ? AND weakness_tag_id = '' AND id <> ?
		`
		dedupeArgs := []any{item.SessionID, existingID}
		if item.WeaknessTagID != "" {
			dedupeQuery = `
				DELETE FROM review_schedule
				WHERE weakness_tag_id = ? AND id <> ?
			`
			dedupeArgs = []any{item.WeaknessTagID, existingID}
		}
		_, err = s.db.ExecContext(ctx, dedupeQuery, dedupeArgs...)
		if err != nil {
			return fmt.Errorf("dedupe review schedule: %w", err)
		}
	}
	return nil
}

func (s *Store) GetReviewSchedule(ctx context.Context, id int64) (*domain.ReviewScheduleItem, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT rs.id, rs.session_id, rs.review_card_id, rs.weakness_tag_id,
			COALESCE(wt.kind, ''), COALESCE(wt.label, ''),
			rs.topic, rs.next_review_at, rs.interval_days, rs.ease_factor, rs.created_at
		FROM review_schedule rs
		LEFT JOIN weakness_tags wt ON wt.id = rs.weakness_tag_id
		WHERE rs.id = ?
	`, id)

	var (
		scheduleID                             int64
		sessionID, reviewCardID, weaknessTagID string
		weaknessKind, weaknessLabel, topic     string
		nextReviewAt, createdAt                string
		intervalDays                           int
		easeFactor                             float64
	)
	if err := row.Scan(
		&scheduleID,
		&sessionID,
		&reviewCardID,
		&weaknessTagID,
		&weaknessKind,
		&weaknessLabel,
		&topic,
		&nextReviewAt,
		&intervalDays,
		&easeFactor,
		&createdAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get review schedule: %w", err)
	}

	return &domain.ReviewScheduleItem{
		ID:            scheduleID,
		SessionID:     sessionID,
		ReviewCardID:  reviewCardID,
		WeaknessTagID: weaknessTagID,
		WeaknessKind:  weaknessKind,
		WeaknessLabel: weaknessLabel,
		Topic:         topic,
		NextReviewAt:  parseTime(nextReviewAt),
		IntervalDays:  intervalDays,
		EaseFactor:    easeFactor,
		CreatedAt:     parseTime(createdAt),
	}, nil
}

func (s *Store) ListDueReviews(ctx context.Context, now time.Time) ([]domain.ReviewScheduleItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT rs.id, rs.session_id, rs.review_card_id, rs.weakness_tag_id,
			COALESCE(wt.kind, ''), COALESCE(wt.label, ''),
			rs.topic, rs.next_review_at, rs.interval_days, rs.ease_factor, rs.created_at
		FROM review_schedule rs
		LEFT JOIN weakness_tags wt ON wt.id = rs.weakness_tag_id
		WHERE rs.next_review_at <= ?
		ORDER BY rs.next_review_at ASC
		LIMIT 20
	`, now.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("list due reviews: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.ReviewScheduleItem, 0)
	for rows.Next() {
		var id int64
		var sessionID, reviewCardID, weaknessTagID, weaknessKind, weaknessLabel string
		var topic, nextReviewAt, createdAt string
		var intervalDays int
		var easeFactor float64
		if err := rows.Scan(
			&id,
			&sessionID,
			&reviewCardID,
			&weaknessTagID,
			&weaknessKind,
			&weaknessLabel,
			&topic,
			&nextReviewAt,
			&intervalDays,
			&easeFactor,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan review schedule: %w", err)
		}
		items = append(items, domain.ReviewScheduleItem{
			ID:            id,
			SessionID:     sessionID,
			ReviewCardID:  reviewCardID,
			WeaknessTagID: weaknessTagID,
			WeaknessKind:  weaknessKind,
			WeaknessLabel: weaknessLabel,
			Topic:         topic,
			NextReviewAt:  parseTime(nextReviewAt),
			IntervalDays:  intervalDays,
			EaseFactor:    easeFactor,
			CreatedAt:     parseTime(createdAt),
		})
	}
	return items, nil
}

func (s *Store) CompleteReviewSchedule(ctx context.Context, id int64, score float64) error {
	var intervalDays int
	var easeFactor float64
	err := s.db.QueryRowContext(ctx, `
		SELECT interval_days, ease_factor FROM review_schedule WHERE id = ?
	`, id).Scan(&intervalDays, &easeFactor)
	if err != nil {
		return fmt.Errorf("get review schedule for update: %w", err)
	}

	// Simplified SM-2: adjust ease factor and interval based on score
	var q float64
	switch {
	case score >= 85:
		q = 5
	case score >= 70:
		q = 4
	case score >= 55:
		q = 3
	case score >= 40:
		q = 2
	default:
		q = 1
	}

	newEase := easeFactor + (0.1 - (5-q)*(0.08+(5-q)*0.02))
	if newEase < 1.3 {
		newEase = 1.3
	}

	var newInterval int
	if q < 3 {
		newInterval = 1
	} else if intervalDays <= 1 {
		newInterval = 3
	} else {
		newInterval = int(float64(intervalDays) * newEase)
	}

	nextReview := time.Now().UTC().AddDate(0, 0, newInterval)
	_, err = s.db.ExecContext(ctx, `
		UPDATE review_schedule
		SET interval_days = ?, ease_factor = ?, next_review_at = ?, updated_at = ?
		WHERE id = ?
	`, newInterval, newEase, nextReview.Format(time.RFC3339), nowUTC(), id)
	if err != nil {
		return fmt.Errorf("complete review schedule: %w", err)
	}
	return nil
}
