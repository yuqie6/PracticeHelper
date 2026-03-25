package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"

	"practicehelper/server/internal/domain"
)

func TestDeleteSessionsRemovesSessionScopedRecordsAndRebindsWeaknessSchedule(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	first := seedSessionDeletionFixture(t, store, deleteSessionFixture{
		SessionID:     "sess_delete_first",
		Topic:         domain.BasicsTopicRedis,
		WeaknessLabel: "缓存一致性",
		ConceptLabel:  "cache_consistency",
	})
	second := seedSessionDeletionFixture(t, store, deleteSessionFixture{
		SessionID:     "sess_delete_second",
		Topic:         domain.BasicsTopicRedis,
		WeaknessLabel: "缓存一致性",
		ConceptLabel:  "cache_consistency",
	})

	tag, err := store.GetWeaknessTag(ctx, "detail", "缓存一致性")
	if err != nil {
		t.Fatalf("GetWeaknessTag() error = %v", err)
	}
	if tag == nil {
		t.Fatal("expected shared weakness tag")
	}

	nextReviewAt := time.Date(2026, 3, 28, 9, 0, 0, 0, time.UTC)
	if err := store.CreateReviewSchedule(ctx, &domain.ReviewScheduleItem{
		SessionID:     first.ID,
		ReviewCardID:  first.ReviewID,
		WeaknessTagID: tag.ID,
		Topic:         domain.BasicsTopicRedis,
		NextReviewAt:  nextReviewAt,
		IntervalDays:  6,
		EaseFactor:    2.9,
	}); err != nil {
		t.Fatalf("CreateReviewSchedule() error = %v", err)
	}
	firstSummaryIndexID := loadMemoryIndexIDByRef(
		t,
		store.db,
		"session_memory_summaries",
		"summary_"+first.ID,
	)
	firstObservationIndexID := loadMemoryIndexIDByRef(
		t,
		store.db,
		"agent_observations",
		"obs_"+first.ID,
	)

	deletedVectorIDs := make([]string, 0)
	result, err := store.DeleteSessions(ctx, []string{first.ID}, func(_ context.Context, ids []string) error {
		deletedVectorIDs = append(deletedVectorIDs, ids...)
		return nil
	})
	if err != nil {
		t.Fatalf("DeleteSessions() error = %v", err)
	}
	if result == nil || result.DeletedCount != 1 {
		t.Fatalf("unexpected delete result: %+v", result)
	}

	assertSessionDeleted(t, store, first.ID)
	assertRowCountEquals(t, store.db, "evaluation_logs", "session_id = ?", 0, first.ID)
	assertRowCountEquals(t, store.db, "session_memory_summaries", "session_id = ?", 0, first.ID)
	assertRowCountEquals(t, store.db, "agent_observations", "session_id = ?", 0, first.ID)
	assertRowCountEquals(t, store.db, "knowledge_snapshots", "session_id = ?", 0, first.ID)

	updatedTag, err := store.GetWeaknessTag(ctx, "detail", "缓存一致性")
	if err != nil {
		t.Fatalf("GetWeaknessTag() after delete error = %v", err)
	}
	if updatedTag == nil {
		t.Fatal("expected shared weakness tag to survive")
	}
	if updatedTag.Frequency != 1 {
		t.Fatalf("expected weakness frequency=1 after delete, got %d", updatedTag.Frequency)
	}
	if updatedTag.EvidenceSessionID != second.ID {
		t.Fatalf("expected weakness evidence to rebind to %s, got %s", second.ID, updatedTag.EvidenceSessionID)
	}

	var (
		scheduleSessionID string
		reviewCardID      string
		topic             string
		intervalDays      int
		easeFactor        float64
		storedNextReview  string
	)
	if err := store.db.QueryRow(`
		SELECT session_id, review_card_id, topic, interval_days, ease_factor, next_review_at
		FROM review_schedule
		WHERE weakness_tag_id = ?
	`, updatedTag.ID).Scan(
		&scheduleSessionID,
		&reviewCardID,
		&topic,
		&intervalDays,
		&easeFactor,
		&storedNextReview,
	); err != nil {
		t.Fatalf("query rebound review schedule: %v", err)
	}
	if scheduleSessionID != second.ID {
		t.Fatalf("expected review schedule to point at %s, got %s", second.ID, scheduleSessionID)
	}
	if reviewCardID != second.ReviewID {
		t.Fatalf("expected review schedule review_id=%s, got %s", second.ReviewID, reviewCardID)
	}
	if topic != domain.BasicsTopicRedis {
		t.Fatalf("expected rebound schedule topic redis, got %s", topic)
	}
	if intervalDays != 6 || easeFactor != 2.9 {
		t.Fatalf("expected preserved review interval/ease, got interval=%d ease=%.2f", intervalDays, easeFactor)
	}
	if storedNextReview != nextReviewAt.Format(time.RFC3339) {
		t.Fatalf("expected preserved next review %s, got %s", nextReviewAt.Format(time.RFC3339), storedNextReview)
	}

	if !containsString(deletedVectorIDs, firstSummaryIndexID) {
		t.Fatalf("expected session summary vector ids to be purged, got %v", deletedVectorIDs)
	}
	if !containsString(deletedVectorIDs, firstObservationIndexID) {
		t.Fatalf("expected observation vector ids to be purged, got %v", deletedVectorIDs)
	}
}

