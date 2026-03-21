package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
	"unicode"

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
		var lastSeenAt string
		err := s.db.QueryRowContext(ctx, `
			SELECT id, severity, frequency, last_seen_at
			FROM weakness_tags
			WHERE kind = ? AND label = ?
		`, hit.Kind, hit.Label).Scan(&existingID, &currentSeverity, &frequency, &lastSeenAt)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			id := newID("weak")
			severity := math.Min(1.0, hit.Severity)
			_, err = s.db.ExecContext(ctx, `
				INSERT INTO weakness_tags (id, kind, label, severity, frequency, last_seen_at, evidence_session_id)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, id, hit.Kind, hit.Label, severity, 1, now, sessionID)
			if err == nil {
				_ = s.recordWeaknessSnapshot(ctx, id, sessionID, severity)
			}
		case err == nil:
			decayed := applyWeaknessTimeDecay(domain.WeaknessTag{
				Severity:   currentSeverity,
				LastSeenAt: parseTime(lastSeenAt),
			}, time.Now().UTC())
			newSeverity := math.Min(1.5, decayed.Severity+(hit.Severity*0.35))
			_, err = s.db.ExecContext(ctx, `
				UPDATE weakness_tags
				SET severity = ?, frequency = ?, last_seen_at = ?, evidence_session_id = ?
				WHERE id = ?
			`, newSeverity, frequency+1, now, sessionID, existingID)
			if err == nil {
				_ = s.recordWeaknessSnapshot(ctx, existingID, sessionID, newSeverity)
			}
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

func (s *Store) RelieveWeaknessesMatchingText(
	ctx context.Context,
	kinds []string,
	text string,
	amount float64,
) error {
	normalizedText := normalizeWeaknessMatchText(text)
	if normalizedText == "" || amount <= 0 {
		return nil
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT kind, label
		FROM weakness_tags
	`)
	if err != nil {
		return fmt.Errorf("query weakness tags for fuzzy relief: %w", err)
	}
	defer func() { _ = rows.Close() }()

	allowedKinds := make(map[string]struct{}, len(kinds))
	for _, kind := range kinds {
		allowedKinds[kind] = struct{}{}
	}

	matches := make([]struct {
		kind  string
		label string
	}, 0)

	for rows.Next() {
		var kind string
		var label string
		if err := rows.Scan(&kind, &label); err != nil {
			return fmt.Errorf("scan weakness tag for fuzzy relief: %w", err)
		}
		if _, ok := allowedKinds[kind]; !ok {
			continue
		}
		normalizedLabel := normalizeWeaknessMatchText(label)
		if len(normalizedLabel) < 4 {
			continue
		}
		if !strings.Contains(normalizedText, normalizedLabel) {
			continue
		}
		matches = append(matches, struct {
			kind  string
			label string
		}{kind: kind, label: label})
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate weakness tags for fuzzy relief: %w", err)
	}

	for _, match := range matches {
		if err := s.RelieveWeakness(ctx, match.kind, match.label, amount); err != nil {
			return err
		}
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
	`)
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
		items = append(items, applyWeaknessTimeDecay(*item, time.Now().UTC()))
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Severity != items[j].Severity {
			return items[i].Severity > items[j].Severity
		}
		if items[i].Frequency != items[j].Frequency {
			return items[i].Frequency > items[j].Frequency
		}
		return items[i].LastSeenAt.After(items[j].LastSeenAt)
	})

	if len(items) > limit {
		items = items[:limit]
	}

	return items, nil
}

func applyWeaknessTimeDecay(item domain.WeaknessTag, now time.Time) domain.WeaknessTag {
	if item.LastSeenAt.IsZero() {
		return item
	}

	staleDays := now.Sub(item.LastSeenAt).Hours() / 24
	if staleDays <= 0 {
		return item
	}

	// 这里不改数据库里的原始 severity，只在读取时算“当前热度”：
	// 最近 3 周内没再出现的弱点会逐步降权，避免一次偶发失误长期霸榜首页推荐。
	decayMultiplier := 1 / (1 + staleDays/21)
	item.Severity = math.Round((item.Severity*decayMultiplier)*100) / 100
	return item
}

func normalizeWeaknessMatchText(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func (s *Store) recordWeaknessSnapshot(ctx context.Context, weaknessID, sessionID string, severity float64) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO weakness_snapshots (weakness_id, session_id, severity, created_at)
		VALUES (?, ?, ?, ?)
	`, weaknessID, sessionID, severity, nowUTC())
	return err
}

func (s *Store) GetWeaknessTrends(ctx context.Context, limit int) ([]domain.WeaknessTrend, error) {
	if limit <= 0 {
		limit = 5
	}

	topWeaknesses, err := s.ListWeaknesses(ctx, limit)
	if err != nil {
		return nil, err
	}

	trends := make([]domain.WeaknessTrend, 0, len(topWeaknesses))
	for _, w := range topWeaknesses {
		rows, err := s.db.QueryContext(ctx, `
			SELECT session_id, severity, created_at
			FROM weakness_snapshots
			WHERE weakness_id = ?
			ORDER BY created_at ASC
			LIMIT 50
		`, w.ID)
		if err != nil {
			return nil, fmt.Errorf("get weakness snapshots: %w", err)
		}

		points := make([]domain.WeaknessTrendPoint, 0)
		for rows.Next() {
			var sessionID, createdAt string
			var severity float64
			if err := rows.Scan(&sessionID, &severity, &createdAt); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("scan weakness snapshot: %w", err)
			}
			points = append(points, domain.WeaknessTrendPoint{
				SessionID: sessionID,
				Severity:  severity,
				CreatedAt: createdAt,
			})
		}
		_ = rows.Close()

		trends = append(trends, domain.WeaknessTrend{
			ID:     w.ID,
			Kind:   w.Kind,
			Label:  w.Label,
			Points: points,
		})
	}

	return trends, nil
}
