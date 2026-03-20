package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"

	"practicehelper/server/internal/domain"
)

func (s *Store) UpsertWeaknesses(ctx context.Context, sessionID string, hits []domain.WeaknessHit) error {
	now := nowUTC()
	// weakness_tags 存的是聚合后的“热度快照”，不是原始事件日志。
	// 同一轮先按 kind/label 去重，避免一次评估里的重复命中把 severity 人为放大；
	// 后续增量也会做封顶，让长期问题逐步升温，但不会被单次异常值拉爆。
	for _, hit := range dedupeWeaknessHits(hits) {
		if hit.Label == "" || hit.Kind == "" {
			continue
		}

		var existingID string
		var currentSeverity float64
		var frequency int
		err := s.db.QueryRowContext(ctx, `SELECT id, severity, frequency FROM weakness_tags WHERE kind = ? AND label = ?`, hit.Kind, hit.Label).Scan(&existingID, &currentSeverity, &frequency)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			_, err = s.db.ExecContext(ctx, `
				INSERT INTO weakness_tags (id, kind, label, severity, frequency, last_seen_at, evidence_session_id)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, newID("weak"), hit.Kind, hit.Label, math.Min(1.0, hit.Severity), 1, now, sessionID)
		case err == nil:
			_, err = s.db.ExecContext(ctx, `
				UPDATE weakness_tags
				SET severity = ?, frequency = ?, last_seen_at = ?, evidence_session_id = ?
				WHERE id = ?
			`, math.Min(1.5, currentSeverity+(hit.Severity*0.35)), frequency+1, now, sessionID, existingID)
		}
		if err != nil {
			return fmt.Errorf("upsert weakness: %w", err)
		}
	}

	return nil
}

func (s *Store) RelieveWeakness(ctx context.Context, kind, label string, amount float64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE weakness_tags
		SET severity = CASE
			WHEN severity - ? < 0 THEN 0
			ELSE severity - ?
		END
		WHERE kind = ? AND label = ?
	`, amount, amount, kind, label)
	if err != nil {
		return fmt.Errorf("relieve weakness: %w", err)
	}

	return nil
}

func (s *Store) ListWeaknesses(ctx context.Context, limit int) ([]domain.WeaknessTag, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, label, severity, frequency, last_seen_at, evidence_session_id
		FROM weakness_tags
		ORDER BY severity DESC, frequency DESC, last_seen_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list weaknesses: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]domain.WeaknessTag, 0)
	for rows.Next() {
		item, err := scanWeaknessTag(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}

	return items, nil
}