func TestDeleteSessionsDeletesOrphanWeaknessAndKnowledgeArtifacts(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	session := seedSessionDeletionFixture(t, store, deleteSessionFixture{
		SessionID:     "sess_delete_orphan",
		Topic:         domain.BasicsTopicKafka,
		WeaknessLabel: "offset 提交时机",
		ConceptLabel:  "kafka_offset_commit",
	})

	tag, err := store.GetWeaknessTag(ctx, "detail", "offset 提交时机")
	if err != nil {
		t.Fatalf("GetWeaknessTag() error = %v", err)
	}
	if tag == nil {
		t.Fatal("expected weakness tag for orphan delete case")
	}
	if err := store.CreateReviewSchedule(ctx, &domain.ReviewScheduleItem{
		SessionID:     session.ID,
		ReviewCardID:  session.ReviewID,
		WeaknessTagID: tag.ID,
		Topic:         domain.BasicsTopicKafka,
		NextReviewAt:  time.Now().UTC().Add(24 * time.Hour),
		IntervalDays:  3,
		EaseFactor:    2.6,
	}); err != nil {
		t.Fatalf("CreateReviewSchedule() error = %v", err)
	}

	conceptNodeID := loadKnowledgeNodeIDByLabel(t, store.db, "kafka_offset_commit")
	knowledgeIndexID := loadMemoryIndexIDByRef(t, store.db, "knowledge_nodes", conceptNodeID)

	deletedVectorIDs := make([]string, 0)
	if _, err := store.DeleteSessions(ctx, []string{session.ID}, func(_ context.Context, ids []string) error {
		deletedVectorIDs = append(deletedVectorIDs, ids...)
		return nil
	}); err != nil {
		t.Fatalf("DeleteSessions() error = %v", err)
	}

	assertSessionDeleted(t, store, session.ID)
	if deletedTag, err := store.GetWeaknessTag(ctx, "detail", "offset 提交时机"); err != nil {
		t.Fatalf("GetWeaknessTag() after delete error = %v", err)
	} else if deletedTag != nil {
		t.Fatalf("expected weakness tag to be deleted, got %+v", deletedTag)
	}
	assertRowCountEquals(t, store.db, "review_schedule", "weakness_tag_id = ?", 0, tag.ID)
	assertRowCountEquals(t, store.db, "knowledge_nodes", "id = ?", 0, conceptNodeID)
	assertRowCountEquals(t, store.db, "memory_index", "ref_table = ? AND ref_id = ?", 0, "knowledge_nodes", conceptNodeID)

	if !containsString(deletedVectorIDs, knowledgeIndexID) {
		t.Fatalf("expected knowledge node vector id %s to be purged, got %v", knowledgeIndexID, deletedVectorIDs)
	}
}

