package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"practicehelper/server/internal/domain"
)

type deleteReviewScheduleState struct {
	WeaknessTagID string
	Topic         string
	NextReviewAt  string
	IntervalDays  int
	EaseFactor    float64
}

type deleteWeaknessTagState struct {
	ID    string
	Kind  string
	Label string
}

type deleteWeaknessRebuildResult struct {
	Exists          bool
	SourceSessionID string
}

type deleteKnowledgeNodeState struct {
	ID        string
	ScopeType string
	ScopeID   string
	ParentID  string
	Label     string
	NodeType  string
}

type deleteKnowledgeSnapshotState struct {
	SessionID   string
	Proficiency float64
	Confidence  float64
	CreatedAt   string
}

type deleteLightweightSession struct {
	ID       string
	Mode     string
	Topic    string
	ReviewID string
}

func (s *Store) DeleteSessions(
	ctx context.Context,
	sessionIDs []string,
	deleteVectorPoints func(context.Context, []string) error,
) (*domain.DeleteSessionsResult, error) {
	sessionIDs = uniqueStrings(sessionIDs)
	if len(sessionIDs) == 0 {
		return &domain.DeleteSessionsResult{}, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin delete sessions: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existingIDs, err := loadExistingSessionIDsTx(ctx, tx, sessionIDs)
	if err != nil {
		return nil, err
	}
	if len(existingIDs) != len(sessionIDs) {
		return nil, nil
	}

	summaryRefs, err := loadSessionScopedRefsTx(ctx, tx, "session_memory_summaries", sessionIDs)
	if err != nil {
		return nil, err
	}
	observationRefs, err := loadSessionScopedRefsTx(ctx, tx, "agent_observations", sessionIDs)
	if err != nil {
		return nil, err
	}
	vectorPointIDs, err := loadMemoryIndexIDsByRefsTx(ctx, tx, append(summaryRefs, observationRefs...))
	if err != nil {
		return nil, err
	}

	affectedWeaknesses, err := loadAffectedWeaknessTagsTx(ctx, tx, sessionIDs)
	if err != nil {
		return nil, err
	}
	scheduleState, err := loadReviewSchedulesByWeaknessTagIDsTx(ctx, tx, collectDeleteWeaknessTagIDs(affectedWeaknesses))
	if err != nil {
		return nil, err
	}
	affectedKnowledgeNodes, err := loadAffectedKnowledgeNodesTx(ctx, tx, sessionIDs)
	if err != nil {
		return nil, err
	}

	if err := deleteRowsByStringFieldTx(ctx, tx, "evaluation_logs", "session_id", sessionIDs); err != nil {
		return nil, err
	}
	if err := deleteRowsByStringFieldTx(ctx, tx, "review_schedule", "session_id", sessionIDs); err != nil {
		return nil, err
	}
	if err := s.deleteMemoryIndexArtifactsByIDsTx(ctx, tx, vectorPointIDs); err != nil {
		return nil, err
	}
	if err := deleteRowsByStringFieldTx(ctx, tx, "agent_observations", "session_id", sessionIDs); err != nil {
		return nil, err
	}
	if err := deleteRowsByStringFieldTx(ctx, tx, "session_memory_summaries", "session_id", sessionIDs); err != nil {
		return nil, err
	}
	if err := deleteRowsByStringFieldTx(ctx, tx, "knowledge_snapshots", "session_id", sessionIDs); err != nil {
		return nil, err
	}
	if err := deleteRowsByStringFieldTx(ctx, tx, "weakness_snapshots", "session_id", sessionIDs); err != nil {
		return nil, err
	}
	if err := deleteRowsByStringFieldTx(ctx, tx, "training_sessions", "id", sessionIDs); err != nil {
		return nil, err
	}

	knowledgeVectorIDs, err := s.reconcileKnowledgeNodesTx(ctx, tx, affectedKnowledgeNodes)
	if err != nil {
		return nil, err
	}
	weaknessState, err := s.reconcileWeaknessTagsTx(ctx, tx, affectedWeaknesses)
	if err != nil {
		return nil, err
	}
	if err := s.reconcileReviewSchedulesTx(ctx, tx, scheduleState, weaknessState); err != nil {
		return nil, err
	}

	vectorPointIDs = uniqueStrings(append(vectorPointIDs, knowledgeVectorIDs...))
	if deleteVectorPoints != nil && len(vectorPointIDs) > 0 {
		if err := deleteVectorPoints(ctx, vectorPointIDs); err != nil {
			return nil, fmt.Errorf("delete vector points: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit delete sessions: %w", err)
	}

	return &domain.DeleteSessionsResult{
		DeletedCount:      len(sessionIDs),
		DeletedSessionIDs: sessionIDs,
	}, nil
}

func (s *Store) reconcileWeaknessTagsTx(
	ctx context.Context,
	tx *sql.Tx,
	items []deleteWeaknessTagState,
) (map[string]deleteWeaknessRebuildResult, error) {
	results := make(map[string]deleteWeaknessRebuildResult, len(items))
	for _, item := range items {
		rows, err := tx.QueryContext(ctx, `
			SELECT session_id, severity, created_at
			FROM weakness_snapshots
			WHERE weakness_id = ?
			ORDER BY created_at ASC, id ASC
		`, item.ID)
		if err != nil {
			return nil, fmt.Errorf("list remaining weakness snapshots %s: %w", item.ID, err)
		}

		var (
			hasAny          bool
			finalSeverity   float64
			frequency       int
			evidenceSession string
			lastSeenAt      string
		)
		for rows.Next() {
			var (
				sessionID string
				severity  float64
				createdAt string
			)
			if err := rows.Scan(&sessionID, &severity, &createdAt); err != nil {
				_ = rows.Close()
				return nil, fmt.Errorf("scan weakness snapshot %s: %w", item.ID, err)
			}
			hasAny = true
			finalSeverity = severity
			if sessionID != "" {
				frequency++
				evidenceSession = sessionID
				lastSeenAt = createdAt
			}
		}
		_ = rows.Close()

		if !hasAny || frequency == 0 || evidenceSession == "" {
			if _, err := tx.ExecContext(ctx, `DELETE FROM weakness_snapshots WHERE weakness_id = ?`, item.ID); err != nil {
				return nil, fmt.Errorf("delete orphan weakness snapshots %s: %w", item.ID, err)
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM weakness_tags WHERE id = ?`, item.ID); err != nil {
				return nil, fmt.Errorf("delete orphan weakness tag %s: %w", item.ID, err)
			}
			results[item.ID] = deleteWeaknessRebuildResult{}
			continue
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE weakness_tags
			SET severity = ?, frequency = ?, last_seen_at = ?, evidence_session_id = ?
			WHERE id = ?
		`, finalSeverity, frequency, lastSeenAt, evidenceSession, item.ID); err != nil {
			return nil, fmt.Errorf("rebuild weakness tag %s: %w", item.ID, err)
		}
		results[item.ID] = deleteWeaknessRebuildResult{
			Exists:          true,
			SourceSessionID: evidenceSession,
		}
	}
	return results, nil
}

func (s *Store) reconcileReviewSchedulesTx(
	ctx context.Context,
	tx *sql.Tx,
	schedules map[string]deleteReviewScheduleState,
	weaknessState map[string]deleteWeaknessRebuildResult,
) error {
	for weaknessTagID, schedule := range schedules {
		state, ok := weaknessState[weaknessTagID]
		if !ok || !state.Exists || state.SourceSessionID == "" {
			if _, err := tx.ExecContext(ctx, `DELETE FROM review_schedule WHERE weakness_tag_id = ?`, weaknessTagID); err != nil {
				return fmt.Errorf("delete review schedule for weakness %s: %w", weaknessTagID, err)
			}
			continue
		}

		session, err := loadLightweightSessionTx(ctx, tx, state.SourceSessionID)
		if err != nil {
			return err
		}
		if session == nil {
			if _, err := tx.ExecContext(ctx, `DELETE FROM review_schedule WHERE weakness_tag_id = ?`, weaknessTagID); err != nil {
				return fmt.Errorf("delete dangling review schedule for weakness %s: %w", weaknessTagID, err)
			}
			continue
		}

		topic := normalizeTopicLabel(schedule.Topic)
		if topic == "" && session.Mode == domain.ModeBasics {
			topic = normalizeTopicLabel(session.Topic)
			if topic == domain.BasicsTopicMixed {
				topic = ""
			}
		}
		if topic == "" {
			topic = domain.BasicsTopicGo
		}

		var existingID int64
		err = tx.QueryRowContext(ctx, `
			SELECT id
			FROM review_schedule
			WHERE weakness_tag_id = ?
			ORDER BY id DESC
			LIMIT 1
		`, weaknessTagID).Scan(&existingID)
		switch err {
		case nil:
			if _, err := tx.ExecContext(ctx, `
				UPDATE review_schedule
				SET session_id = ?, review_card_id = ?, topic = ?, next_review_at = ?, interval_days = ?, ease_factor = ?, updated_at = ?
				WHERE id = ?
			`,
				session.ID,
				session.ReviewID,
				topic,
				schedule.NextReviewAt,
				schedule.IntervalDays,
				schedule.EaseFactor,
				nowUTC(),
				existingID,
			); err != nil {
				return fmt.Errorf("update review schedule %d: %w", existingID, err)
			}
		case sql.ErrNoRows:
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO review_schedule (
					session_id, review_card_id, weakness_tag_id, topic, next_review_at, interval_days, ease_factor, created_at, updated_at
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			`,
				session.ID,
				session.ReviewID,
				weaknessTagID,
				topic,
				schedule.NextReviewAt,
				schedule.IntervalDays,
				schedule.EaseFactor,
				nowUTC(),
				nowUTC(),
			); err != nil {
				return fmt.Errorf("insert review schedule for weakness %s: %w", weaknessTagID, err)
			}
		default:
			return fmt.Errorf("load review schedule for weakness %s: %w", weaknessTagID, err)
		}
	}
	return nil
}

func (s *Store) reconcileKnowledgeNodesTx(
	ctx context.Context,
	tx *sql.Tx,
	nodes []deleteKnowledgeNodeState,
) ([]string, error) {
	vectorPointIDs := make([]string, 0)
	for _, node := range nodes {
		snapshots, err := loadKnowledgeSnapshotsByNodeIDTx(ctx, tx, node.ID)
		if err != nil {
			return nil, err
		}
		if len(snapshots) == 0 {
			if node.NodeType == domain.KnowledgeNodeTypeTopic {
				if _, err := tx.ExecContext(ctx, `
					UPDATE knowledge_nodes
					SET proficiency = 0, confidence = 0.5, hit_count = 0, last_assessed_at = '', updated_at = ?
					WHERE id = ?
				`, nowUTC(), node.ID); err != nil {
					return nil, fmt.Errorf("reset root knowledge node %s: %w", node.ID, err)
				}
				continue
			}

			ids, err := s.deleteMemoryIndexArtifactsByRefsTx(ctx, tx, []domain.MemoryRef{{
				RefTable: "knowledge_nodes",
				RefID:    node.ID,
			}})
			if err != nil {
				return nil, err
			}
			vectorPointIDs = append(vectorPointIDs, ids...)

			if _, err := tx.ExecContext(ctx, `
				DELETE FROM knowledge_edges
				WHERE source_id = ? OR target_id = ?
			`, node.ID, node.ID); err != nil {
				return nil, fmt.Errorf("delete knowledge edges for %s: %w", node.ID, err)
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM knowledge_nodes WHERE id = ?`, node.ID); err != nil {
				return nil, fmt.Errorf("delete knowledge node %s: %w", node.ID, err)
			}
			continue
		}

		last := snapshots[len(snapshots)-1]
		if _, err := tx.ExecContext(ctx, `
			UPDATE knowledge_nodes
			SET proficiency = ?, confidence = ?, hit_count = ?, last_assessed_at = ?, updated_at = ?
			WHERE id = ?
		`, last.Proficiency, last.Confidence, len(snapshots), last.CreatedAt, nowUTC(), node.ID); err != nil {
			return nil, fmt.Errorf("rebuild knowledge node %s: %w", node.ID, err)
		}

		topic, err := s.resolveRootTopicTx(ctx, tx, &domain.KnowledgeNode{
			ID:         node.ID,
			ScopeType:  node.ScopeType,
			ScopeID:    node.ScopeID,
			ParentID:   node.ParentID,
			Label:      node.Label,
			NodeType:   node.NodeType,
			Confidence: last.Confidence,
		})
		if err != nil {
			return nil, fmt.Errorf("resolve root topic for knowledge node %s: %w", node.ID, err)
		}

		if err := s.upsertMemoryIndexEntries(ctx, tx, []domain.MemoryIndexEntry{{
			MemoryType: "knowledge_node",
			ScopeType:  node.ScopeType,
			ScopeID:    node.ScopeID,
			Topic:      topic,
			SessionID:  last.SessionID,
			Summary:    node.Label,
			Salience:   0.55,
			Confidence: last.Confidence,
			Freshness:  1.0,
			RefTable:   "knowledge_nodes",
			RefID:      node.ID,
		}}); err != nil {
			return nil, fmt.Errorf("rebuild knowledge memory index %s: %w", node.ID, err)
		}
	}
	return uniqueStrings(vectorPointIDs), nil
}

func (s *Store) deleteMemoryIndexArtifactsByRefsTx(
	ctx context.Context,
	tx *sql.Tx,
	refs []domain.MemoryRef,
) ([]string, error) {
	ids, err := loadMemoryIndexIDsByRefsTx(ctx, tx, refs)
	if err != nil {
		return nil, err
	}
	if err := s.deleteMemoryIndexArtifactsByIDsTx(ctx, tx, ids); err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *Store) deleteMemoryIndexArtifactsByIDsTx(
	ctx context.Context,
	tx *sql.Tx,
	memoryIndexIDs []string,
) error {
	memoryIndexIDs = uniqueStrings(memoryIndexIDs)
	if len(memoryIndexIDs) == 0 {
		return nil
	}

	if err := deleteRowsByStringFieldTx(ctx, tx, "memory_embedding_records", "memory_index_id", memoryIndexIDs); err != nil {
		return err
	}
	if err := deleteRowsByStringFieldTx(ctx, tx, "memory_embedding_jobs", "memory_index_id", memoryIndexIDs); err != nil {
		return err
	}
	if err := deleteRowsByStringFieldTx(ctx, tx, "memory_index", "id", memoryIndexIDs); err != nil {
		return err
	}
	return nil
}

func loadExistingSessionIDsTx(
	ctx context.Context,
	tx *sql.Tx,
	sessionIDs []string,
) ([]string, error) {
	return loadStringColumnTx(ctx, tx, "training_sessions", "id", "id", sessionIDs)
}

func loadSessionScopedRefsTx(
	ctx context.Context,
	tx *sql.Tx,
	table string,
	sessionIDs []string,
) ([]domain.MemoryRef, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(sessionIDs)), ",")
	args := make([]any, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		args = append(args, id)
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT id
		FROM %s
		WHERE session_id IN (%s)
	`, table, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("list %s refs: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	refs := make([]domain.MemoryRef, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan %s ref: %w", table, err)
		}
		refs = append(refs, domain.MemoryRef{RefTable: table, RefID: id})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s refs: %w", table, err)
	}
	return refs, nil
}

func loadMemoryIndexIDsByRefsTx(
	ctx context.Context,
	tx *sql.Tx,
	refs []domain.MemoryRef,
) ([]string, error) {
	refs = uniqueMemoryRefs(refs)
	if len(refs) == 0 {
		return nil, nil
	}

	clauses := make([]string, 0, len(refs))
	args := make([]any, 0, len(refs)*2)
	for _, ref := range refs {
		if ref.RefTable == "" || ref.RefID == "" {
			continue
		}
		clauses = append(clauses, "(ref_table = ? AND ref_id = ?)")
		args = append(args, ref.RefTable, ref.RefID)
	}
	if len(clauses) == 0 {
		return nil, nil
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT id
		FROM memory_index
		WHERE %s
	`, strings.Join(clauses, " OR ")), args...)
	if err != nil {
		return nil, fmt.Errorf("list memory index ids by refs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	ids := make([]string, 0, len(refs))
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan memory index id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memory index ids by refs: %w", err)
	}
	return uniqueStrings(ids), nil
}

func loadAffectedWeaknessTagsTx(
	ctx context.Context,
	tx *sql.Tx,
	sessionIDs []string,
) ([]deleteWeaknessTagState, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(sessionIDs)), ",")
	args := make([]any, 0, len(sessionIDs)*2)
	for _, id := range sessionIDs {
		args = append(args, id)
	}
	for _, id := range sessionIDs {
		args = append(args, id)
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT DISTINCT wt.id, wt.kind, wt.label
		FROM weakness_tags wt
		LEFT JOIN weakness_snapshots ws ON ws.weakness_id = wt.id
		WHERE wt.evidence_session_id IN (%s) OR ws.session_id IN (%s)
	`, placeholders, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("list affected weakness tags: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]deleteWeaknessTagState, 0)
	for rows.Next() {
		var item deleteWeaknessTagState
		if err := rows.Scan(&item.ID, &item.Kind, &item.Label); err != nil {
			return nil, fmt.Errorf("scan affected weakness tag: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate affected weakness tags: %w", err)
	}
	return items, nil
}

func loadReviewSchedulesByWeaknessTagIDsTx(
	ctx context.Context,
	tx *sql.Tx,
	weaknessTagIDs []string,
) (map[string]deleteReviewScheduleState, error) {
	weaknessTagIDs = uniqueStrings(weaknessTagIDs)
	if len(weaknessTagIDs) == 0 {
		return map[string]deleteReviewScheduleState{}, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(weaknessTagIDs)), ",")
	args := make([]any, 0, len(weaknessTagIDs))
	for _, id := range weaknessTagIDs {
		args = append(args, id)
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT weakness_tag_id, topic, next_review_at, interval_days, ease_factor
		FROM review_schedule
		WHERE weakness_tag_id IN (%s)
		ORDER BY id DESC
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("list review schedules by weakness tag: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make(map[string]deleteReviewScheduleState, len(weaknessTagIDs))
	for rows.Next() {
		var item deleteReviewScheduleState
		if err := rows.Scan(
			&item.WeaknessTagID,
			&item.Topic,
			&item.NextReviewAt,
			&item.IntervalDays,
			&item.EaseFactor,
		); err != nil {
			return nil, fmt.Errorf("scan review schedule state: %w", err)
		}
		if _, exists := items[item.WeaknessTagID]; !exists {
			items[item.WeaknessTagID] = item
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate review schedule states: %w", err)
	}
	return items, nil
}

func loadAffectedKnowledgeNodesTx(
	ctx context.Context,
	tx *sql.Tx,
	sessionIDs []string,
) ([]deleteKnowledgeNodeState, error) {
	if len(sessionIDs) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(sessionIDs)), ",")
	args := make([]any, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		args = append(args, id)
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT DISTINCT kn.id, kn.scope_type, kn.scope_id, kn.parent_id, kn.label, kn.node_type
		FROM knowledge_nodes kn
		JOIN knowledge_snapshots ks ON ks.node_id = kn.id
		WHERE ks.session_id IN (%s)
	`, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("list affected knowledge nodes: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]deleteKnowledgeNodeState, 0)
	for rows.Next() {
		var item deleteKnowledgeNodeState
		if err := rows.Scan(
			&item.ID,
			&item.ScopeType,
			&item.ScopeID,
			&item.ParentID,
			&item.Label,
			&item.NodeType,
		); err != nil {
			return nil, fmt.Errorf("scan affected knowledge node: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate affected knowledge nodes: %w", err)
	}
	return items, nil
}

func loadKnowledgeSnapshotsByNodeIDTx(
	ctx context.Context,
	tx *sql.Tx,
	nodeID string,
) ([]deleteKnowledgeSnapshotState, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT session_id, proficiency, COALESCE(confidence, 0.5), created_at
		FROM knowledge_snapshots
		WHERE node_id = ?
		ORDER BY created_at ASC, id ASC
	`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("list knowledge snapshots for %s: %w", nodeID, err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]deleteKnowledgeSnapshotState, 0)
	for rows.Next() {
		var item deleteKnowledgeSnapshotState
		if err := rows.Scan(&item.SessionID, &item.Proficiency, &item.Confidence, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan knowledge snapshot for %s: %w", nodeID, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate knowledge snapshots for %s: %w", nodeID, err)
	}
	return items, nil
}

func loadLightweightSessionTx(
	ctx context.Context,
	tx *sql.Tx,
	sessionID string,
) (*deleteLightweightSession, error) {
	row := tx.QueryRowContext(ctx, `
		SELECT id, mode, topic, review_id
		FROM training_sessions
		WHERE id = ?
	`, sessionID)
	var item deleteLightweightSession
	if err := row.Scan(&item.ID, &item.Mode, &item.Topic, &item.ReviewID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("load lightweight session %s: %w", sessionID, err)
	}
	return &item, nil
}

func deleteRowsByStringFieldTx(
	ctx context.Context,
	tx *sql.Tx,
	table string,
	column string,
	ids []string,
) error {
	ids = uniqueStrings(ids)
	if len(ids) == 0 {
		return nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}

	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		DELETE FROM %s
		WHERE %s IN (%s)
	`, table, column, placeholders), args...); err != nil {
		return fmt.Errorf("delete %s by %s: %w", table, column, err)
	}
	return nil
}