func TestDeleteSessionsRollsBackWhenVectorPurgeFails(t *testing.T) {
	store, err := openTestStore(t)
	if err != nil {
		t.Fatalf("openTestStore() error = %v", err)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	session := seedSessionDeletionFixture(t, store, deleteSessionFixture{
		SessionID:     "sess_delete_rollback",
		Topic:         domain.BasicsTopicGo,
		WeaknessLabel: "goroutine 泄漏排查",
	})

	if _, err := store.DeleteSessions(ctx, []string{session.ID}, func(_ context.Context, ids []string) error {
		return fmt.Errorf("qdrant unavailable for %v", ids)
	}); err == nil {
		t.Fatal("expected vector purge failure to abort deletion")
	}

	if saved, err := store.GetSession(ctx, session.ID); err != nil {
		t.Fatalf("GetSession() after rollback error = %v", err)
	} else if saved == nil {
		t.Fatal("expected session to still exist after rollback")
	}
	assertRowCountEquals(t, store.db, "session_memory_summaries", "session_id = ?", 1, session.ID)
	assertRowCountEquals(t, store.db, "agent_observations", "session_id = ?", 1, session.ID)
}

type deleteSessionFixture struct {
	SessionID     string
	Topic         string
	WeaknessLabel string
	ConceptLabel  string
}

func seedSessionDeletionFixture(
	t *testing.T,
	store *Store,
	fixture deleteSessionFixture,
) *domain.TrainingSession {
	t.Helper()

	ctx := context.Background()
	session := &domain.TrainingSession{
		ID:         fixture.SessionID,
		Mode:       domain.ModeBasics,
		Topic:      fixture.Topic,
		Intensity:  "standard",
		Status:     domain.StatusCompleted,
		MaxTurns:   1,
		TotalScore: 78,
		ReviewID:   "review_" + fixture.SessionID,
	}
	turn := &domain.TrainingTurn{
		ID:             "turn_" + fixture.SessionID,
		SessionID:      fixture.SessionID,
		TurnIndex:      1,
		Stage:          "question",
		Question:       "请解释一下这一轮的核心点。",
		ExpectedPoints: []string{"先说主线", "补细节"},
		Answer:         "我会先说主线，再补关键细节。",
		WeaknessHits: []domain.WeaknessHit{{
			Kind:     "detail",
			Label:    fixture.WeaknessLabel,
			Severity: 0.92,
		}},
	}
	if err := store.CreateSession(ctx, session, turn); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if err := store.CreateReview(ctx, &domain.ReviewCard{
		ID:                session.ReviewID,
		SessionID:         session.ID,
		Overall:           "还需要补关键细节。",
		Highlights:        []string{"主线明确"},
		Gaps:              []string{"关键细节没展开"},
		SuggestedTopics:   []string{fixture.Topic},
		NextTrainingFocus: []string{"补关键细节"},
		ScoreBreakdown:    map[string]float64{"准确性": 78},
	}); err != nil {
		t.Fatalf("CreateReview() error = %v", err)
	}
	if err := store.UpsertWeaknesses(ctx, session.ID, []domain.WeaknessHit{{
		Kind:     "detail",
		Label:    fixture.WeaknessLabel,
		Severity: 0.92,
	}}); err != nil {
		t.Fatalf("UpsertWeaknesses() error = %v", err)
	}
	if err := store.CreateEvaluationLog(
		ctx,
		session.ID,
		turn.ID,
		"generate_review",
		"mock-model",
		"",
		"",
		"raw-output",
		nil,
		123,
	); err != nil {
		t.Fatalf("CreateEvaluationLog() error = %v", err)
	}
	if err := store.UpsertSessionMemorySummary(ctx, &domain.SessionMemorySummary{
		ID:               "summary_" + fixture.SessionID,
		SessionID:        session.ID,
		Mode:             session.Mode,
		Topic:            fixture.Topic,
		Summary:          "这一轮的训练总结。",
		RecommendedFocus: []string{"补关键细节"},
		Salience:         0.8,
	}); err != nil {
		t.Fatalf("UpsertSessionMemorySummary() error = %v", err)
	}
	if err := store.CreateObservations(ctx, session.ID, []domain.AgentObservation{{
		ID:        "obs_" + fixture.SessionID,
		SessionID: session.ID,
		ScopeType: domain.MemoryScopeGlobal,
		Topic:     fixture.Topic,
		Category:  domain.ObservationCategoryPattern,
		Content:   "用户回答主线稳定，但细节支撑偏少。",
		Tags:      []string{"detail"},
		Relevance: 0.76,
	}}); err != nil {
		t.Fatalf("CreateObservations() error = %v", err)
	}
	if fixture.ConceptLabel != "" {
		if err := store.EnsureKnowledgeSeeds(ctx); err != nil {
			t.Fatalf("EnsureKnowledgeSeeds() error = %v", err)
		}
		subgraph, err := store.GetKnowledgeSubgraph(ctx, fixture.Topic, "", 4)
		if err != nil {
			t.Fatalf("GetKnowledgeSubgraph() error = %v", err)
		}
		if len(subgraph.Nodes) == 0 {
			t.Fatalf("expected seeded topic node for %s", fixture.Topic)
		}
		if err := store.UpsertKnowledgeNodes(ctx, session.ID, []domain.KnowledgeUpdate{{
			ScopeType:   domain.MemoryScopeGlobal,
			ParentID:    subgraph.Nodes[0].ID,
			Label:       fixture.ConceptLabel,
			NodeType:    domain.KnowledgeNodeTypeConcept,
			Proficiency: 1.35,
			Confidence:  0.82,
			Evidence:    "这一轮对关键概念还不够稳。",
		}}); err != nil {
			t.Fatalf("UpsertKnowledgeNodes() error = %v", err)
		}
	}

	insertEmbeddingArtifactsForSession(t, store, session.ID, fixture.ConceptLabel)
	return session
}

func insertEmbeddingArtifactsForSession(
	t *testing.T,
	store *Store,
	sessionID string,
	conceptLabel string,
) {
	t.Helper()

	ctx := context.Background()
	refs := []domain.MemoryRef{
		{RefTable: "session_memory_summaries", RefID: "summary_" + sessionID},
		{RefTable: "agent_observations", RefID: "obs_" + sessionID},
	}
	if conceptLabel != "" {
		nodeID := loadKnowledgeNodeIDByLabel(t, store.db, conceptLabel)
		refs = append(refs, domain.MemoryRef{RefTable: "knowledge_nodes", RefID: nodeID})
	}
	entries, err := store.GetMemoryIndexEntriesByRefs(ctx, refs)
	if err != nil {
		t.Fatalf("GetMemoryIndexEntriesByRefs() error = %v", err)
	}
	for _, entry := range entries {
		if _, err := store.db.Exec(`
			INSERT OR REPLACE INTO memory_embedding_records (
				id, memory_index_id, memory_type, ref_table, ref_id, content_hash,
				model_name, vector_store_id, vector_dim, status, last_error, last_indexed_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', ?, ?, ?)
		`,
			"memb_"+entry.ID,
			entry.ID,
			entry.MemoryType,
			entry.RefTable,
			entry.RefID,
			"hash_"+entry.ID,
			"embed-test",
			entry.ID,
			3,
			domain.MemoryEmbeddingStatusIndexed,
			nowUTC(),
			nowUTC(),
			nowUTC(),
		); err != nil {
			t.Fatalf("insert memory_embedding_records %s: %v", entry.ID, err)
		}
		if _, err := store.db.Exec(`
			INSERT OR REPLACE INTO memory_embedding_jobs (
				id, memory_index_id, memory_type, ref_table, ref_id, status,
				attempt_count, error_message, claim_token, claim_expires_at,
				created_at, updated_at, started_at, finished_at
			) VALUES (?, ?, ?, ?, ?, ?, 0, '', '', '', ?, ?, '', '')
		`,
			"mjob_"+entry.ID,
			entry.ID,
			entry.MemoryType,
			entry.RefTable,
			entry.RefID,
			domain.MemoryEmbeddingJobStatusQueued,
			nowUTC(),
			nowUTC(),
		); err != nil {
			t.Fatalf("insert memory_embedding_jobs %s: %v", entry.ID, err)
		}
	}
}

func assertSessionDeleted(t *testing.T, store *Store, sessionID string) {
	t.Helper()

	saved, err := store.GetSession(context.Background(), sessionID)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if saved != nil {
		t.Fatalf("expected session %s to be deleted", sessionID)
	}
	assertRowCountEquals(t, store.db, "training_turns", "session_id = ?", 0, sessionID)
	assertRowCountEquals(t, store.db, "review_cards", "session_id = ?", 0, sessionID)
}

func assertRowCountEquals(
	t *testing.T,
	db *sql.DB,
	table string,
	where string,
	expected int,
	args ...any,
) {
	t.Helper()

	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	if strings.TrimSpace(where) != "" {
		query += " WHERE " + where
	}
	if err := db.QueryRow(query, args...).Scan(&count); err != nil {
		t.Fatalf("count rows in %s: %v", table, err)
	}
	if count != expected {
		t.Fatalf("expected %d rows in %s where %s, got %d", expected, table, where, count)
	}
}

func loadKnowledgeNodeIDByLabel(t *testing.T, db *sql.DB, label string) string {
	t.Helper()

	var id string
	if err := db.QueryRow(`
		SELECT id
		FROM knowledge_nodes
		WHERE label = ?
		ORDER BY updated_at DESC
		LIMIT 1
	`, label).Scan(&id); err != nil {
		t.Fatalf("load knowledge node by label %s: %v", label, err)
	}
	return id
}

func loadMemoryIndexIDByRef(t *testing.T, db *sql.DB, refTable, refID string) string {
	t.Helper()

	var id string
	if err := db.QueryRow(`
		SELECT id
		FROM memory_index
		WHERE ref_table = ? AND ref_id = ?
	`, refTable, refID).Scan(&id); err != nil {
		t.Fatalf("load memory index by ref %s/%s: %v", refTable, refID, err)
	}
	return id
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