func loadStringColumnTx(
	ctx context.Context,
	tx *sql.Tx,
	table string,
	column string,
	matchColumn string,
	values []string,
) ([]string, error) {
	values = uniqueStrings(values)
	if len(values) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimRight(strings.Repeat("?,", len(values)), ",")
	args := make([]any, 0, len(values))
	for _, value := range values {
		args = append(args, value)
	}

	rows, err := tx.QueryContext(ctx, fmt.Sprintf(`
		SELECT %s
		FROM %s
		WHERE %s IN (%s)
	`, column, table, matchColumn, placeholders), args...)
	if err != nil {
		return nil, fmt.Errorf("list %s.%s: %w", table, column, err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]string, 0, len(values))
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			return nil, fmt.Errorf("scan %s.%s: %w", table, column, err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate %s.%s: %w", table, column, err)
	}
	return uniqueStrings(items), nil
}

func collectDeleteWeaknessTagIDs(items []deleteWeaknessTagState) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return uniqueStrings(ids)
}

func uniqueMemoryRefs(refs []domain.MemoryRef) []domain.MemoryRef {
	seen := make(map[string]struct{}, len(refs))
	items := make([]domain.MemoryRef, 0, len(refs))
	for _, ref := range refs {
		key := ref.RefTable + "\x00" + ref.RefID
		if ref.RefTable == "" || ref.RefID == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		items = append(items, ref)
	}
	return items
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
